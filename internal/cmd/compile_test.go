package cmd

import (
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		desc    string
		files   map[string]*errfs.File
		opts    Options
		args    []string
		wantErr error
	}{{
		desc: "absolute target",
	}}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Log(test.desc)
		})
	}
}

func TestFullValidPath(t *testing.T) {
	tests := []struct {
		desc     string
		cwd      string
		name     string
		wantPath string
	}{
		{
			desc:     "absolute name, absolute cwd",
			cwd:      "/abs/cwd",
			name:     "/abs/name",
			wantPath: "abs/name",
		},
		{
			desc:     "absolute name, relative cwd",
			cwd:      "rel/cwd",
			name:     "/abs/name",
			wantPath: "abs/name",
		},
		{
			desc:     "relative name, absolute cwd",
			cwd:      "/abs/cwd",
			name:     "rel/name",
			wantPath: "abs/cwd/rel/name",
		},
		{
			desc:     "relative name, relative cwd",
			cwd:      "rel/cwd",
			name:     "rel/name",
			wantPath: "rel/cwd/rel/name",
		},
		{
			desc:     "cleans cwd",
			cwd:      "/rel/a/b/c/../../../cwd",
			name:     "rel/name",
			wantPath: "rel/cwd/rel/name",
		},
		{
			desc:     "cleans name",
			cwd:      "rel/cwd",
			name:     "/rel/a/b/c/../../../name",
			wantPath: "rel/name",
		},
		{
			desc:     "cleans result",
			cwd:      "rel/a/b/cwd",
			name:     "../../../name",
			wantPath: "rel/name",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := fullValidPath(test.cwd, test.name)
			if got != test.wantPath {
				t.Errorf("got %s, want %s", got, test.wantPath)
			}
		})
	}
}
