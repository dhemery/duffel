package plan_test

import (
	"bytes"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	. "github.com/dhemery/duffel/internal/plan"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMerge(t *testing.T) {
	tests := map[string]struct {
		mergeDir   string                // The name of the directory to merge.
		target     string                // The target to merge into.
		files      []*errfs.File         // Other files on the file system.
		wantErr    error                 // Error returned by Merge.
		wantStates map[string]file.State // States added to index during Merge.
	}{
		"not in a package": {
			mergeDir: "dir1/dir2/dir3/dir4/dir5/dir6",
			files:    []*errfs.File{}, // No other files, so no .duffel file
			wantErr:  &MergeError{Name: "dir1/dir2/dir3/dir4/dir5/dir6", Err: ErrNotInPackage},
		},
		"duffel source dir": {
			mergeDir: "duffel/source-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
			},
			wantErr: &MergeError{Name: "duffel/source-dir", Err: ErrIsSource},
		},
		"duffel package": {
			mergeDir: "duffel/source-dir/pkg-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
			},
			wantErr: &MergeError{Name: "duffel/source-dir/pkg-dir", Err: ErrIsPackage},
		},
		"top level item in a package": {
			mergeDir: "duffel/source-dir/pkg-dir/item",
			target:   "target-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
				errfs.NewFile("duffel/source-dir/pkg-dir/item/content", 0o644),
			},
			wantStates: map[string]file.State{
				"target-dir/item/content": file.LinkState(
					"../../duffel/source-dir/pkg-dir/item/content",
					file.TypeFile),
			},
			wantErr: nil,
		},
		"nested item in a package": {
			mergeDir: "duffel/source-dir/pkg-dir/item1/item2/item3",
			target:   "target-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
				errfs.NewFile("duffel/source-dir/pkg-dir/item1/item2/item3/content", 0o644),
			},
			wantStates: map[string]file.State{
				"target-dir/item1/item2/item3/content": file.LinkState(
					"../../../../duffel/source-dir/pkg-dir/item1/item2/item3/content",
					file.TypeFile),
			},
			wantErr: nil,
		},
		"various file types in a package": {
			mergeDir: "duffel/source-dir/pkg-dir/item",
			target:   "target-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
				errfs.NewDir("duffel/source-dir/pkg-dir/item/dir", 0o755),
				errfs.NewFile("duffel/source-dir/pkg-dir/item/file", 0o644),
				errfs.NewLink("duffel/source-dir/pkg-dir/item/link", "some/dest"),
			},
			wantStates: map[string]file.State{
				"target-dir/item/dir": file.LinkState(
					"../../duffel/source-dir/pkg-dir/item/dir",
					file.TypeDir),
				"target-dir/item/file": file.LinkState(
					"../../duffel/source-dir/pkg-dir/item/file",
					file.TypeFile),
				"target-dir/item/link": file.LinkState(
					"../../duffel/source-dir/pkg-dir/item/link",
					file.TypeSymlink),
			},
			wantErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var logbuf bytes.Buffer
			logger := log.Logger(&logbuf, duftest.LogLevel)

			testFS := errfs.New()
			errfs.AddDir(testFS, test.mergeDir, 0o755)
			for _, tf := range test.files {
				errfs.Add(testFS, tf)
			}

			stater := file.NewStater(testFS)
			index := NewIndex(stater)
			analyzer := NewAnalyst(testFS, test.target, index)
			itemizer := NewItemizer(testFS)

			merger := NewMerger(itemizer, analyzer)

			err := merger.Merge(test.mergeDir, logger)

			if diff := cmp.Diff(test.wantErr, err); diff != "" {
				t.Errorf("Merge(%q, %q) error:\n%s",
					test.mergeDir, test.target, diff)
			}

			gotStates := map[string]file.State{}
			for n, spec := range index.All() {
				gotStates[n] = spec.Planned
			}
			if diff := cmp.Diff(test.wantStates, gotStates, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("planned states after Merge(%q, %q):\n%s",
					test.mergeDir, test.target, diff)
			}
			if t.Failed() || testing.Verbose() {
				t.Log("files:\n", testFS)
				t.Log("log:\n", logbuf.String())
			}
		})
	}
}
