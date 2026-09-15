package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		missing bool
		wantErr string
		check   func(*testing.T, *Config)
	}{
		{
			name: "valid config",
			yaml: `
hosts:
  - name: web-01
    image: ubuntu-24.04
    ssh_port: 2222
    limits:
      cpu_max: 50
      memory_max: 256M
      pids_max: 128
`,
			check: func(t *testing.T, cfg *Config) {
				if len(cfg.Hosts) != 1 {
					t.Fatalf("got %d hosts, want 1", len(cfg.Hosts))
				}
				h := cfg.Hosts[0]
				if h.Name != "web-01" || h.Image != "ubuntu-24.04" || h.SSHPort != 2222 {
					t.Errorf("unexpected host: %+v", h)
				}
				if h.Limits.CPUMax != 50 || h.Limits.MemoryMax != "256M" || h.Limits.PIDsMax != 128 {
					t.Errorf("unexpected limits: %+v", h.Limits)
				}
			},
		},
		{
			name: "empty host list is valid",
			yaml: "hosts: []\n",
			check: func(t *testing.T, cfg *Config) {
				if len(cfg.Hosts) != 0 {
					t.Fatalf("got %d hosts, want 0", len(cfg.Hosts))
				}
			},
		},
		{
			name: "ssh port defaults to 22",
			yaml: `
hosts:
  - name: db-01
    image: debian-12
`,
			check: func(t *testing.T, cfg *Config) {
				if got := cfg.Hosts[0].SSHPort; got != DefaultSSHPort {
					t.Errorf("SSHPort = %d, want %d", got, DefaultSSHPort)
				}
			},
		},
		{
			name:    "invalid memory: not a number",
			yaml:    hostWithMemory("abc"),
			wantErr: "memory_max",
		},
		{
			name:    "invalid memory: unknown suffix",
			yaml:    hostWithMemory("256XB"),
			wantErr: "memory_max",
		},
		{
			name:    "invalid memory: negative",
			yaml:    hostWithMemory("-4G"),
			wantErr: "memory_max",
		},
		{
			name:    "invalid memory: fractional",
			yaml:    hostWithMemory("1.5T"),
			wantErr: "memory_max",
		},
		{
			name:    "invalid memory: suffix only",
			yaml:    hostWithMemory("M"),
			wantErr: "memory_max",
		},
		{
			name: "cpu max above 100",
			yaml: `
hosts:
  - name: h1
    image: img
    limits:
      cpu_max: 150
`,
			wantErr: "cpu_max",
		},
		{
			name: "negative pids max",
			yaml: `
hosts:
  - name: h1
    image: img
    limits:
      pids_max: -1
`,
			wantErr: "pids_max",
		},
		{
			name: "ssh port out of range",
			yaml: `
hosts:
  - name: h1
    image: img
    ssh_port: 70000
`,
			wantErr: "ssh_port",
		},
		{
			name: "missing host name",
			yaml: `
hosts:
  - image: img
`,
			wantErr: "name is required",
		},
		{
			name: "missing image",
			yaml: `
hosts:
  - name: h1
`,
			wantErr: "image is required",
		},
		{
			name: "duplicate host names",
			yaml: `
hosts:
  - name: h1
    image: img
  - name: h1
    image: img
`,
			wantErr: "duplicate host name",
		},
		{
			name: "unknown field",
			yaml: `
hosts:
  - name: h1
    image: img
    port: 22
`,
			wantErr: "field port not found",
		},
		{
			name:    "malformed yaml",
			yaml:    "hosts: [\n",
			wantErr: "parse config",
		},
		{
			name:    "missing file",
			missing: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			if !tc.missing {
				if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := Load(path)

			if tc.missing {
				if err == nil {
					t.Fatal("Load() succeeded, want error for missing file")
				}
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("Load() error = %v, want fs.ErrNotExist", err)
				}
				return
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Load() succeeded, want error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Load() error = %q, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, cfg)
			}
		})
	}
}

func TestLoadGolden(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "testdata", "config", "valid.yml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Hosts) != 2 {
		t.Fatalf("got %d hosts, want 2", len(cfg.Hosts))
	}

	db := cfg.Hosts[0]
	if db.Name != "db-01" || db.Image != "debian-12" || db.SSHPort != 2222 {
		t.Errorf("unexpected db host: %+v", db)
	}
	if db.Limits.CPUMax != 25.5 || db.Limits.PIDsMax != 64 {
		t.Errorf("unexpected db limits: %+v", db.Limits)
	}
	if want := int64(512 << 20); mustMemory(t, db.Limits.MemoryMax) != want {
		t.Errorf("db memory = %d, want %d", mustMemory(t, db.Limits.MemoryMax), want)
	}

	web := cfg.Hosts[1]
	if web.Name != "web-01" || web.Image != "ubuntu-24.04" {
		t.Errorf("unexpected web host: %+v", web)
	}
	if web.SSHPort != DefaultSSHPort {
		t.Errorf("web SSHPort = %d, want default %d", web.SSHPort, DefaultSSHPort)
	}
	if web.Limits.CPUMax != 0 || web.Limits.PIDsMax != 0 {
		t.Errorf("unexpected web limits: %+v", web.Limits)
	}
	if want := int64(1 << 30); mustMemory(t, web.Limits.MemoryMax) != want {
		t.Errorf("web memory = %d, want %d", mustMemory(t, web.Limits.MemoryMax), want)
	}
}

func hostWithMemory(memory string) string {
	return "hosts:\n  - name: h1\n    image: img\n    limits:\n      memory_max: " + memory + "\n"
}

func mustMemory(t *testing.T, s string) int64 {
	t.Helper()
	n, err := ParseMemory(s)
	if err != nil {
		t.Fatalf("ParseMemory(%q) error = %v", s, err)
	}
	return n
}
