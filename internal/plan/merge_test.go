package plan

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMerge(t *testing.T) {
	tests := map[string]struct {
		mergeDir   string                 // The name of the directory to merge.
		target     string                 // The target to merge into.
		files      []*errfs.File          // Other files on the file system.
		wantErr    error                  // Error returned by Merge.
		wantStates map[string]*file.State // States added to index during Merge.
	}{
		"not in a package": {
			mergeDir: "dir1/dir2/dir3/dir4/dir5/dir6",
			files:    []*errfs.File{}, // No other files, so no .duffel file
			wantErr:  file.ErrNotInPackage,
		},
		"duffel source dir": {
			mergeDir: "duffel/source-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
			},
			wantErr: file.ErrIsSource,
		},
		"duffel package": {
			mergeDir: "duffel/source-dir/pkg-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
			},
			wantErr: file.ErrIsPackage,
		},
		"top level item in a package": {
			mergeDir: "duffel/source-dir/pkg-dir/item",
			target:   "target-dir",
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
				errfs.NewFile("duffel/source-dir/pkg-dir/item/content", 0o644),
			},
			wantStates: map[string]*file.State{
				"target-dir/item/content": {
					Type: fs.ModeSymlink,
					Dest: "../../duffel/source-dir/pkg-dir/item/content",
				},
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
			wantStates: map[string]*file.State{
				"target-dir/item1/item2/item3/content": {
					Type: fs.ModeSymlink,
					Dest: "../../../../duffel/source-dir/pkg-dir/item1/item2/item3/content",
				},
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
			wantStates: map[string]*file.State{
				"target-dir/item/dir": {
					Type:     fs.ModeSymlink,
					Dest:     "../../duffel/source-dir/pkg-dir/item/dir",
					DestType: fs.ModeDir,
				},
				"target-dir/item/file": {
					Type:     fs.ModeSymlink,
					Dest:     "../../duffel/source-dir/pkg-dir/item/file",
					DestType: 0,
				},
				"target-dir/item/link": {
					Type:     fs.ModeSymlink,
					Dest:     "../../duffel/source-dir/pkg-dir/item/link",
					DestType: fs.ModeSymlink,
				},
			},
			wantErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			errfs.AddDir(testFS, test.mergeDir, 0o755)
			for _, tf := range test.files {
				errfs.Add(testFS, tf)
			}

			stater := file.Stater{FS: testFS}
			index := NewIndex(stater)
			analyzer := NewAnalyst(testFS, index)
			pkgFinder := file.NewPkgFinder(testFS)

			merger := NewMerger(pkgFinder, analyzer)

			err := merger.Merge(test.mergeDir, test.target)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("Merge(%q, %q) error:\n got: %v\nwant: %v",
					test.mergeDir, test.target, err, test.wantErr)
			}

			gotStates := map[string]*file.State{}
			for n, spec := range index.Specs() {
				gotStates[n] = spec.Planned
			}
			indexDiff := cmp.Diff(test.wantStates, gotStates, cmpopts.EquateEmpty())
			if indexDiff != "" {
				t.Errorf("planned states after Merge(%q, %q):\n%s",
					test.mergeDir, test.target, indexDiff)
			}
			if t.Failed() {
				t.Logf("files after Merge(%q, %q):\n%s",
					test.mergeDir, test.target, testFS)
			}
		})
	}
}
