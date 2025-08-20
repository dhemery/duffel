package cmd

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		desc    string        // Description of the test.
		files   []*errfs.File // Files on the file system.
		opts    Options       // Options passed to Compile.
		args    []string      // Args passed to Compile.
		wantErr error         // Error result from Compile.
		skip    string        // Reason to skip this test.
	}{
		{
			desc:    "target does not exist",
			files:   []*errfs.File{},
			opts:    Options{Target: "target"},
			wantErr: fs.ErrNotExist,
		},
		{
			desc:    "target is not a dir",
			files:   []*errfs.File{errfs.NewFile("target", 0644)},
			opts:    Options{Target: "target"},
			wantErr: fs.ErrInvalid,
		},
		{
			desc:    "source does not exist",
			files:   []*errfs.File{},
			opts:    Options{Source: "source"},
			wantErr: fs.ErrNotExist,
		},
		{
			desc:    "source is not a dir",
			files:   []*errfs.File{errfs.NewFile("source", 0644)},
			opts:    Options{Source: "source"},
			wantErr: fs.ErrInvalid,
		},
		{
			desc: "source is not a duffel source",
			files: []*errfs.File{
				errfs.NewDir("source", 0755),
			},
			opts:    Options{Source: "source"},
			wantErr: fs.ErrInvalid,
			skip:    "not yet implemented",
		},
		{
			desc:    "package does not exist",
			files:   []*errfs.File{errfs.NewDir("source", 0755)},
			opts:    Options{Source: "source"},
			args:    []string{"pkg"},
			wantErr: fs.ErrNotExist,
		},
		{
			desc:    "package is not a dir",
			files:   []*errfs.File{errfs.NewFile("source/pkg", 0644)},
			opts:    Options{Source: "source"},
			args:    []string{"pkg"},
			wantErr: fs.ErrInvalid,
		},
		{
			desc:    "empty package",
			files:   []*errfs.File{errfs.NewDir("source", 0755)},
			opts:    Options{Source: "source"},
			args:    []string{""},
			wantErr: fs.ErrInvalid,
		},
		{
			desc:    "package is .",
			files:   []*errfs.File{errfs.NewDir("source", 0755)},
			opts:    Options{Source: "source"},
			args:    []string{"."},
			wantErr: fs.ErrInvalid,
		},
		{
			desc: "package is not in source",
			files: []*errfs.File{
				errfs.NewDir("source", 0755),
				errfs.NewDir("pkg", 0755),
			},
			opts:    Options{Source: "source"},
			args:    []string{"../pkg"},
			wantErr: fs.ErrInvalid,
		},
		{
			desc:    "package is deeper than child",
			files:   []*errfs.File{errfs.NewDir("source/sub1/sub2", 0755)},
			opts:    Options{Source: "source"},
			args:    []string{"sub1/sub2"},
			wantErr: fs.ErrInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}

			testfs := errfs.New()
			for _, file := range test.files {
				errfs.Add(testfs, file)
			}

			_, err := Compile(test.opts, test.args, testfs, "/", nil, nil)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("error result:\n got: %v\nwant: %v", err, test.wantErr)
			}
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
