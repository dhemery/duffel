package file

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/errfs"
)

func TestActions(t *testing.T) {
	tests := []struct {
		desc      string
		files     []*errfs.File
		name      string
		action    Action
		wantFiles []*errfs.File
		wantErr   error
	}{
		{
			desc:    "mkdir",
			files:   []*errfs.File{errfs.NewDir("parent", 0o755)},
			name:    "parent/dir",
			action:  MkdirAction(),
			wantErr: nil,
		},
		{
			desc:    "mkdir error",
			files:   []*errfs.File{errfs.NewDir("unmodifiable-dir", 0o755, errfs.ErrWrite)},
			name:    "unmodifiable-dir/dir",
			action:  MkdirAction(),
			wantErr: errfs.ErrWrite,
		},
		{
			desc:    "remove file",
			files:   []*errfs.File{errfs.NewDir("parent/file", 0o644)},
			name:    "parent/file",
			action:  RemoveAction(),
			wantErr: nil,
		},
		{
			desc: "remove error",
			files: []*errfs.File{
				errfs.NewDir("unmodifiable-dir", 0o755, errfs.ErrWrite),
				errfs.NewFile("unmodifiable-dir/file", 0o644),
			},
			name:    "unmodifiable-dir/file",
			action:  RemoveAction(),
			wantErr: errfs.ErrWrite,
		},
		{
			desc:    "symlink",
			files:   []*errfs.File{errfs.NewDir("parent", 0o755)},
			name:    "parent/symlink",
			action:  SymlinkAction("some/dest"),
			wantErr: nil,
		},
		{
			desc:    "symlink error",
			files:   []*errfs.File{errfs.NewDir("unmodifiable-dir", 0o755, errfs.ErrWrite)},
			name:    "unmodifiable-dir/symlink",
			action:  SymlinkAction("some/dest"),
			wantErr: errfs.ErrWrite,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			testfs := errfs.New()
			for _, f := range test.files {
				errfs.Add(testfs, f)
			}
			defer duftest.Dump(t, "files", testfs)

			err := test.action.Execute(testfs, test.name)

			if diff := cmp.Diff(test.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("error:\n got: %v\nwant: %v", err, test.wantErr)
			}
		})
	}
}
