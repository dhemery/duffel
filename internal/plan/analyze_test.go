package plan_test

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	. "github.com/dhemery/duffel/internal/plan"
)

func TestEntryAnalyzer(t *testing.T) {
	analyzeItemSuite.run(t)
	earlyExitSuite.run(t)
}

type entryAnalyzerTest struct {
	desc         string           // Description of the test.
	targetState  targetState      // The state of the target in the index as a result of prior analysis.
	itemState    itemState        // The state of the source item to analyze.
	itemAnalyzer testItemAnalyzer // The ItemAnalyzer to call.
	wantErr      error            // Error result.
	wantState    *file.State      // State passed to index.SetState.
}

var (
	errFromItemFunc      = errors.New("error returned from item func")
	errFromIndexState    = errors.New("error returned from index.State")
	errPassedToVisitFunc = errors.New("error passed to visit func")
)

// Scenarios where AnalyzeEntry calls ItemAnalyzer.
var analyzeItemSuite = analyzeEntrySuite{
	name: "AnalyzeItem",
	tests: []entryAnalyzerTest{
		{
			desc:         "index has no file, item func returns state",
			targetState:  targetNoFile("target", "item"),
			itemState:    fileItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantState:    ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:         "index has no file, item func returns state and SkipDir",
			targetState:  targetNoFile("target", "item"),
			itemState:    dirItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.LinkState("item/func/dest", file.TypeFile), err: fs.SkipDir},
			wantErr:      fs.SkipDir,
			wantState:    ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:         "index has no file, item func reports error",
			targetState:  targetNoFile("target", "item"),
			itemState:    fileItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
		{
			desc:         "index has dir, item func returns state",
			targetState:  targetDir("target", "item"),
			itemState:    fileItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantState:    ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:         "index state is link, item func returns state",
			targetState:  targetLink("target", "item", "index/dest", file.TypeFile),
			itemState:    fileItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.DirState()},
			wantState:    ptr(file.DirState()),
		},
		{
			desc:         "index state is file, item func reports error",
			targetState:  targetFile("target", "item"),
			itemState:    fileItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
		{
			desc:         "index has dir, item func reports error",
			targetState:  targetDir("target", "item"),
			itemState:    fileItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
		{
			desc:         "index has link, item func reports error",
			targetState:  targetLink("target", "item", "index/dest", file.TypeFile),
			itemState:    fileItem("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
	},
}

// Scenarios where AnalyzeEntry determines the outcome without calling ItemAnalyzer.
var earlyExitSuite = analyzeEntrySuite{
	name: "EarlyExit",
	tests: []entryAnalyzerTest{
		{
			desc:      "package directory",
			itemState: dirItem("source", "pkg", "."),
			wantErr:   nil,
		},
		{
			desc:      "error arg for package directory",
			itemState: dirItemWithError("source", "pkg", ".", errPassedToVisitFunc),
			wantErr:   errPassedToVisitFunc,
		},
		{
			desc:      "error arg for item",
			itemState: dirItemWithError("source", "pkg", "item", errPassedToVisitFunc),
			wantErr:   errPassedToVisitFunc,
		},
		{
			desc:        "index error for item",
			targetState: targetError("target", "item", errFromIndexState),
			itemState:   dirItem("source", "pkg", "item"),
			wantErr:     errFromIndexState,
		},
	},
}

type analyzeEntrySuite struct {
	name  string
	tests []entryAnalyzerTest
}

func (s analyzeEntrySuite) run(t *testing.T) {
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

		analyzer := EntryAnalyzer{
			WalkRoot:     test.WalkRoot(),
			Target:       test.TargetDir(),
			Index:        &test.targetState,
			ItemAnalyzer: &test.itemAnalyzer,
			Logger:       logger,
		}

		err := analyzer.AnalyzeEntry(test.NameArg(), test.EntryArg(), test.ErrArg())

		if !errors.Is(err, test.wantErr) {
			t.Errorf("error:\n got: %v\nwant: %v", err, test.wantErr)
		}

		test.targetState.checkSetState(t, test.TargetName(), test.wantState)
		test.itemAnalyzer.checkCall(t, test.SourceItem(), test.TargetItem())
	})
}

func (test entryAnalyzerTest) EntryArg() fs.DirEntry {
	return errfs.DirEntry(test.itemState.item, test.itemState.fmode)
}

func (test entryAnalyzerTest) ErrArg() error {
	return test.itemState.errArg
}

func (test entryAnalyzerTest) NameArg() string {
	return path.Join(test.itemState.source, test.itemState.pkg, test.itemState.item)
}

func (test entryAnalyzerTest) Package() string {
	return test.itemState.pkg
}

func (test entryAnalyzerTest) SourceDir() string {
	return test.itemState.source
}

func (test entryAnalyzerTest) WalkRoot() SourcePath {
	return NewSourcePath(test.itemState.source, test.itemState.pkg, "")
}

func (test entryAnalyzerTest) TargetItem() TargetItem {
	return NewTargetItem(test.targetState.target, test.targetState.item, test.targetState.state)
}

func (test entryAnalyzerTest) TargetDir() string {
	return test.targetState.target
}

func (test entryAnalyzerTest) TargetName() string {
	return path.Join(test.targetState.target, test.targetState.item)
}

func (test entryAnalyzerTest) SourceItem() SourceItem {
	return NewSourceItem(test.itemState.source, test.itemState.pkg, test.itemState.item, test.itemState.ftype)
}

type itemState struct {
	source string      // The source part of the source item's name.
	pkg    string      // The package part of the source item's name.
	item   string      // The item part of the source item's name.
	ftype  file.Type   // The source item's file type. Must match fmode.
	fmode  fs.FileMode // The source item's file mode. Must match ftype.
	errArg error       // Error passed to PackageOp's visit func.
}

func dirItem(source, pkg, item string) itemState {
	return itemState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeDir,
		fmode:  fs.ModeDir | 0o755,
	}
}

