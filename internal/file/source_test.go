package file_test

import (
	"errors"
	"io/fs"
	"path"
	"testing"

	. "github.com/dhemery/duffel/internal/file"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestSourceDir(t *testing.T) {
	tests := []struct {
		desc       string        // Description of the test.
		files      []*errfs.File // Files on the file system.
		name       string        // The name of the directory whose source to seek.
		wantSource string        // Source result from SourceDir.
		wantErr    error         // The error result from SourceDir.
	}{
		{
			desc:       "name is source dir",
			files:      []*errfs.File{sourceDir("a/b/c/name")},
			name:       "a/b/c/name",
			wantSource: "a/b/c/name",
		},
		{
			desc: "name is inside source dir",
			files: []*errfs.File{
				sourceDir("a/b/c/source"),
				errfs.NewDir("a/b/c/source/d/e/f/name", 0o755),
			},
			name:       "a/b/c/source/d/e/f/name",
			wantSource: "a/b/c/source",
		},
		{
			desc:    "source dir is inside name",
			files:   []*errfs.File{sourceDir("a/b/c/name/d/e/f/source")},
			name:    "a/b/c/name",
			wantErr: fs.ErrNotExist,
		},
		{
			desc:    "no duffel file",
			files:   []*errfs.File{errfs.NewDir("a/b/c/dir", 0o755)},
			name:    "a/b/c/dir",
			wantErr: fs.ErrNotExist,
		},
		{
			desc:    "name is not a dir",
			files:   []*errfs.File{errfs.NewFile("a/b/c/file", 0o644)},
			name:    "a/b/c/file",
			wantErr: fs.ErrInvalid,
		},
		{
			desc:    "name does not exist",
			files:   []*errfs.File{},
			name:    "a/b/c/name",
			wantErr: fs.ErrNotExist,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			testfs := errfs.New()
			for _, file := range test.files {
				errfs.Add(testfs, file)
			}

			source, err := SourceDir(testfs, test.name)

			if source != test.wantSource {
				t.Errorf("SourceDir(%q) source:\n got: %v\nwant: %v",
					test.name, source, test.wantSource)
			}

			if !errors.Is(err, test.wantErr) {
				t.Errorf("SourceDir(%q) error:\n got: %v\nwant %v",
					test.name, err, test.wantErr)
			}
		})
	}
}

func sourceDir(dir string) *errfs.File {
	return errfs.NewFile(path.Join(dir, SourceMarkerFile), 0o644)
}
