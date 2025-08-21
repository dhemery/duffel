package plan_test

import (
	"bytes"
	"path"
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
		target     string                // The target to merge into.
		files      []*errfs.File         // Files on the file system in addition to the merge dir.
		nameArg    string                // The name of the directory to merge.
		wantErr    error                 // Error returned by Merge.
		wantStates map[string]file.State // States added to index during Merge.
	}{
		"not in a package": {
			files:   []*errfs.File{}, // No other files, so no source marker file
			nameArg: "dir1/dir2/dir3/dir4/dir5/dir6",
			wantErr: &MergeError{Name: "dir1/dir2/dir3/dir4/dir5/dir6", Err: ErrNotInPackage},
		},
		"duffel source dir": {
			files: []*errfs.File{
				sourceDir("duffel/source-dir"),
			},
			nameArg: "duffel/source-dir",
			wantErr: &MergeError{Name: "duffel/source-dir", Err: ErrIsSource},
		},
		"duffel package": {
			files: []*errfs.File{
				sourceDir("duffel/source-dir"),
			},
			nameArg: "duffel/source-dir/pkg-dir",
			wantErr: &MergeError{Name: "duffel/source-dir/pkg-dir", Err: ErrIsPackage},
		},
		"top level item in a package": {
			target: "target-dir",
			files: []*errfs.File{
				sourceDir("duffel/source-dir"),
				errfs.NewFile("duffel/source-dir/pkg-dir/item/content", 0o644),
			},
			nameArg: "duffel/source-dir/pkg-dir/item",
			wantStates: map[string]file.State{
				"target-dir/item/content": file.LinkState(
					"../../duffel/source-dir/pkg-dir/item/content",
					file.TypeFile),
			},
			wantErr: nil,
		},
		"nested item in a package": {
			target: "target-dir",
			files: []*errfs.File{
				sourceDir("duffel/source-dir"),
				errfs.NewFile("duffel/source-dir/pkg-dir/item1/item2/item3/content", 0o644),
			},
			nameArg: "duffel/source-dir/pkg-dir/item1/item2/item3",
			wantStates: map[string]file.State{
				"target-dir/item1/item2/item3/content": file.LinkState(
					"../../../../duffel/source-dir/pkg-dir/item1/item2/item3/content",
					file.TypeFile),
			},
			wantErr: nil,
		},
		"various file types in a package": {
			target: "target-dir",
			files: []*errfs.File{
				sourceDir("duffel/source-dir"),
				errfs.NewDir("duffel/source-dir/pkg-dir/item/dir", 0o755),
				errfs.NewFile("duffel/source-dir/pkg-dir/item/file", 0o644),
				errfs.NewLink("duffel/source-dir/pkg-dir/item/link", "some/dest"),
			},
			nameArg: "duffel/source-dir/pkg-dir/item",
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
			errfs.AddDir(testFS, test.nameArg, 0o755)
			for _, tf := range test.files {
				errfs.Add(testFS, tf)
			}

			stater := file.NewStater(testFS)
			index := NewIndex(stater)
			analyzer := NewAnalyzer(testFS, test.target, index)
			itemizer := NewItemizer(testFS)

			merger := NewMerger(itemizer, analyzer)

			err := merger.Merge(test.nameArg, logger)

			if diff := cmp.Diff(test.wantErr, err); diff != "" {
				t.Errorf("Merge(%q, %q) error:\n%s",
					test.nameArg, test.target, diff)
			}

			gotStates := map[string]file.State{}
			for n, spec := range index.All() {
				gotStates[n] = spec.Planned
			}
			if diff := cmp.Diff(test.wantStates, gotStates, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("planned states after Merge(%q, %q):\n%s",
					test.nameArg, test.target, diff)
			}
			if t.Failed() || testing.Verbose() {
				t.Log("files:\n", testFS)
				t.Log("log:\n", logbuf.String())
			}
		})
	}
}

// SourceDir returns a source marker file in dir.
func sourceDir(dir string) *errfs.File {
	return errfs.NewFile(path.Join(dir, file.SourceMarkerFile), 0o644)
}
