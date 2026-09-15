package cli

import (
	"io"
	"runtime"
	"strings"
	"testing"
)

func TestLimitInvalidFlagValues(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "invalid cpu percent", args: []string{"--cpu-max", "abc"}, wantErr: "invalid --cpu-max"},
		{name: "cpu percent out of range", args: []string{"--cpu-max", "150%"}, wantErr: "invalid --cpu-max"},
		{name: "invalid memory", args: []string{"--memory-max", "12XB"}, wantErr: "invalid --memory-max"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := &Options{Stderr: io.Discard}
			cmd := newRootCommand(opts)
			args := append([]string{"limit", "--name", "x"}, tc.args...)
			args = append(args, "--", "true")
			cmd.SetArgs(args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("Execute() succeeded, want error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
			}
			assertExitCode(t, err, ExitConfigError)
		})
	}
}

func TestLimitRequiresName(t *testing.T) {
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"limit", "--", "true"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error for missing --name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error = %q, want mention of name", err)
	}
	assertExitCode(t, err, ExitProvisionFailure)
}

func TestLimitRequiresLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("linux hosts can actually apply limits")
	}
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"limit", "--name", "x", "--", "true"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "requires Linux") {
		t.Errorf("error = %q, want clear platform message", err)
	}
	assertExitCode(t, err, ExitProvisionFailure)
}
