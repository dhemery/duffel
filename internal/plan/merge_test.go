package plan

import (
	"errors"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
)

func TestMerge(t *testing.T) {
	tests := map[string]struct {
		mergeDir  string                 // The name of the directory to merge.
		target    string                 // The target to merge into.
		files     []testFile             // Other files on the file system.
		wantErr   error                  // Error returned by Merge.
		wantIndex map[string]*file.State // States added to index during Merge.
	}{
		"not in a package": {
			mergeDir: "dir1/dir2/dir3/dir4/dir5/dir6",
			files:    []testFile{}, // No other files, so no .duffel file
			wantErr:  file.ErrNotInPackage,
		},
		"duffel source dir": {
			mergeDir: "duffel/source-dir",
			files: []testFile{{
				name: "duffel/source-dir/.duffel",
				mode: 0o644,
			}},
			wantErr: file.ErrIsSource,
		},
		"duffel package": {
			mergeDir: "duffel/source-dir/pkg-dir",
			files: []testFile{{
				name: "duffel/source-dir/.duffel",
				mode: 0o644,
			}},
			wantErr: file.ErrIsPackage,
		},
		"top level item in a package": {
			mergeDir: "duffel/source-dir/pkg-dir/item",
			files: []testFile{{
				name: "duffel/source-dir/.duffel",
				mode: 0o644,
			}, {
				name: "duffel/source-dir/pkg-dir/item/content",
				mode: 0o644,
			}},
			wantErr: nil,
		},
		"nested item in a package": {
			mergeDir: "duffel/source-dir/pkg-dir/item1/item2/item3",
			files: []testFile{{
				name: "duffel/source-dir/.duffel",
				mode: 0o644,
			}, {
				name: "duffel/source-dir/pkg-dir/item1/item2/item3/content",
				mode: 0o644,
			}},
			wantErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			testFS.AddDir(test.mergeDir, 0o755)
			for _, tf := range test.files {
				testFS.Add(tf.name, tf.mode, "")
			}

			stater := file.Stater{FS: testFS}
			index := NewIndex(stater)
			analyzer := NewAnalyst(testFS, index)
			pkgFinder := file.NewPkgFinder(testFS)

			merger := NewMerger(pkgFinder, analyzer)

			err := merger.Merge(test.mergeDir, test.target)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("Merge() error:\n got: %v\nwant: %v", err, test.wantErr)
			}
		})
	}
}
