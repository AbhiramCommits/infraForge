package cli

import (
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// initializeRun validates flag values and sets up the logger for this
// invocation. It runs once, before any subcommand executes.
func initializeRun(opts *Options) error {
	if opts.LogFormat != "text" && opts.LogFormat != "json" {
		return fmt.Errorf("invalid --log-format %q: must be \"text\" or \"json\"", opts.LogFormat)
	}
	runID, err := newRunID()
	if err != nil {
		return fmt.Errorf("generate run id: %w", err)
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	opts.RunID = runID
	opts.Logger = newLogger(opts.LogFormat, runID, opts.Stderr)
	opts.Logger.Debug("run initialized",
		"inventory", opts.Inventory,
		"config", opts.Config,
		"dry_run", opts.DryRun,
	)
	return nil
}

// newLogger builds a slog logger whose handler format matches the given
// format. Every line carries the run ID.
func newLogger(format, runID string, w io.Writer) *slog.Logger {
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, nil)
	} else {
		handler = slog.NewTextHandler(w, nil)
	}
	return slog.New(handler).With(slog.String("run_id", runID))
}

// newRunID returns a random UUID v4 string.
func newRunID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
