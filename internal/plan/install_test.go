package plan_test

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
	. "github.com/dhemery/duffel/internal/plan"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestInstall(t *testing.T) {
	entryAndStateSuite.run(t)
	conflictSuite.run(t)
	mergeSuite.run(t)
}

type test struct {
	desc          string                // Description of the test.
	itemPath      SourcePath            // The package item to pass to Install.Apply.
	entry         file.Type             // The entry to pass to Install.Apply.
	target        string                // The target directory to install to.
	targetState   file.State            // The target state passed to Apply.
	files         []*errfs.File         // Files on the file system.
	wantState     file.State            // State returned by Apply.
	wantErr       error                 // Error returned by Apply.
	wantNewStates map[string]file.State // States added to index during Apply.
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
			desc:        "create new target link to file item",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeFile,
			target:      "target",
			targetState: file.NoFileState(),
			wantState:   file.LinkState("../source/pkg/item", file.TypeFile),
		},
		{
			desc:        "create new target link to dir item",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.NoFileState(),
			wantState:   file.LinkState("../source/pkg/item", file.TypeDir),
			wantErr:     fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		{
			desc:        "create new target link to symlink item",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeSymlink,
			target:      "target",
			targetState: file.NoFileState(),
			wantState:   file.LinkState("../source/pkg/item", file.TypeSymlink),
		},
		{
			desc:        "create new target link to sub-item",
			itemPath:    NewSourcePath("source", "pkg", "dir/sub1/sub2/item"),
			entry:       file.TypeFile,
			target:      "target",
			targetState: file.NoFileState(),
			wantState:   file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", file.TypeFile),
		},
		{
			desc:        "existing target is link to nowhere",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeFile,
			target:      "target",
			targetState: file.LinkState("link/to/nowhere", file.TypeNoFile),
			wantState:   file.LinkState("../source/pkg/item", file.TypeFile),
		},
		{
			desc:        "install dir item contents to existing target dir",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.DirState(),
			wantState:   file.DirState(), // No change in state.
			wantErr:     nil,             // No error: Continue walking to install the item's contents.
		},
		{
			desc:        "target already links to current dir item",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../source/pkg/item", file.TypeDir),
			wantState:   file.LinkState("../source/pkg/item", file.TypeDir),
			wantErr:     fs.SkipDir, // Do not walk the dir item. It's already linked.
		},
		{
			desc:        "target already links to current non-dir item",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeFile,
			target:      "target",
			targetState: file.LinkState("../source/pkg/item", file.TypeFile),
			wantState:   file.LinkState("../source/pkg/item", file.TypeFile),
			wantErr:     nil,
		},
		{
			desc:        "target already links to current sub-item",
			itemPath:    NewSourcePath("source", "pkg", "dir/sub1/sub2/item"),
			entry:       file.TypeFile,
			target:      "target",
			targetState: file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", file.TypeFile),
			wantState:   file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", file.TypeFile),
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
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../dir1/dir2/item", file.TypeDir),
			files:       []*errfs.File{errfs.NewDir("dir1/dir2/item", 0o755)},
			wantErr:     ErrNotInPackage,
		},
		{
			desc:        "dest is a duffel source dir",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../duffel/source-dir", file.TypeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
			},
			wantErr: ErrIsSource,
		},
		{
			desc:        "dest is duffel package",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../duffel/source/pkg", file.TypeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source/.duffel", 0o644),
				errfs.NewFile("duffel/source/pkg/item/content", 0o644),
			},
			wantErr: ErrIsPackage,
		},
		{
			desc:        "dest is a top level item in a package",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../duffel/source/pkg/item", file.TypeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source/.duffel", 0o644),
				errfs.NewFile("duffel/source/pkg/item/content", 0o644),
			},
			wantState: file.DirState(),
			wantErr:   nil,
			wantNewStates: map[string]file.State{
				"target/item/content": file.LinkState(
					"../../duffel/source/pkg/item/content",
					file.TypeFile),
			},
		},
		{
			desc:        "dest is a nested item in a package",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../duffel/source/pkg/item1/item2/item3", file.TypeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source/.duffel", 0o644),
				errfs.NewFile("duffel/source/pkg/item1/item2/item3/content", 0o644),
			},
			wantState: file.DirState(),
			wantErr:   nil,
			wantNewStates: map[string]file.State{
				"target/item1/item2/item3/content": file.LinkState(
					"../../../../duffel/source/pkg/item1/item2/item3/content",
					file.TypeFile),
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
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.FileState(),
			wantErr: &ConflictError{
				SourceItem{NewSourcePath("source", "pkg", "item"), file.TypeDir},
				TargetItem{NewTargetPath("target", "item"), file.FileState()},
			},
		},
		{
			desc:        "target links to a non-dir, source is a dir",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("link/to/file", file.TypeFile),
			wantErr: &ConflictError{
				SourceItem{NewSourcePath("source", "pkg", "item"), file.TypeDir},
				TargetItem{
					NewTargetPath("target", "item"),
					file.LinkState("link/to/file", file.TypeFile),
				},
			},
		},
		{
			desc:        "target is a dir, source is not a dir",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeFile,
			target:      "target",
			targetState: file.DirState(),
			wantErr: &ConflictError{
				SourceItem{NewSourcePath("source", "pkg", "item"), file.TypeFile},
				TargetItem{NewTargetPath("target", "item"), file.DirState()},
			},
		},
		{
			desc:        "target links to a dir, source is not a dir",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeFile,
			target:      "target",
			targetState: file.LinkState("target/some/dest", file.TypeDir),
			wantErr: &ConflictError{
				SourceItem{NewSourcePath("source", "pkg", "item"), file.TypeFile},
				TargetItem{
					NewTargetPath("target", "item"),
					file.LinkState("target/some/dest", file.TypeDir),
				},
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
		var logbuf bytes.Buffer
		logger := log.Logger(&logbuf, duftest.LogLevel)
		defer duftest.Dump(t, "log", &logbuf)

		testFS := errfs.New()
		defer duftest.Dump(t, "files", testFS)

		for _, tf := range test.files {
			errfs.Add(testFS, tf)
		}
		stater := file.NewStater(testFS)
		index := NewIndex(stater)
		analyst := NewAnalyst(testFS, test.target, index)
		itemizer := NewItemizer(testFS)
		merger := NewMerger(itemizer, analyst)
		install := NewInstall(merger)

		sourceItem := SourceItem{test.itemPath, test.entry}
		targetItem := TargetItem{
			NewTargetPath(test.target, test.itemPath.Item),
			test.targetState,
		}
		gotState, gotErr := install.Apply(sourceItem, targetItem, logger)

		if diff := cmp.Diff(test.wantState, gotState); diff != "" {
			t.Errorf("Apply(%q) state result:\n%s", test.itemPath, diff)
		}

		switch want := test.wantErr.(type) {
		case *ConflictError:
			if diff := cmp.Diff(want, gotErr); diff != "" {
				t.Errorf("Apply(%q) error:\n%s",
					test.itemPath, diff)
			}
		default:
			if !errors.Is(gotErr, want) {
				t.Errorf("Apply(%q) error:\n got: %v\nwant: %v",
					test.itemPath, gotErr, want)
			}
		}

		gotStates := map[string]file.State{}
		for n, spec := range index.All() {
			gotStates[n] = spec.Planned
		}
		if diff := cmp.Diff(test.wantNewStates, gotStates, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("planned states after Apply(%q):\n%s",
				test.itemPath, diff)
		}
	})
}
