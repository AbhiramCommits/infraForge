package cli

import (
	"errors"
	"fmt"
)

// Process exit codes returned by infraforge. Every error path maps to one
// of these, documented in the README.
const (
	// ExitOK means the command succeeded.
	ExitOK = 0
	// ExitProvisionFailure means ansible or the provisioning machinery
	// failed (exec/parse errors, unreachable or failed hosts, missing
	// backend support).
	ExitProvisionFailure = 1
	// ExitConfigError means the config file or a command's configuring
	// flags are invalid.
	ExitConfigError = 2
	// ExitLimitViolation means a limited workload was OOM-killed or exited
	// non-zero.
	ExitLimitViolation = 3
)

// ExitError carries a process exit code along with the wrapped error.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }

// Unwrap returns the wrapped error.
func (e *ExitError) Unwrap() error { return e.Err }

// exitErrorf wraps a formatted error with an exit code.
func exitErrorf(code int, format string, args ...any) error {
	return &ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

// exitCode maps an error to a process exit code. Unclassified errors map to
// ExitProvisionFailure.
func exitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitProvisionFailure
}
