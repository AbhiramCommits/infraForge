// Package config loads and validates the infraforge configuration file.
package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultSSHPort is applied when a host does not set ssh_port.
const DefaultSSHPort = 22

// Limits describes the cgroup v2 resource limits applied to a host.
type Limits struct {
	// CPUMax is the CPU bandwidth limit as a percentage of one CPU core.
	CPUMax float64 `yaml:"cpu_max"`
	// MemoryMax is the memory limit as a byte string, e.g. "256M" or "2G".
	MemoryMax string `yaml:"memory_max"`
	// PIDsMax caps the number of tasks allowed in the cgroup.
	PIDsMax int64 `yaml:"pids_max"`
}

// Host is a single machine managed by infraforge.
type Host struct {
	Name    string `yaml:"name"`
	Image   string `yaml:"image"`
	SSHPort int    `yaml:"ssh_port"`
	Limits  Limits `yaml:"limits"`
}

// Config is the top-level infraforge configuration.
type Config struct {
	Hosts []Host `yaml:"hosts"`
}

// Load reads, parses, and validates the configuration file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	return Parse(data)
}

// Parse parses and validates YAML configuration data. Unknown fields are
// rejected so typos in the config file surface early.
func Parse(data []byte) (*Config, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks the configuration and applies defaults.
func (c *Config) Validate() error {
	seen := make(map[string]bool, len(c.Hosts))
	for i := range c.Hosts {
		h := &c.Hosts[i]
		if strings.TrimSpace(h.Name) == "" {
			return fmt.Errorf("hosts[%d]: name is required", i)
		}
		if seen[h.Name] {
			return fmt.Errorf("hosts[%d]: duplicate host name %q", i, h.Name)
		}
		seen[h.Name] = true
		if strings.TrimSpace(h.Image) == "" {
			return fmt.Errorf("host %q: image is required", h.Name)
		}
		switch {
		case h.SSHPort == 0:
			h.SSHPort = DefaultSSHPort
		case h.SSHPort < 1 || h.SSHPort > 65535:
			return fmt.Errorf("host %q: ssh_port %d is out of range [1, 65535]", h.Name, h.SSHPort)
		}
		if err := h.Limits.validate(); err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
	}
	return nil
}

func (l Limits) validate() error {
	if l.CPUMax < 0 || l.CPUMax > 100 {
		return fmt.Errorf("cpu_max must be a percentage between 0 and 100, got %v", l.CPUMax)
	}
	if l.MemoryMax != "" {
		if _, err := ParseMemory(l.MemoryMax); err != nil {
			return fmt.Errorf("invalid memory_max: %w", err)
		}
	}
	if l.PIDsMax < 0 {
		return fmt.Errorf("pids_max must not be negative, got %d", l.PIDsMax)
	}
	return nil
}
