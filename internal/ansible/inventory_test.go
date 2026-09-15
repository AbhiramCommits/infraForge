package ansible

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/infraforge/infraforge/internal/config"
	"gopkg.in/yaml.v3"
)

func TestGenerateInventory(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{
			Name:    "web-01",
			Image:   "ubuntu:22.04",
			SSHPort: 2222,
			Limits:  config.Limits{CPUMax: 50, MemoryMax: "256M", PIDsMax: 128},
		},
		{
			Name:   "db-01",
			Image:  "debian:12",
			Limits: config.Limits{MemoryMax: "1G"},
		},
	}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("cfg.Validate() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "nested", "inventory.yml")
	if err := GenerateInventory(cfg, path); err != nil {
		t.Fatalf("GenerateInventory() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated inventory: %v", err)
	}
	var inv map[string]any
	if err := yaml.Unmarshal(data, &inv); err != nil {
		t.Fatalf("generated inventory is not valid YAML: %v", err)
	}

	all, ok := inv["all"].(map[string]any)
	if !ok {
		t.Fatalf("inventory missing top-level 'all' group: %v", inv)
	}
	hosts, ok := all["hosts"].(map[string]any)
	if !ok {
		t.Fatalf("inventory missing all.hosts: %v", all)
	}

	web, ok := hosts["web-01"].(map[string]any)
	if !ok {
		t.Fatalf("inventory missing host web-01: %v", hosts)
	}
	webChecks := map[string]string{
		"ansible_host":     "127.0.0.1",
		"ansible_port":     "2222",
		"ansible_user":     "infraforge",
		"ansible_password": "infraforge",
		"node_image":       "ubuntu:22.04",
		"node_cpu_max":     "50",
		"node_memory_max":  "256M",
		"node_pids_max":    "128",
	}
	for key, want := range webChecks {
		if got := fmt.Sprint(web[key]); got != want {
			t.Errorf("web-01[%q] = %q, want %q", key, got, want)
		}
	}
	if web["ansible_host_key_checking"] != false {
		t.Errorf("web-01 ansible_host_key_checking = %v, want false", web["ansible_host_key_checking"])
	}

	db, ok := hosts["db-01"].(map[string]any)
	if !ok {
		t.Fatalf("inventory missing host db-01: %v", hosts)
	}
	if got := fmt.Sprint(db["ansible_port"]); got != "22" {
		t.Errorf("db-01 ansible_port = %q, want default 22", got)
	}
	if got := fmt.Sprint(db["node_memory_max"]); got != "1G" {
		t.Errorf("db-01 node_memory_max = %q, want 1G", got)
	}

	children, ok := all["children"].(map[string]any)
	if !ok {
		t.Fatalf("inventory missing all.children: %v", all)
	}
	nodes, ok := children["nodes"].(map[string]any)
	if !ok {
		t.Fatalf("inventory missing nodes group: %v", children)
	}
	nodeHosts, ok := nodes["hosts"].(map[string]any)
	if !ok {
		t.Fatalf("nodes group missing hosts: %v", nodes)
	}
	for _, name := range []string{"web-01", "db-01"} {
		if _, ok := nodeHosts[name]; !ok {
			t.Errorf("nodes group missing %q: %v", name, nodeHosts)
		}
	}
}
