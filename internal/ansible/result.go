package ansible

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

// HostStats summarizes one host's ansible-playbook run.
type HostStats struct {
	// OK is the number of tasks that ran without changes.
	OK int `json:"ok"`
	// Changed is the number of tasks that reported changed: true.
	Changed int `json:"changed"`
	// Unreachable is the number of tasks skipped because the host could not
	// be contacted.
	Unreachable int `json:"unreachable"`
	// Failed is the number of tasks that failed on the host.
	Failed int `json:"failures"`
}

// Result is the parsed outcome of an ansible-playbook run.
type Result struct {
	// Hosts maps host names to their per-run statistics.
	Hosts map[string]HostStats
	// ChangedTasks holds the names of tasks that reported changed: true on
	// at least one host, in the order they were first seen.
	ChangedTasks []string
}

// Problems returns the number of hosts with failures and the number of
// unreachable hosts.
func (r *Result) Problems() (failed, unreachable int) {
	for _, s := range r.Hosts {
		if s.Failed > 0 {
			failed++
		}
		if s.Unreachable > 0 {
			unreachable++
		}
	}
	return failed, unreachable
}

// Validate returns an error if any host failed or was unreachable.
func (r *Result) Validate() error {
	failed, unreachable := r.Problems()
	if failed > 0 || unreachable > 0 {
		return fmt.Errorf("provisioning failed: %d host(s) with failures, %d host(s) unreachable", failed, unreachable)
	}
	return nil
}

// jsonCallbackDoc mirrors the structure emitted by the ansible "json"
// stdout callback plugin.
type jsonCallbackDoc struct {
	Plays []jsonPlay           `json:"plays"`
	Stats map[string]HostStats `json:"stats"`
}

type jsonPlay struct {
	Play struct {
		Name string `json:"name"`
	} `json:"play"`
	Tasks []jsonTask `json:"tasks"`
}

type jsonTask struct {
	Task struct {
		Name string `json:"name"`
	} `json:"task"`
	Hosts map[string]jsonTaskResult `json:"hosts"`
}

type jsonTaskResult struct {
	Changed bool   `json:"changed"`
	Failed  bool   `json:"failed"`
	Msg     string `json:"msg"`
}

// ParseJSON reads whitespace-delimited ansible JSON callback documents from r
// and folds them into a single Result, logging each task transition as the
// documents stream in. Later stats replace earlier ones: the callback emits
// cumulative statistics in its final document.
func ParseJSON(r io.Reader, logger *slog.Logger) (*Result, error) {
	res := &Result{Hosts: map[string]HostStats{}}
	dec := json.NewDecoder(r)
	for {
		var doc jsonCallbackDoc
		if err := dec.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode ansible output: %w", err)
		}
		res.merge(&doc, logger)
	}
	return res, nil
}

func (r *Result) merge(doc *jsonCallbackDoc, logger *slog.Logger) {
	if len(doc.Stats) > 0 {
		r.Hosts = doc.Stats
	}
	for _, play := range doc.Plays {
		for _, task := range play.Tasks {
			changed := 0
			for _, hostRes := range task.Hosts {
				if hostRes.Changed {
					changed++
				}
			}
			if changed > 0 {
				r.addChangedTask(task.Task.Name)
			}
			logger.Info("task finished",
				"task", task.Task.Name,
				"play", play.Play.Name,
				"hosts", len(task.Hosts),
				"changed_hosts", changed,
			)
		}
	}
}

func (r *Result) addChangedTask(name string) {
	for _, existing := range r.ChangedTasks {
		if existing == name {
			return
		}
	}
	r.ChangedTasks = append(r.ChangedTasks, name)
}
