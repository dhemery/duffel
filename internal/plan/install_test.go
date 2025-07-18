package plan

import (
	"errors"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
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

func TestInstallOp(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	targetToSource, _ := filepath.Rel(target, source)

	tests := map[string]struct {
		item        string      // Item being analyzed, relative to pkg dir
		entry       fs.DirEntry // Dir entry passed to Apply for the item
		targetState *file.State // Target state passed to Apply
		wantState   *file.State // State returned by Apply
		wantErr     error       // Error returned by Apply
	}{
		"create new target link to dir item": {
			item:        "item",
			entry:       testFile{mode: fs.ModeDir | 0o755},
			targetState: nil,
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		"create new target link to non-dir item": {
			item:        "item",
			targetState: nil,
			entry:       testFile{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"create new target link to sub-item": {
			item:        "dir/sub1/sub2/item",
			targetState: nil,
			entry:       testFile{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantErr: nil,
		},
		"install dir item contents to existing target dir": {
			item:        "item",
			entry:       testFile{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
			// No change in state
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// No error, so walk will continue with the item's contents
			wantErr: nil,
		},
		"target already links to current dir item": {
			item:  "item",
			entry: testFile{mode: fs.ModeDir | 0o755},
			targetState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			// Do not walk the dir item. It's already linked.
			wantErr: fs.SkipDir,
		},
		"target already links to current non-dir item": {
			item:  "item",
			entry: testFile{mode: 0o644},
			targetState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"target already links to current sub-item": {
			item:  "dir/sub1/sub2/item",
			entry: testFile{mode: 0o644},
			targetState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := NewInstallOp(source, target, nil)

			sourcePkgItem := path.Join(source, pkg, test.item)
			gotState, gotErr := install.Apply(sourcePkgItem, test.entry, test.targetState)

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
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
		item   = "item"
	)
	tests := map[string]struct {
		sourceEntry fs.DirEntry // The dir entry for the item
		targetState *file.State // The existing target state for the item
	}{
		"target is a file, source is a dir": {
			sourceEntry: dirFile(""),
			targetState: fileState(),
		},
		"target is unknown type, source is a dir": {
			sourceEntry: dirFile(""),
			targetState: modeState(fs.ModeDevice),
		},
		"target links to a non-dir, source is a dir": {
			sourceEntry: dirFile(""),
			targetState: linkState("link/to/file", 0o644),
		},
		"target is a dir, source is not a dir": {
			sourceEntry: regularFile(""),
			targetState: dirState(),
		},
		"target links to a dir, source is not a dir": {
			sourceEntry: regularFile(""),
			targetState: linkState("target/some/dest", fs.ModeDir|0o755),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := NewInstallOp(source, target, nil)

			sourcePkgItem := path.Join(source, pkg, item)
			gotState, gotErr := install.Apply(sourcePkgItem, test.sourceEntry, test.targetState)

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

// If the package item is a dir
// and the target is a link to a dir in a duffel package,
// install should replace the target link with a dir
// and analyze the linked dir.
func TestInstallOpMerge(t *testing.T) {
	tests := map[string]struct {
		source            string
		target            string
		itemFile          testFile               // The item file being installed.
		targetItemState   *file.State            // The target state passed to Apply.
		files             []testFile             // Files to be analyzed for merging.
		wantState         *file.State            // State returned by Apply.
		wantErr           error                  // Error returned by Apply.
		wantIndexedStates map[string]*file.State // States added to index during Apply.
	}{
		"dest is not in a package": {
			source:          "source",
			itemFile:        dirFile("source/pkg/item"),
			targetItemState: linkState("../dir1/dir2/item", fs.ModeDir|0o755),
			files:           []testFile{dirFile("dir1/dir2/item")},
			wantState:       nil,
			wantErr:         file.ErrNotInPackage,
		},
		"dest is a duffel source dir": {
			source:          "source",
			target:          "target-dir",
			itemFile:        dirFile("source/pkg/item"),
			targetItemState: linkState("../duffel/source-dir", fs.ModeDir|0o755),
			files: []testFile{
				regularFile("duffel/source-dir/.duffel"),
			},
			wantState: nil,
			wantErr:   file.ErrIsSource,
		},
		"dest is duffel package": {
			source:          "source",
			itemFile:        dirFile("source/pkg/item"),
			targetItemState: linkState("../duffel/source/pkg", fs.ModeDir|0o755),
			target:          "target",
			files: []testFile{
				regularFile("duffel/source/.duffel"),
				regularFile("duffel/source/pkg/item/content"),
			},
			wantState: nil,
			wantErr:   file.ErrIsPackage,
		},
		"dest is a top level item in a package": {
			source:          "source",
			target:          "target",
			itemFile:        dirFile("source/pkg/item"),
			targetItemState: linkState("../duffel/source/pkg/item", fs.ModeDir|0o755),
			files: []testFile{
				regularFile("duffel/source/.duffel"),
				regularFile("duffel/source/pkg/item/content"),
			},
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			wantErr:   nil,
			wantIndexedStates: map[string]*file.State{
				"target/item/content": linkState(
					"../../duffel/source/pkg/item/content", 0),
			},
		},
		"dest is a nested item in a package": {
			source:   "source",
			target:   "target",
			itemFile: dirFile("source/pkg/item3"),
			targetItemState: linkState("../duffel/source/pkg/item1/item2/item3",
				fs.ModeDir|0o755),
			files: []testFile{
				regularFile("duffel/source/.duffel"),
				regularFile("duffel/source/pkg/item1/item2/item3/content"),
			},
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			wantErr:   nil,
			wantIndexedStates: map[string]*file.State{
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
			state := test.targetItemState
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
			indexDiff := cmp.Diff(test.wantIndexedStates, gotStates, cmpopts.EquateEmpty())
			if indexDiff != "" {
				t.Errorf("indexed states after Apply(%q):\n%s",
					test.itemFile.name, indexDiff)
			}
		})
	}
}
