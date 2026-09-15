package cgroup

import "testing"

func TestParseCPUPercent(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "half core", in: "50%", want: "50000 100000"},
		{name: "quarter core", in: "25%", want: "25000 100000"},
		{name: "fractional percent", in: "25.5%", want: "25500 100000"},
		{name: "full core", in: "100%", want: "100000 100000"},
		{name: "no percent sign", in: "10", want: "10000 100000"},
		{name: "with spaces", in: " 50% ", want: "50000 100000"},
		{name: "invalid letters", in: "abc", wantErr: true},
		{name: "negative", in: "-5%", wantErr: true},
		{name: "over one core", in: "150%", wantErr: true},
		{name: "empty", in: "", wantErr: true},
		{name: "percent sign only", in: "%", wantErr: true},
		{name: "rounds to zero", in: "0.0001%", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCPUPercent(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseCPUPercent(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCPUPercent(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseCPUPercent(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseKeyValue(t *testing.T) {
	input := []byte(`low 3
high 0
max 0
oom 0
oom_kill 42
oom_group_kill 0
`)
	got, err := parseKeyValue(input)
	if err != nil {
		t.Fatalf("parseKeyValue() error = %v", err)
	}
	want := map[string]int64{
		"low":            3,
		"high":           0,
		"max":            0,
		"oom":            0,
		"oom_kill":       42,
		"oom_group_kill": 0,
	}
	if len(got) != len(want) {
		t.Fatalf("parseKeyValue() = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseKeyValue()[%q] = %d, want %d", k, got[k], v)
		}
	}
}

func TestParseKeyValueErrors(t *testing.T) {
	for _, in := range []string{"high 1 2", "high", "high abc", "=1"} {
		if _, err := parseKeyValue([]byte(in)); err == nil {
			t.Errorf("parseKeyValue(%q) succeeded, want error", in)
		}
	}
}

func TestParseIntFile(t *testing.T) {
	got, err := parseIntFile([]byte(" 123456 \n"))
	if err != nil {
		t.Fatalf("parseIntFile() error = %v", err)
	}
	if got != 123456 {
		t.Errorf("parseIntFile() = %d, want 123456", got)
	}
	if _, err := parseIntFile([]byte("abc")); err == nil {
		t.Error("parseIntFile(\"abc\") succeeded, want error")
	}
}

func TestControllerPath(t *testing.T) {
	c := &Controller{}
	if got := c.Path("burner"); got != "/sys/fs/cgroup/infraforge/burner" {
		t.Errorf("Path() = %q, want default path", got)
	}
	c = &Controller{Root: "/tmp/fake", Group: "demo"}
	if got := c.Path("x"); got != "/tmp/fake/demo/x" {
		t.Errorf("Path() = %q, want /tmp/fake/demo/x", got)
	}
}
