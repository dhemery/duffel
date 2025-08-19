package cmd_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/dhemery/duffel/internal/cmd"
)

func TestOptions(t *testing.T) {
	tests := []struct {
		desc       string        // Description of the test.
		args       []string      // The args passed to ParseArgs.
		wantOpts   checkOptsFunc // Assertions for Options result.
		wantArgs   []string      // Args result.
		wantErr    error         // Error result.
		wantErrOut string        // Output written to error writer.
	}{
		{
			desc: "default",
			args: []string{},
			wantOpts: checkOpts(
				checkSource("."),
				checkTarget(".."),
				checkDryRun(false),
				checkLogLevel(slog.LevelError)),
		},
		{
			desc:     "source",
			args:     []string{"-source", "my-source"},
			wantOpts: checkSource("my-source"),
		},
		{
			desc:     "target",
			args:     []string{"-target", "my-target"},
			wantOpts: checkTarget("my-target"),
		},
		{
			desc:     "dry run",
			args:     []string{"-n"},
			wantOpts: checkDryRun(true),
		},
		{
			desc:     "log level none",
			args:     []string{"-log", "none"},
			wantOpts: checkLogLevel(slog.LevelError + 4),
		},
		{
			desc:     "log level error",
			args:     []string{"-log", "error"},
			wantOpts: checkLogLevel(slog.LevelError),
		},
		{
			desc:     "log level warn",
			args:     []string{"-log", "warn"},
			wantOpts: checkLogLevel(slog.LevelWarn),
		},
		{
			desc:     "log level info",
			args:     []string{"-log", "info"},
			wantOpts: checkLogLevel(slog.LevelInfo),
		},
		{
			desc:     "log level debug",
			args:     []string{"-log", "debug"},
			wantOpts: checkLogLevel(slog.LevelDebug),
		},
		{
			desc: "positional args",
			args: []string{"positional1", "positional2", "positional3"},
			wantOpts: checkOpts(
				checkSource("."),
				checkTarget(".."),
				checkDryRun(false),
				checkLogLevel(slog.LevelError)),
			wantArgs: []string{"positional1", "positional2", "positional3"},
		},
		{
			desc: "options and positional args",
			args: []string{"-n",
				"-log", "warn",
				"-target", "my-target",
				"-source", "my-source",
				"positional1", "positional2", "positional3",
			},
			wantOpts: checkOpts(
				checkDryRun(true),
				checkLogLevel(slog.LevelWarn),
				checkTarget("my-target"),
				checkSource("my-source"),
			),
			wantArgs: []string{"positional1", "positional2", "positional3"},
		},
		{
			desc:       "unknown log level",
			args:       []string{"-log", "bad-log-level"},
			wantErr:    cmpopts.AnyError,
			wantErrOut: "bad-log-level",
		},
		{
			desc:       "unknown option",
			args:       []string{"-bad-option"},
			wantErr:    cmpopts.AnyError,
			wantErrOut: "bad-option",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var werr bytes.Buffer
			opts, args, err := cmd.ParseArgs(test.args, &werr)

			if test.wantOpts != nil {
				test.wantOpts(t, opts)
			}

			if diff := cmp.Diff(test.wantArgs, args, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("args:\n%s", diff)
			}

			if diff := cmp.Diff(test.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("error:\n got: %v\nwant: %v", err, test.wantErr)
			}

			gotErrOut := werr.String()
			if !strings.Contains(gotErrOut, test.wantErrOut) {
				t.Errorf("error output:\n got: %s\nwant: %s", gotErrOut, test.wantErrOut)
			}
		})
	}
}

type checkOptsFunc func(t *testing.T, o cmd.Options)

func checkOpts(funcs ...checkOptsFunc) checkOptsFunc {
	return func(t *testing.T, o cmd.Options) {
		for _, f := range funcs {
			f(t, o)
		}
	}
}

func checkSource(want string) checkOptsFunc {
	return func(t *testing.T, o cmd.Options) {
		if o.Source != want {
			t.Errorf("source: got %s, want %s", o.Source, want)
		}
	}
}

func checkTarget(want string) checkOptsFunc {
	return func(t *testing.T, o cmd.Options) {
		if o.Target != want {
			t.Errorf("target: got %s, want %s", o.Target, want)
		}
	}
}

func checkDryRun(want bool) checkOptsFunc {
	return func(t *testing.T, o cmd.Options) {
		if o.DryRun != want {
			t.Errorf("target: got %t want %t", o.DryRun, want)
		}
	}
}

func checkLogLevel(want slog.Level) checkOptsFunc {
	return func(t *testing.T, o cmd.Options) {
		if o.LogLevel != want {
			t.Errorf("target: got %s, want %s", o.LogLevel, want)
		}
	}
}
