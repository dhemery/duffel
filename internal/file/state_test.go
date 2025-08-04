package file_test

import (
	"errors"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
	. "github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"

	"github.com/dhemery/duffel/internal/errfs"
)

func add(tfs *errfs.FS, f *errfs.File) {
	if f == nil {
		return
	}
	errfs.Add(tfs, f)
}

func TestStater(t *testing.T) {
	tests := map[string]struct {
		name      string
		file      *errfs.File
		destFile  *errfs.File
		wantState State
		wantError error
	}{
		"file": {
			name:      "dir/file",
			file:      errfs.NewFile("dir/file", 0o644),
			wantState: FileState(),
		},
		"dir": {
			name:      "dir/dir",
			file:      errfs.NewDir("dir/dir", 0o755),
			wantState: DirState(),
		},
		"link": {
			name:      "dir/link",
			file:      errfs.NewLink("dir/link", "../dest-dir/dest-file"),
			destFile:  errfs.NewFile("dest-dir/dest-file", 0o644),
			wantState: LinkState("../dest-dir/dest-file", TypeFile),
		},
		"file lstat error": {
			name:      "dir/file",
			file:      errfs.NewFile("dir/file", 0o644, errfs.ErrLstat),
			wantError: errfs.ErrLstat,
		},
		"file readlink error": {
			name:      "dir/link",
			file:      errfs.NewLink("dir/link", "bad/dest", errfs.ErrReadLink),
			wantError: errfs.ErrReadLink,
		},
		"dest lstat error": {
			name:      "dir/link",
			file:      errfs.NewLink("dir/link", "../dest-dir/dest-file"),
			destFile:  errfs.NewFile("dest-dir/dest-file", 0o644, errfs.ErrLstat),
			wantError: errfs.ErrLstat,
		},
		"no file": {
			name:      "missing/file",
			file:      nil,
			wantState: State{},
			wantError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			defer duftest.Dump(t, "files", testFS)

			add(testFS, test.file)
			add(testFS, test.destFile)

			stater := NewStater(testFS)

			state, err := stater.State(test.name)

			if !errors.Is(err, test.wantError) {
				t.Errorf("State(%s) error:\n got %v\nwant %v",
					test.name, err, test.wantError)
			}
			if diff := cmp.Diff(test.wantState, state); diff != "" {
				t.Errorf("State(%s) state:\n%s", test.name, diff)
			}
		})
	}
}