func dirItemWithError(source, pkg, item string, err error) itemState {
	return itemState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeDir,
		fmode:  fs.ModeDir | 0o755,
		errArg: err,
	}
}

func fileItem(source, pkg, item string) itemState {
	return itemState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeFile,
		fmode:  0o644,
	}
}

type targetState struct {
	target   string      // The target part of the item's name.
	item     string      // The item part of the item's name.
	state    file.State  // State to return from State.
	err      error       // Error to return from State.
	gotName  string      // Name passed to SetState.
	gotState *file.State // State passed to SetState.
}

func targetNoFile(target, item string) targetState {
	return targetState{target: target, item: item, state: file.NoFileState()}
}

func targetFile(target, item string) targetState {
	return targetState{target: target, item: item, state: file.FileState()}
}

func targetDir(target, item string) targetState {
	return targetState{target: target, item: item, state: file.DirState()}
}

func targetLink(target, item, dest string, destType file.Type) targetState {
	return targetState{target: target, item: item, state: file.LinkState(dest, destType)}
}

func targetError(target, item string, err error) targetState {
	return targetState{target: target, item: item, err: err}
}

func (ts *targetState) State(name string, _ *slog.Logger) (file.State, error) {
	ts.gotName = name
	return ts.state, ts.err
}

func (ts *targetState) SetState(name string, state file.State, _ *slog.Logger) {
	ts.gotName = name
	ts.gotState = &state
}

func (ts *targetState) checkSetState(t *testing.T, wantName string, wantState *file.State) {
	t.Helper()
	if wantState == nil {
		if ts.gotState != nil {
			t.Errorf("unwanted call to index.SetState():\n name: %q\nstate: %s",
				ts.gotName, ts.gotState)
		}
		return
	}
	if wantName != ts.gotName {
		t.Errorf("index.SetState() name arg: got %q, want %q", ts.gotName, wantName)
	}
	if diff := cmp.Diff(wantState, ts.gotState); diff != "" {
		t.Errorf("index.SetState() state arg:\n%s", diff)
	}
}

type testItemAnalyzer struct {
	state     file.State  // State to return from AnalyzeItem.
	err       error       // Error to return from AnalyzeItem.
	gotSource *SourceItem // SourceItem passed to AnalyzeItem.
	gotTarget *TargetItem // TargetItem passed to AnalyzeItem.
}

func (tia *testItemAnalyzer) Goal() ItemGoal {
	return GoalInstall
}

func (tia *testItemAnalyzer) AnalyzeItem(gotSource SourceItem, gotTarget TargetItem, l *slog.Logger) (file.State, error) {
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

func ptr[T any](v T) *T {
	return &v
}
