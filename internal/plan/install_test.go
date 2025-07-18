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

func TestInstallOp(t *testing.T) {
	entryAndStateSuite.run(t)
	conflictSuite.run(t)
	mergeSuite.run(t)
}

type test struct {
	desc          string                 // Description of the test.
	source        string                 // The source directory to install.
	itemFile      testFile               // The item to install.
	target        string                 // The target directory to install to.
	targetState   *file.State            // The target state passed to Apply.
	files         []testFile             // Files on the file system.
	wantState     *file.State            // State returned by Apply.
	wantErr       error                  // Error returned by Apply.
	wantNewStates map[string]*file.State // States added to index during Apply.
}

type suite struct {
	name  string
	tests []test
}

// Simpler scenarios in which Apply can install the item based solely on
// the given entry and state, without merging and without conflict.
var entryAndStateSuite = suite{
	name: "Entry and State",
	tests: []test{
		{
			desc:        "create new target link to dir item",
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: nil,
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		{
			desc:        "create new target link to non-dir item",
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: nil,
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     nil,
		},
		{
			desc:        "create new target link to sub-item",
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/dir/sub1/sub2/item"),
			targetState: nil,
			wantState:   linkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantErr:     nil,
		},
		{
			desc:        "install dir item contents to existing target dir",
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: dirState(),
			wantState:   dirState(), // No change in state.
			wantErr:     nil,        // No error: Continue walking to install the item's contents.
		},
		{
			desc:        "target already links to current dir item",
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("../source/pkg/item", 0),
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     fs.SkipDir, // Do not walk the dir item. It's already linked.
		},
		{
			desc:        "target already links to current non-dir item",
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: linkState("../source/pkg/item", 0),
			wantState:   linkState("../source/pkg/item", 0),
			wantErr:     nil,
		},
		{
			desc:        "target already links to current sub-item",
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/dir/sub1/sub2/item"),
			targetState: linkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantState:   linkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantErr:     nil,
		},
	},
}

// Scenarios where the entry is a dir and the target state links to a
// dir. If the target state's destination is in a duffel package,
// install must merge the two dirs by executing these steps:
// - Merge the contents of the linked dir into the index.
// - Return an fs.ModeDir state to convert the target to a dir.
// - Return a nil error to walk the contents of the entry.
var mergeSuite = suite{
	name: "Merge",
	tests: []test{
		{
			desc:        "dest is not in a package",
			source:      "source",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("../dir1/dir2/item", fs.ModeDir|0o755),
			files:       []testFile{dirFile("dir1/dir2/item")},
			wantState:   nil,
			wantErr:     file.ErrNotInPackage,
		},
		{
			desc:        "dest is a duffel source dir",
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
		{
			desc:        "dest is duffel package",
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
		{
			desc:        "dest is a top level item in a package",
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
		{
			desc:     "dest is a nested item in a package",
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
	},
}

// Scenarios in which Apply must return an error describing a conflict
// between the entry and the target state.
var conflictSuite = suite{
	name: "Conflict",
	tests: []test{
		{
			desc:        "target is a file, source is a dir",
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: fileState(),
			wantErr: &InstallError{
				Item:        "source/pkg/item",
				ItemType:    fs.ModeDir,
				Target:      "target/item",
				TargetState: fileState(),
			},
		},
		{
			desc:        "target is unknown type, source is a dir",
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: modeState(fs.ModeDevice),
			wantErr: &InstallError{
				Item:        "source/pkg/item",
				ItemType:    fs.ModeDir,
				Target:      "target/item",
				TargetState: modeState(fs.ModeDevice),
			},
		},
		{
			desc:        "target links to a non-dir, source is a dir",
			source:      "source",
			target:      "target",
			itemFile:    dirFile("source/pkg/item"),
			targetState: linkState("link/to/file", 0o644),
			wantErr: &InstallError{
				Item:        "source/pkg/item",
				ItemType:    fs.ModeDir,
				Target:      "target/item",
				TargetState: linkState("link/to/file", 0o644),
			},
		},
		{
			desc:        "target is a dir, source is not a dir",
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: dirState(),
			wantErr: &InstallError{
				Item:        "source/pkg/item",
				ItemType:    0, // regular file
				Target:      "target/item",
				TargetState: dirState(),
			},
		},
		{
			desc:        "target links to a dir, source is not a dir",
			source:      "source",
			target:      "target",
			itemFile:    regularFile("source/pkg/item"),
			targetState: linkState("target/some/dest", fs.ModeDir|0o755),
			wantErr: &InstallError{
				Item:        "source/pkg/item",
				ItemType:    0,
				Target:      "target/item",
				TargetState: linkState("target/some/dest", fs.ModeDir|0o755),
			},
		},
	},
}

func (s suite) run(t *testing.T) {
	t.Run(s.name, func(t *testing.T) {
		for _, test := range s.tests {
			test.run(t)
		}
	})
}

func (test test) run(t *testing.T) {
	t.Run(test.desc, func(t *testing.T) {
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
		state := test.targetState

		gotState, gotErr := install.Apply(test.itemFile.name, entry, state)

		if !cmp.Equal(gotState, test.wantState) {
			t.Errorf("Apply(%q) state result:\n got %v\nwant %v",
				test.itemFile.name, gotState, test.wantState)
		}

		switch want := test.wantErr.(type) {
		case *InstallError:
			errDiff := cmp.Diff(want, gotErr)
			if errDiff != "" {
				t.Errorf("Apply(%q) error:\n%s",
					test.itemFile.name, errDiff)
			}
		default:
			if !errors.Is(gotErr, want) {
				t.Errorf("Apply(%q) error:\n got: %v\nwant: %v",
					test.itemFile.name, gotErr, want)
			}
		}

		gotStates := maps.Collect(index.All())
		indexDiff := cmp.Diff(test.wantNewStates, gotStates, cmpopts.EquateEmpty())
		if indexDiff != "" {
			t.Errorf("indexed states after Apply(%q):\n%s",
				test.itemFile.name, indexDiff)
		}
	})
}

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
