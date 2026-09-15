package ansible

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infraforge/infraforge/internal/config"
	"gopkg.in/yaml.v3"
)

// DefaultAnsibleUser is the service user the generated inventory connects as.
const DefaultAnsibleUser = "infraforge"

// DefaultAnsiblePassword is the dev-only password baked into the node image.
const DefaultAnsiblePassword = "infraforge"

type inventoryFile struct {
	All inventoryGroup `yaml:"all"`
}

type inventoryGroup struct {
	Hosts    map[string]inventoryHost  `yaml:"hosts"`
	Children map[string]inventoryChild `yaml:"children"`
}

type inventoryChild struct {
	Hosts map[string]struct{} `yaml:"hosts"`
}

type inventoryHost struct {
	AnsibleHost         string  `yaml:"ansible_host"`
	AnsiblePort         int     `yaml:"ansible_port"`
	AnsibleUser         string  `yaml:"ansible_user"`
	AnsiblePassword     string  `yaml:"ansible_password"`
	AnsibleHostKeyCheck bool    `yaml:"ansible_host_key_checking"`
	NodeImage           string  `yaml:"node_image"`
	NodeCPUMax          float64 `yaml:"node_cpu_max"`
	NodeMemoryMax       string  `yaml:"node_memory_max"`
	NodePIDsMax         int64   `yaml:"node_pids_max"`
}

// GenerateInventory writes an Ansible inventory derived from cfg to path.
// The file is fully generated and must not be edited by hand.
func GenerateInventory(cfg *config.Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create inventory directory: %w", err)
	}

	inv := inventoryFile{
		All: inventoryGroup{
			Hosts: make(map[string]inventoryHost, len(cfg.Hosts)),
			Children: map[string]inventoryChild{
				"nodes": {Hosts: make(map[string]struct{}, len(cfg.Hosts))},
			},
		},
	}
	for _, h := range cfg.Hosts {
		inv.All.Hosts[h.Name] = inventoryHost{
			AnsibleHost:         "127.0.0.1",
			AnsiblePort:         h.SSHPort,
			AnsibleUser:         DefaultAnsibleUser,
			AnsiblePassword:     DefaultAnsiblePassword,
			AnsibleHostKeyCheck: false,
			NodeImage:           h.Image,
			NodeCPUMax:          h.Limits.CPUMax,
			NodeMemoryMax:       h.Limits.MemoryMax,
			NodePIDsMax:         h.Limits.PIDsMax,
		}
		inv.All.Children["nodes"].Hosts[h.Name] = struct{}{}
	}

	data, err := yaml.Marshal(inv)
	if err != nil {
		return fmt.Errorf("marshal inventory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write inventory %q: %w", path, err)
	}
	return nil
}
