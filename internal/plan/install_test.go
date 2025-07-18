package plan

import (
	"errors"
	"io/fs"
	"maps"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type testFile struct {
	name string
	mode fs.FileMode
	dest string
}

func (tf testFile) Info() (fs.FileInfo, error) {
	return nil, nil
}

func (tf testFile) IsDir() bool {
	return tf.mode.IsDir()
}

func (tf testFile) Name() string {
	return tf.name
}

func (tf testFile) Type() fs.FileMode {
	return tf.mode.Type()
}

func dirFile(name string) testFile {
	return testFile{name: name, mode: fs.ModeDir | 0o755}
}

func regularFile(name string) testFile {
	return testFile{name: name, mode: 0o644}
}

func dirState() *file.State {
	return modeState(fs.ModeDir | 0o755)
}

func fileState() *file.State {
	return modeState(0o644)
}

func linkState(dest string, destMode fs.FileMode) *file.State {
	return &file.State{Mode: fs.ModeSymlink, Dest: dest, DestMode: destMode}
}

func modeState(mode fs.FileMode) *file.State {
	return &file.State{Mode: mode}
}

func TestInstallOp(t *testing.T) {
	tests := map[string]struct {
		source      string
		target      string
		itemFile    testFile
		targetState *file.State // Target state passed to Apply
		wantState   *file.State // State returned by Apply
		wantErr     error       // Error returned by Apply
	}{
		"create new target link to dir item": {
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: nil,
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		"create new target link to non-dir item": {
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: nil,
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     nil,
		},
		"create new target link to sub-item": {
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/dir/sub1/sub2/item"),
			targetState: nil,
			wantState:   linkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantErr:     nil,
		},
		"install dir item contents to existing target dir": {
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: dirState(),
			wantState:   dirState(), // No change in state
			wantErr:     nil,        // No error, so walk will continue with the item's contents
		},
		"target already links to current dir item": {
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("../source/pkg/item", 0),
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     fs.SkipDir, // Do not walk the dir item. It's already linked.
		},
		"target already links to current non-dir item": {
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: linkState("../source/pkg/item", 0),
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     nil,
		},
		"target already links to current sub-item": {
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/dir/sub1/sub2/item"),
			targetState: linkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantState:   linkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantErr:     nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := NewInstallOp(test.source, test.target, nil)

			gotState, gotErr := install.Apply(test.itemFile.name, test.itemFile, test.targetState)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Apply() error:\n got %v\nwant %v", gotErr, test.wantErr)
			}

			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("Apply() state result:\n got %#v\nwant %#v", gotState, test.wantState)
			}
		})
	}
}

func TestInstallOpConlictErrors(t *testing.T) {
	tests := map[string]struct {
		source      string
		target      string
		itemFile    testFile    // The dir entry for the item
		targetState *file.State // The existing target state for the item
	}{
		"target is a file, source is a dir": {
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: fileState(),
		},
		"target is unknown type, source is a dir": {
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: modeState(fs.ModeDevice),
		},
		"target links to a non-dir, source is a dir": {
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("link/to/file", 0o644),
		},
		"target is a dir, source is not a dir": {
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: dirState(),
		},
		"target links to a dir, source is not a dir": {
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: linkState("target/some/dest", fs.ModeDir|0o755),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := NewInstallOp(test.source, test.target, nil)

			gotState, gotErr := install.Apply(test.itemFile.name, test.itemFile, test.targetState)

			if gotState != nil {
				t.Errorf("Apply() state: want nil, got %v", gotState)
			}

			var wantErr *InstallError

			if !errors.As(gotErr, &wantErr) {
				t.Errorf("Apply() error:\n got %s\nwant *InstallError", gotErr)
			}
		})
	}
}

// If the package item is a dir
// and the target is a link to a dir in a duffel package,
// install should replace the target link with a dir
// and analyze the linked dir.
func TestInstallOpMerge(t *testing.T) {
	tests := map[string]struct {
		source        string
		target        string
		itemFile      testFile               // The item file being installed.
		targetState   *file.State            // The target state passed to Apply.
		files         []testFile             // Files to be analyzed for merging.
		wantState     *file.State            // State returned by Apply.
		wantErr       error                  // Error returned by Apply.
		wantNewStates map[string]*file.State // States added to index during Apply.
	}{
		"dest is not in a package": {
			source:      "source",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("../dir1/dir2/item", fs.ModeDir|0o755),
			files:       []testFile{dirFile("dir1/dir2/item")},
			wantState:   nil,
			wantErr:     file.ErrNotInPackage,
		},
		"dest is a duffel source dir": {
			source:      "source",
			target:      "target-dir",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("../duffel/source-dir", fs.ModeDir|0o755),
			files: []testFile{
				regularFile("duffel/source-dir/.duffel"),
			},
			wantState: nil,
			wantErr:   file.ErrIsSource,
		},
		"dest is duffel package": {
			source:      "source",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("../duffel/source/pkg", fs.ModeDir|0o755),
			target:      "target",
			files: []testFile{
				regularFile("duffel/source/.duffel"),
				regularFile("duffel/source/pkg/item/content"),
			},
			wantState: nil,
			wantErr:   file.ErrIsPackage,
		},
		"dest is a top level item in a package": {
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("../duffel/source/pkg/item", fs.ModeDir|0o755),
			files: []testFile{
				regularFile("duffel/source/.duffel"),
				regularFile("duffel/source/pkg/item/content"),
			},
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			wantErr:   nil,
			wantNewStates: map[string]*file.State{
				"target/item/content": linkState(
					"../../duffel/source/pkg/item/content", 0),
			},
		},
		"dest is a nested item in a package": {
			source:   "source",
			target:   "target",
			itemFile: dirFile("source/pkg/item3"),
			targetState: linkState("../duffel/source/pkg/item1/item2/item3",
				fs.ModeDir|0o755),
			files: []testFile{
				regularFile("duffel/source/.duffel"),
				regularFile("duffel/source/pkg/item1/item2/item3/content"),
			},
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			wantErr:   nil,
			wantNewStates: map[string]*file.State{
				"target/item1/item2/item3/content": linkState(
					"../../../../duffel/source/pkg/item1/item2/item3/content",
					0),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			for _, tf := range test.files {
				testFS.Add(tf.name, tf.mode, tf.dest)
			}
			pkgFinder := file.NewPkgFinder(testFS)
			stater := file.Stater{FS: testFS}
			index := NewIndex(stater)
			analyzer := NewAnalyst(testFS, index)
			merger := NewMerger(pkgFinder, analyzer)
			install := NewInstallOp(test.source, test.target, merger)

			entry := test.itemFile
			if !entry.IsDir() {
				t.Fatalf("Bad test.itemFile %q: must be a dir, but is %s",
					entry.name, entry.mode)
			}
			state := test.targetState
			if state.Mode&fs.ModeSymlink == 0 || !state.DestMode.IsDir() {
				t.Fatalf("Bad test.targetItemState: must be a link to a dir, but is %s",
					state)
			}

			gotState, gotErr := install.Apply(test.itemFile.name, entry, state)

			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("Apply(%q) state result:\n got %v\nwant %v",
					test.itemFile.name, gotState, test.wantState)
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Apply(%q) error:\n got %v\nwant %v",
					test.itemFile.name, gotErr, test.wantErr)
			}

			gotStates := maps.Collect(index.All())
			indexDiff := cmp.Diff(test.wantNewStates, gotStates, cmpopts.EquateEmpty())
			if indexDiff != "" {
				t.Errorf("indexed states after Apply(%q):\n%s",
					test.itemFile.name, indexDiff)
			}
		})
	}
}
