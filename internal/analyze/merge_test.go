package analyze_test

import (
	"bytes"
	"io/fs"
	"log/slog"
	"testing"

	. "github.com/dhemery/duffel/internal/analyze"

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
			var logbuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logbuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

			testFS := errfs.New()
			errfs.AddDir(testFS, test.mergeDir, 0o755)
			for _, tf := range test.files {
				errfs.Add(testFS, tf)
			}

			stater := file.NewStater(testFS)
			index := NewIndex(stater, logger)
			analyzer := NewAnalyst(testFS, test.target, index)
			pkgFinder := NewPkgFinder(testFS)

			merger := NewMerger(pkgFinder, analyzer)

			err := merger.Merge(test.mergeDir, test.target)

			if diff := cmp.Diff(test.wantErr, err, equateErrFields()); diff != "" {
				t.Errorf("Merge(%q, %q) error:\n%s",
					test.mergeDir, test.target, diff)
			}

			gotStates := map[string]*file.State{}
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

func isErrField() func(cmp.Path) bool {
	return func(p cmp.Path) bool {
		last := p.Last()
		sf, ok := last.(cmp.StructField)
		if !ok {
			return false
		}
		return sf.Name() == "Err"
	}
}

func equateErrFields() cmp.Option {
	return cmp.FilterPath(isErrField(), cmpopts.EquateErrors())
}
