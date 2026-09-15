// Package ansible drives Ansible playbooks used to provision hosts.
package ansible

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
)

// Runner executes ansible-playbook against an inventory.
type Runner struct {
	// Playbook is the path to the site playbook to run.
	Playbook string
	// Inventory is the path to the generated inventory file.
	Inventory string
	// DryRun passes --check --diff to ansible-playbook.
	DryRun bool
	// Logger receives one line per completed task as the run streams.
	Logger *slog.Logger
}

// Run executes ansible-playbook with the JSON stdout callback, streaming the
// per-task results into a Result. It returns a non-nil error when the
// playbook cannot be run, produces unparsable output, or leaves any host
// failed or unreachable.
func (r *Runner) Run(ctx context.Context) (*Result, error) {
	logger := r.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	cmd := exec.CommandContext(ctx, "ansible-playbook", r.buildArgs()...)
	cmd.Env = append(os.Environ(),
		"ANSIBLE_STDOUT_CALLBACK=json",
		"ANSIBLE_HOST_KEY_CHECKING=False",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	logger.Info("running ansible-playbook", "args", r.buildArgs(), "dry_run", r.DryRun)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ansible-playbook: %w", err)
	}

	result, parseErr := ParseJSON(stdout, logger)
	waitErr := cmd.Wait()
	switch {
	case parseErr != nil:
		return nil, fmt.Errorf("parse ansible-playbook output: %w (stderr: %s)", parseErr, stderr.String())
	case waitErr != nil:
		return nil, fmt.Errorf("ansible-playbook failed: %w (stderr: %s)", waitErr, stderr.String())
	}
	if err := result.Validate(); err != nil {
		return result, err
	}
	return result, nil
}

// buildArgs assembles the ansible-playbook argument list.
func (r *Runner) buildArgs() []string {
	args := []string{"-i", r.Inventory}
	if r.DryRun {
		args = append(args, "--check", "--diff")
	}
	return append(args, r.Playbook)
}
