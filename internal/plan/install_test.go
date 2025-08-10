package plan

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	"github.com/google/go-cmp/cmp"
)

func TestInstall(t *testing.T) {
	entryAndStateSuite.run(t)
	conflictSuite.run(t)
	mergeSuite.run(t)
}

type test struct {
	desc          string     // Description of the test.
	itemPath      SourcePath // The package item to install.
	entry         file.Type  // The entry for the package item.
	target        string     // The target directory to install to.
	targetState   file.State // The state of the target file.
	wantMergeCall *mergeCall //
	wantState     file.State // State result.
	wantErr       error      // Error result.
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
// dir. Install must call merge.
var errTestMerge = errors.New("error from Merge")
var mergeSuite = suite{
	name: "Merge",
	tests: []test{
		{
			desc:        "merge returns error",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../duffel/source-dir", file.TypeDir),
			wantMergeCall: &mergeCall{
				name: "duffel/source-dir",
				err:  errTestMerge,
			},
			wantErr: errTestMerge,
		},
		{
			desc:        "merge succeeds",
			itemPath:    NewSourcePath("source", "pkg", "item"),
			entry:       file.TypeDir,
			target:      "target",
			targetState: file.LinkState("../duffel/source-dir", file.TypeDir),
			wantMergeCall: &mergeCall{
				name: "duffel/source-dir",
				err:  nil,
			},
			wantState: file.DirState(),
			wantErr:   nil,
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

		testMerger := &testMerger{wantCall: test.wantMergeCall}

		sourceItem := SourceItem{test.itemPath, test.entry}
		targetItem := TargetItem{
			NewTargetPath(test.target, test.itemPath.Item),
			test.targetState,
		}

		install := &installer{testMerger}
		gotState, gotErr := install.Analyze(sourceItem, targetItem, logger)

		testMerger.checkCall(t)

		if diff := cmp.Diff(test.wantState, gotState); diff != "" {
			t.Errorf("state:\n%s", diff)
		}

		switch want := test.wantErr.(type) {
		case *ConflictError:
			if diff := cmp.Diff(want, gotErr); diff != "" {
				t.Errorf("error:\n%s", diff)
			}
		default:
			if !errors.Is(gotErr, want) {
				t.Errorf("error:\n got: %v\nwant: %v", gotErr, want)
			}
		}
	})
}

type mergeCall struct {
	name string
	err  error
}

type testMerger struct {
	wantCall *mergeCall
	gotCall  bool
	gotName  string
}

func (m *testMerger) Merge(gotName string, _ *slog.Logger) error {
	m.gotCall = true
	m.gotName = gotName
	if m.wantCall != nil {
		return m.wantCall.err
	}
	return nil
}

func (m *testMerger) checkCall(t *testing.T) {
	t.Helper()
	if m.wantCall == nil {
		if m.gotCall {
			t.Errorf("Unexpected Merge(%q)", m.gotName)
		}
		return
	}
	if !m.gotCall {
		t.Error("Want call to Merge(), got none")
		return
	}
	if m.wantCall.name != m.gotName {
		t.Errorf("Merge() called with %q, want %q", m.gotName, m.wantCall.name)
	}
}
