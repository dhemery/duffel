package plan_test

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	. "github.com/dhemery/duffel/internal/plan"
)

func TestEntryAnalyzer(t *testing.T) {
	itemAnalyzerSuite.run(t)
	earlyExitSuite.run(t)
}

type entryAnalyzerTest struct {
	desc              string     // Description of the test.
	targetItem        targetItem // The state of the target item in the index.
	sourceItem        sourceItem // The source item state passed to Analyze.
	itemAnalyzerState file.State // The state result from ItemAnalyzer.
	itemAnalyzerError error      // The error result from ItemAnalyzer.
	wantErr           error      // Error result.
	wantState         file.State // State passed to index.SetState.
}

var (
	errFromIndex        = errors.New("error returned from index")
	errFromItemAnalyzer = errors.New("error returned from item analyzer")
	errPassedToAnalyze  = errors.New("error passed to Analyze")
)

// Scenarios where Analyze calls the ItemAnalyzer.
var itemAnalyzerSuite = entryAnalyzerSuite{
	name: "ItemAnalyzer",
	tests: []entryAnalyzerTest{
		{
			desc:              "target is no file, item analyzer returns state",
			targetItem:        targetNoFileItem("target", "item"),
			sourceItem:        sourceFileItem("source", "pkg", "item"),
			itemAnalyzerState: file.LinkState("item/func/dest", file.TypeFile),
			wantState:         file.LinkState("item/func/dest", file.TypeFile),
		},
		{
			desc:              "target is no file, item analyzer returns state and SkipDir",
			targetItem:        targetNoFileItem("target", "item"),
			sourceItem:        sourceDirItem("source", "pkg", "item"),
			itemAnalyzerState: file.LinkState("item/func/dest", file.TypeFile),
			itemAnalyzerError: fs.SkipDir,
			wantErr:           fs.SkipDir,
			wantState:         file.LinkState("item/func/dest", file.TypeFile),
		},
		{
			desc:              "target is no file, item analyzer returns error",
			targetItem:        targetNoFileItem("target", "item"),
			sourceItem:        sourceFileItem("source", "pkg", "item"),
			itemAnalyzerError: errFromItemAnalyzer,
			wantErr:           errFromItemAnalyzer,
		},
		{
			desc:              "target is dir, item analyzer returns state",
			targetItem:        targetDirItem("target", "item"),
			sourceItem:        sourceFileItem("source", "pkg", "item"),
			itemAnalyzerState: file.LinkState("item/func/dest", file.TypeFile),
			wantState:         file.LinkState("item/func/dest", file.TypeFile),
		},
		{
			desc:              "target is link, item analyzer returns state",
			targetItem:        targetLinkItem("target", "item", "index/dest", file.TypeFile),
			sourceItem:        sourceFileItem("source", "pkg", "item"),
			itemAnalyzerState: file.DirState(),
			wantState:         file.DirState(),
		},
		{
			desc:              "target is file, item analyzer returns error",
			targetItem:        targetFileItem("target", "item"),
			sourceItem:        sourceFileItem("source", "pkg", "item"),
			itemAnalyzerError: errFromItemAnalyzer,
			wantErr:           errFromItemAnalyzer,
		},
		{
			desc:              "target is dir, item analyzer returns error",
			targetItem:        targetDirItem("target", "item"),
			sourceItem:        sourceFileItem("source", "pkg", "item"),
			itemAnalyzerError: errFromItemAnalyzer,
			wantErr:           errFromItemAnalyzer,
		},
		{
			desc:              "target is link, item analyzer returns error",
			targetItem:        targetLinkItem("target", "item", "index/dest", file.TypeFile),
			sourceItem:        sourceFileItem("source", "pkg", "item"),
			itemAnalyzerError: errFromItemAnalyzer,
			wantErr:           errFromItemAnalyzer,
		},
	},
}

// Scenarios where Analyze determines the outcome without calling ItemAnalyzer.
var earlyExitSuite = entryAnalyzerSuite{
	name: "EarlyExit",
	tests: []entryAnalyzerTest{
		{
			desc:       "package directory",
			sourceItem: sourceDirItem("source", "pkg", "."),
			wantErr:    nil,
		},
		{
			desc:       "error arg for package directory",
			sourceItem: sourceDirErrorItem("source", "pkg", ".", errPassedToAnalyze),
			wantErr:    errPassedToAnalyze,
		},
		{
			desc:       "error arg for item",
			sourceItem: sourceDirErrorItem("source", "pkg", "item", errPassedToAnalyze),
			wantErr:    errPassedToAnalyze,
		},
		{
			desc:       "index error for item",
			targetItem: targetError("target", "item", errFromIndex),
			sourceItem: sourceDirItem("source", "pkg", "item"),
			wantErr:    errFromIndex,
		},
	},
}

type entryAnalyzerSuite struct {
	name  string
	tests []entryAnalyzerTest
}

func (s entryAnalyzerSuite) run(t *testing.T) {
	t.Run(s.name, func(t *testing.T) {
		for _, test := range s.tests {
			test.run(t)
		}
	})
}

func (test entryAnalyzerTest) run(t *testing.T) {
	t.Run(test.desc, func(t *testing.T) {
		var logbuf bytes.Buffer
		logger := log.Logger(&logbuf, duftest.LogLevel)
		defer duftest.Dump(t, "log", &logbuf)

		testItemAnalyzer := &testItemAnalyzer{
			state: test.itemAnalyzerState,
			err:   test.itemAnalyzerError,
		}

		ea := EntryAnalyzer{
			WalkRoot:     test.WalkRoot(),
			Target:       test.TargetDir(),
			Index:        &test.targetItem,
			ItemAnalyzer: testItemAnalyzer,
			Logger:       logger,
		}

		err := ea.Analyze(test.NameArg(), test.EntryArg(), test.ErrArg())

		if !errors.Is(err, test.wantErr) {
			t.Errorf("error:\n got: %v\nwant: %v", err, test.wantErr)
		}

		test.targetItem.checkSetState(t, test.TargetItemName(), test.wantState)
		testItemAnalyzer.checkCall(t, test.SourceItem(), test.TargetItem())
	})
}

