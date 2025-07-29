package analyze_test

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"testing"

	. "github.com/dhemery/duffel/internal/analyze"

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
	item          SourcePath             // The package item to pass to Install.Apply.
	entry         fs.DirEntry            // The entry to pass to Install.Apply.
	target        string                 // The target directory to install to.
	targetState   *file.State            // The target state passed to Apply.
	files         []*errfs.File          // Files on the file system.
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
			desc:        "create new target link to file item",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", 0o644),
			target:      "target",
			targetState: nil,
			wantState:   file.LinkState("../source/pkg/item", 0),
			wantErr:     nil,
		},
		{
			desc:        "create new target link to dir item",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: nil,
			wantState:   file.LinkState("../source/pkg/item", fs.ModeDir),
			wantErr:     fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		{
			desc:        "create new target link to symlink item",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeSymlink),
			target:      "target",
			targetState: nil,
			wantState:   file.LinkState("../source/pkg/item", fs.ModeSymlink),
			wantErr:     nil,
		},
		{
			desc:        "create new target link to sub-item",
			item:        NewSourcePath("source", "pkg", "dir/sub1/sub2/item"),
			entry:       errfs.DirEntry("item", 0o644),
			target:      "target",
			targetState: nil,
			wantState:   file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantErr:     nil,
		},
		{
			desc:        "install dir item contents to existing target dir",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.DirState(),
			wantState:   file.DirState(), // No change in state.
			wantErr:     nil,             // No error: Continue walking to install the item's contents.
		},
		{
			desc:        "target already links to current dir item",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.LinkState("../source/pkg/item", fs.ModeDir),
			wantState:   file.LinkState("../source/pkg/item", fs.ModeDir),
			wantErr:     fs.SkipDir, // Do not walk the dir item. It's already linked.
		},
		{
			desc:        "target already links to current non-dir item",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", 0o644),
			target:      "target",
			targetState: file.LinkState("../source/pkg/item", 0),
			wantState:   file.LinkState("../source/pkg/item", 0),
			wantErr:     nil,
		},
		{
			desc:        "target already links to current sub-item",
			item:        NewSourcePath("source", "pkg", "dir/sub1/sub2/item"),
			entry:       errfs.DirEntry("item", 0o644),
			target:      "target",
			targetState: file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
			wantState:   file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", 0),
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
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.LinkState("../dir1/dir2/item", fs.ModeDir),
			files:       []*errfs.File{errfs.NewDir("dir1/dir2/item", 0o755)},
			wantState:   nil,
			wantErr:     ErrNotInPackage,
		},
		{
			desc:        "dest is a duffel source dir",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.LinkState("../duffel/source-dir", fs.ModeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source-dir/.duffel", 0o644),
			},
			wantState: nil,
			wantErr:   ErrIsSource,
		},
		{
			desc:        "dest is duffel package",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.LinkState("../duffel/source/pkg", fs.ModeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source/.duffel", 0o644),
				errfs.NewFile("duffel/source/pkg/item/content", 0o644),
			},
			wantState: nil,
			wantErr:   ErrIsPackage,
		},
		{
			desc:        "dest is a top level item in a package",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.LinkState("../duffel/source/pkg/item", fs.ModeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source/.duffel", 0o644),
				errfs.NewFile("duffel/source/pkg/item/content", 0o644),
			},
			wantState: &file.State{Type: fs.ModeDir},
			wantErr:   nil,
			wantNewStates: map[string]*file.State{
				"target/item/content": file.LinkState(
					"../../duffel/source/pkg/item/content", 0),
			},
		},
		{
			desc:        "dest is a nested item in a package",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.LinkState("../duffel/source/pkg/item1/item2/item3", fs.ModeDir),
			files: []*errfs.File{
				errfs.NewFile("duffel/source/.duffel", 0o644),
				errfs.NewFile("duffel/source/pkg/item1/item2/item3/content", 0o644),
			},
			wantState: &file.State{Type: fs.ModeDir},
			wantErr:   nil,
			wantNewStates: map[string]*file.State{
				"target/item1/item2/item3/content": file.LinkState(
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
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.FileState(),
			wantErr: &ConflictError{
				Item:        NewSourcePath("source", "pkg", "item"),
				ItemType:    fs.ModeDir,
				Target:      NewTargetPath("target", "item"),
				TargetState: file.FileState(),
			},
		},
		{
			desc:        "target is unknown type, source is a dir",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: &file.State{Type: fs.ModeDevice},
			wantErr: &ConflictError{
				Item:        NewSourcePath("source", "pkg", "item"),
				ItemType:    fs.ModeDir,
				Target:      NewTargetPath("target", "item"),
				TargetState: &file.State{Type: fs.ModeDevice},
			},
		},
		{
			desc:        "target links to a non-dir, source is a dir",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", fs.ModeDir|0o755),
			target:      "target",
			targetState: file.LinkState("link/to/file", 0o644),
			wantErr: &ConflictError{
				Item:        NewSourcePath("source", "pkg", "item"),
				ItemType:    fs.ModeDir,
				Target:      NewTargetPath("target", "item"),
				TargetState: file.LinkState("link/to/file", 0o644),
			},
		},
		{
			desc:        "target is a dir, source is not a dir",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", 0o644),
			target:      "target",
			targetState: file.DirState(),
			wantErr: &ConflictError{
				Item:        NewSourcePath("source", "pkg", "item"),
				ItemType:    0, // regular file
				Target:      NewTargetPath("target", "item"),
				TargetState: file.DirState(),
			},
		},
		{
			desc:        "target links to a dir, source is not a dir",
			item:        NewSourcePath("source", "pkg", "item"),
			entry:       errfs.DirEntry("item", 0o644),
			target:      "target",
			targetState: file.LinkState("target/some/dest", fs.ModeDir),
			wantErr: &ConflictError{
				Item:        NewSourcePath("source", "pkg", "item"),
				ItemType:    0,
				Target:      NewTargetPath("target", "item"),
				TargetState: file.LinkState("target/some/dest", fs.ModeDir),
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
		logger := slog.New(slog.NewJSONHandler(&logbuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		testFS := errfs.New()
		for _, tf := range test.files {
			errfs.Add(testFS, tf)
		}
		stater := file.NewStater(testFS)
		index := NewIndex(stater, logger)
		analyst := NewAnalyst(testFS, test.target, index, logger)
		itemizer := NewItemizer(testFS)
		merger := NewMerger(itemizer, analyst, logger)
		install := NewInstall(test.target, merger, logger)

		state := test.targetState

		targetItem := NewTargetPath(test.target, test.item.Item())
		gotState, gotErr := install.Apply(test.item, test.entry, targetItem, state)

		if !cmp.Equal(gotState, test.wantState) {
			t.Errorf("Apply(%q) state result:\n got %v\nwant %v",
				test.item, gotState, test.wantState)
		}

		switch want := test.wantErr.(type) {
		case *ConflictError:
			if diff := cmp.Diff(want, gotErr); diff != "" {
				t.Errorf("Apply(%q) error:\n%s",
					test.item, diff)
			}
		default:
			if !errors.Is(gotErr, want) {
				t.Errorf("Apply(%q) error:\n got: %v\nwant: %v",
					test.item, gotErr, want)
			}
		}

		gotStates := map[string]*file.State{}
		for n, spec := range index.All() {
			gotStates[n] = spec.Planned
		}
		if diff := cmp.Diff(test.wantNewStates, gotStates, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("planned states after Apply(%q):\n%s",
				test.item, diff)
		}

		if t.Failed() || testing.Verbose() {
			t.Log("files:\n", testFS)
			t.Log("log:\n", logbuf.String())
		}
	})
}
