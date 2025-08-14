package plan

import (
	"bytes"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
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

type installTest struct {
	desc          string     // Description of the test.
	sourceItem    SourceItem // The state of the source item.
	targetItem    TargetItem // The state of the target item as of any earlier planning.
	wantMergeCall *mergeCall // The call the installer must make to Merge, if any.
	wantState     file.State // State result.
	wantErr       error      // Error result.
}

// Simpler scenarios that do not involve merging or conflicts.
var entryAndStateSuite = installSuite{
	name: "Entry and State",
	tests: []installTest{
		{
			desc:       "create new target link to file item",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeFile),
			targetItem: NewTargetItem("target", "item", file.NoFileState()),
			wantState:  file.LinkState("../source/pkg/item", file.TypeFile),
		},
		{
			desc:       "create new target link to dir item",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeDir),
			targetItem: NewTargetItem("target", "item", file.NoFileState()),
			wantState:  file.LinkState("../source/pkg/item", file.TypeDir),
			wantErr:    fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		{
			desc:       "create new target link to symlink item",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeSymlink),
			targetItem: NewTargetItem("target", "item", file.NoFileState()),
			wantState:  file.LinkState("../source/pkg/item", file.TypeSymlink),
		},
		{
			desc:       "create new target link to sub-item",
			sourceItem: NewSourceItem("source", "pkg", "dir/sub1/sub2/item", file.TypeFile),
			targetItem: NewTargetItem("target", "dir/sub1/sub2/item", file.NoFileState()),
			wantState:  file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", file.TypeFile),
		},
		{
			desc:       "existing target is link to nowhere",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeFile),
			targetItem: NewTargetItem("target", "item",
				file.LinkState("link/to/nowhere", file.TypeNoFile)),
			wantState: file.LinkState("../source/pkg/item", file.TypeFile),
		},
		{
			desc:       "install dir item contents to existing target dir",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeDir),
			targetItem: NewTargetItem("target", "item", file.DirState()),
			wantState:  file.DirState(), // No change in state.
			wantErr:    nil,             // No error: Continue walking to install the item's contents.
		},
		{
			desc:       "target already links to current dir item",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeDir),
			targetItem: NewTargetItem("target", "item",
				file.LinkState("../source/pkg/item", file.TypeDir)),
			wantState: file.LinkState("../source/pkg/item", file.TypeDir),
			wantErr:   fs.SkipDir, // Do not walk the dir item. It's already linked.
		},
		{
			desc:       "target already links to current non-dir item",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeFile),
			targetItem: NewTargetItem("target", "item",
				file.LinkState("../source/pkg/item", file.TypeFile)),
			wantState: file.LinkState("../source/pkg/item", file.TypeFile),
			wantErr:   nil,
		},
		{
			desc:       "target already links to current sub-item",
			sourceItem: NewSourceItem("source", "pkg", "dir/sub1/sub2/item", file.TypeFile),
			targetItem: NewTargetItem("target", "dir/sub1/sub2/item",
				file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", file.TypeFile)),
			wantState: file.LinkState("../../../../source/pkg/dir/sub1/sub2/item", file.TypeFile),
			wantErr:   nil,
		},
	},
}

// Scenarios where installing a directory source item requires merging its contents
// with items from another package installed or planned earlier.
var mergeSuite = installSuite{
	name: "Merge",
	tests: []installTest{
		{
			desc:       "merge succeeds",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeDir),
			targetItem: NewTargetItem("target", "item",
				file.LinkState("../duffel/source-dir", file.TypeDir)),
			wantMergeCall: &mergeCall{name: "duffel/source-dir", err: nil},
			wantState:     file.DirState(),
			wantErr:       nil,
		},
		{
			desc:       "merge fails",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeDir),
			targetItem: NewTargetItem("target", "item",
				file.LinkState("../duffel/source-dir", file.TypeDir)),
			wantMergeCall: &mergeCall{
				name: "duffel/source-dir",
				err:  &MergeError{Name: "duffel/source-dir", Err: ErrIsSource},
			},
			wantErr: &MergeError{
				Name: "duffel/source-dir",
				Err:  ErrIsSource,
			},
		},
	},
}

// Scenarios where the source file conflicts
// with the existing or planned state of the target file.
var conflictSuite = installSuite{
	name: "Conflict",
	tests: []installTest{
		{
			desc:       "target is a file, source is a dir",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeDir),
			targetItem: NewTargetItem("target", "item", file.FileState()),
			wantErr: &ConflictError{
				SourceItem{NewSourcePath("source", "pkg", "item"), file.TypeDir},
				TargetItem{NewTargetPath("target", "item"), file.FileState()},
			},
		},
		{
			desc:       "target links to a non-dir, source is a dir",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeDir),
			targetItem: NewTargetItem("target", "item",
				file.LinkState("link/to/file", file.TypeFile)),
			wantErr: &ConflictError{
				SourceItem{NewSourcePath("source", "pkg", "item"), file.TypeDir},
				TargetItem{
					NewTargetPath("target", "item"),
					file.LinkState("link/to/file", file.TypeFile),
				},
			},
		},
		{
			desc:       "target is a dir, source is not a dir",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeFile),
			targetItem: NewTargetItem("target", "item", file.DirState()),
			wantErr: &ConflictError{
				SourceItem{NewSourcePath("source", "pkg", "item"), file.TypeFile},
				TargetItem{NewTargetPath("target", "item"), file.DirState()},
			},
		},
		{
			desc:       "target links to a dir, source is not a dir",
			sourceItem: NewSourceItem("source", "pkg", "item", file.TypeFile),
			targetItem: NewTargetItem("target", "item",
				file.LinkState("target/some/dest", file.TypeDir)),
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

type installSuite struct {
	name  string
	tests []installTest
}

func (s installSuite) run(t *testing.T) {
	t.Run(s.name, func(t *testing.T) {
		for _, test := range s.tests {
			test.run(t)
		}
	})
}

func (test installTest) run(t *testing.T) {
	t.Run(test.desc, func(t *testing.T) {
		var logbuf bytes.Buffer
		logger := log.Logger(&logbuf, duftest.LogLevel)
		defer duftest.Dump(t, "log", &logbuf)

		testMerger := &testMerger{wantCall: test.wantMergeCall}

		install := &installer{testMerger}

		gotState, gotErr := install.Analyze(test.sourceItem, test.targetItem, logger)
		testMerger.checkCall(t)

		if diff := cmp.Diff(test.wantState, gotState); diff != "" {
			t.Errorf("state:\n%s", diff)
		}

		switch want := test.wantErr.(type) {
		case *ConflictError, *MergeError:
			if diff := cmp.Diff(want, gotErr); diff != "" {
				t.Errorf("error:\n%s", diff)
			}
		default:
			if diff := cmp.Diff(test.wantErr, gotErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("error:\n%s", diff)
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