func (test entryAnalyzerTest) EntryArg() fs.DirEntry {
	return errfs.DirEntry(test.SourcePath().Item, test.sourceItem.fmode)
}

func (test entryAnalyzerTest) ErrArg() error {
	return test.sourceItem.errArg
}

func (test entryAnalyzerTest) NameArg() string {
	return test.SourcePath().String()
}

func (test entryAnalyzerTest) Package() string {
	return test.SourcePath().Package
}

func (test entryAnalyzerTest) SourceDir() string {
	return test.SourcePath().Source
}

func (test entryAnalyzerTest) SourceItem() SourceItem {
	return test.sourceItem.sourceItem
}

func (test entryAnalyzerTest) SourcePath() SourcePath {
	return test.SourceItem().Path
}

func (test entryAnalyzerTest) TargetDir() string {
	return test.TargetPath().Target
}

func (test entryAnalyzerTest) TargetPath() TargetPath {
	return test.TargetItem().Path
}

func (test entryAnalyzerTest) TargetItem() TargetItem {
	return test.targetItem.targetItem
}

func (test entryAnalyzerTest) TargetItemName() string {
	return test.TargetPath().String()
}

func (test entryAnalyzerTest) WalkRoot() SourcePath {
	return test.SourcePath().WithItem("")
}

type sourceItem struct {
	sourceItem SourceItem  // The item path and type.
	fmode      fs.FileMode // The source item's file mode. Must match ftype.
	errArg     error       // Error passed to PackageOp's visit func.
}

func sourceDirItem(source, pkg, item string) sourceItem {
	return sourceItem{
		sourceItem: NewSourceItem(source, pkg, item, file.TypeDir),
		fmode:      fs.ModeDir | 0o755,
	}
}

func sourceDirErrorItem(source, pkg, item string, err error) sourceItem {
	return sourceItem{
		sourceItem: NewSourceItem(source, pkg, item, file.TypeDir),
		fmode:      fs.ModeDir | 0o755,
		errArg:     err,
	}
}

func sourceFileItem(source, pkg, item string) sourceItem {
	return sourceItem{
		sourceItem: NewSourceItem(source, pkg, item, file.TypeFile),
		fmode:      0o644,
	}
}

// A targetItem is an index with the state of a single target item.
type targetItem struct {
	targetItem TargetItem  // The target item.
	err        error       // Error to return from State.
	gotName    string      // Name passed to SetState.
	gotState   *file.State // State passed to SetState.
}

func targetNoFileItem(target, item string) targetItem {
	return targetItem{
		targetItem: NewTargetItem(target, item, file.NoFileState()),
	}
}

func targetFileItem(target, item string) targetItem {
	return targetItem{
		targetItem: NewTargetItem(target, item, file.FileState()),
	}
}

func targetDirItem(target, item string) targetItem {
	return targetItem{
		targetItem: NewTargetItem(target, item, file.DirState()),
	}
}

func targetLinkItem(target, item, dest string, destType file.Type) targetItem {
	return targetItem{
		targetItem: NewTargetItem(target, item, file.LinkState(dest, destType)),
	}
}

func targetError(target, item string, err error) targetItem {
	return targetItem{
		targetItem: TargetItem{Path: NewTargetPath(target, item)},
		err:        err,
	}
}

func (ts *targetItem) State(name string, _ *slog.Logger) (file.State, error) {
	ts.gotName = name
	return ts.targetItem.State, ts.err
}

func (ts *targetItem) SetState(name string, state file.State, _ *slog.Logger) {
	ts.gotName = name
	ts.gotState = &state
}

func (ts *targetItem) checkSetState(t *testing.T, wantName string, wantState file.State) {
	t.Helper()
	if wantState.Type == file.TypeUnknown {
		if ts.gotState != nil {
			t.Errorf("unwanted call to index.SetState():\n name: %q\nstate: %s",
				ts.gotName, ts.gotState)
		}
		return
	}
	if wantName != ts.gotName {
		t.Errorf("index.SetState() name arg: got %q, want %q", ts.gotName, wantName)
	}
	if diff := cmp.Diff(&wantState, ts.gotState); diff != "" {
		t.Errorf("index.SetState() state arg:\n%s", diff)
	}
}

type testItemAnalyzer struct {
	state     file.State  // State to return from Analyze.
	err       error       // Error to return from Analyze.
	gotSource *SourceItem // SourceItem passed to Analyze.
	gotTarget *TargetItem // TargetItem passed to Analyze.
}

func (tia *testItemAnalyzer) Goal() ItemGoal {
	return GoalInstall
}

func (tia *testItemAnalyzer) Analyze(gotSource SourceItem, gotTarget TargetItem, l *slog.Logger) (file.State, error) {
	tia.gotSource, tia.gotTarget = &gotSource, &gotTarget
	return tia.state, tia.err
}

func (tia *testItemAnalyzer) checkCall(t *testing.T, wantSource SourceItem, wantTarget TargetItem) {
	t.Helper()
	if tia.gotSource == nil {
		return
	}
	if diff := cmp.Diff(&wantSource, tia.gotSource); diff != "" {
		t.Errorf("item func source arg:\n%s", diff)
	}
	if diff := cmp.Diff(&wantTarget, tia.gotTarget); diff != "" {
		t.Errorf("item func target arg:\n%s", diff)
	}
}
