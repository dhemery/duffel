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

func TestAnalyzeEntry(t *testing.T) {
	analyzeItemSuite.run(t)
	earlyExitSuite.run(t)
}

type analyzeEntryTest struct {
	desc         string           // Description of the test.
	analysis     testAnalysis     // The result of prior analysis.
	itemState    itemState        // The state of the source item to analyze.
	itemAnalyzer testItemAnalyzer // The fake ItemAnalyzer to call.
	wantErr      error            // Error result.
	wantState    *file.State      // State passed to analysis.SetState.
}

var (
	errFromItemFunc      = errors.New("error returned from item func")
	errFromIndexState    = errors.New("error returned from index.State")
	errPassedToVisitFunc = errors.New("error passed to visit func")
)

// Scenarios where AnalyzeEntry calls ItemAnalyzer.AnalyzeItem.
var analyzeItemSuite = analyzeEntrySuite{
	name: "AnalyzeItem",
	tests: []analyzeEntryTest{
		{
			desc:         "index has no file, item func returns state",
			analysis:     targetDoesNotExist("target", "item"),
			itemState:    itemIsFile("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantState:    ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:         "index has no file, item func returns state and SkipDir",
			analysis:     targetDoesNotExist("target", "item"),
			itemState:    itemIsDir("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.LinkState("item/func/dest", file.TypeFile), err: fs.SkipDir},
			wantErr:      fs.SkipDir,
			wantState:    ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:         "index has no file, item func reports error",
			analysis:     targetDoesNotExist("target", "item"),
			itemState:    itemIsFile("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
		{
			desc:         "index has dir, item func returns state",
			analysis:     targetIsDir("target", "item"),
			itemState:    itemIsFile("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantState:    ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:         "index state is link, item func returns state",
			analysis:     targetIsLink("target", "item", "index/dest", file.TypeFile),
			itemState:    itemIsFile("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{state: file.DirState()},
			wantState:    ptr(file.DirState()),
		},
		{
			desc:         "index state is file, item func reports error",
			analysis:     targetIsFile("target", "item"),
			itemState:    itemIsFile("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
		{
			desc:         "index has dir, item func reports error",
			analysis:     targetIsDir("target", "item"),
			itemState:    itemIsFile("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
		{
			desc:         "index has link, item func reports error",
			analysis:     targetIsLink("target", "item", "index/dest", file.TypeFile),
			itemState:    itemIsFile("source", "pkg", "item"),
			itemAnalyzer: testItemAnalyzer{err: errFromItemFunc},
			wantErr:      errFromItemFunc,
			wantState:    nil,
		},
	},
}

// Scenarios where AnalyzeEntry determines the outcome without calling ItemAnalyzer.AnalyzeItem.
var earlyExitSuite = analyzeEntrySuite{
	name: "EarlyExit",
	tests: []analyzeEntryTest{
		{
			desc:      "package directory",
			itemState: itemIsDir("source", "pkg", "."),
			wantErr:   nil,
		},
		{
			desc:      "error arg for package directory",
			itemState: itemIsDirWithError("source", "pkg", ".", errPassedToVisitFunc),
			wantErr:   errPassedToVisitFunc,
		},
		{
			desc:      "error arg for item",
			itemState: itemIsDirWithError("source", "pkg", "item", errPassedToVisitFunc),
			wantErr:   errPassedToVisitFunc,
		},
		{
			desc:      "index error for item",
			analysis:  targetHasError("target", "item", errFromIndexState),
			itemState: itemIsDir("source", "pkg", "item"),
			wantErr:   errFromIndexState,
		},
	},
}

type analyzeEntrySuite struct {
	name  string
	tests []analyzeEntryTest
}

func (s analyzeEntrySuite) run(t *testing.T) {
	t.Run(s.name, func(t *testing.T) {
		for _, test := range s.tests {
			test.run(t)
		}
	})
}

func (test analyzeEntryTest) run(t *testing.T) {
	t.Run(test.desc, func(t *testing.T) {
		var logbuf bytes.Buffer
		logger := log.Logger(&logbuf, duftest.LogLevel)
		defer duftest.Dump(t, "log", &logbuf)

		err := AnalyzeEntry(test.NameArg(), test.EntryArg(), test.ErrArg(), test.WalkRoot(),
			&test.analysis, &test.itemAnalyzer, logger)

		if !errors.Is(err, test.wantErr) {
			t.Errorf("error:\n got: %v\nwant: %v", err, test.wantErr)
		}

		test.analysis.checkSetState(t, test.TargetName(), test.wantState)
		test.itemAnalyzer.checkCall(t, test.SourceItem(), test.TargetItem())
	})
}

func (test analyzeEntryTest) EntryArg() fs.DirEntry {
	return errfs.DirEntry(test.itemState.item, test.itemState.fmode)
}

func (test analyzeEntryTest) ErrArg() error {
	return test.itemState.errArg
}

func (test analyzeEntryTest) NameArg() string {
	return path.Join(test.itemState.source, test.itemState.pkg, test.itemState.item)
}

func (test analyzeEntryTest) Package() string {
	return test.itemState.pkg
}

func (test analyzeEntryTest) SourceDir() string {
	return test.itemState.source
}

func (test analyzeEntryTest) WalkRoot() SourcePath {
	return NewSourcePath(test.itemState.source, test.itemState.pkg, "")
}

func (test analyzeEntryTest) TargetItem() TargetItem {
	return NewTargetItem(test.analysis.target, test.analysis.item, test.analysis.state)
}

func (test analyzeEntryTest) TargetDir() string {
	return test.analysis.target
}

func (test analyzeEntryTest) TargetName() string {
	return path.Join(test.analysis.target, test.analysis.item)
}

func (test analyzeEntryTest) SourceItem() SourceItem {
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

func itemIsDir(source, pkg, item string) itemState {
	return itemState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeDir,
		fmode:  fs.ModeDir | 0o755,
	}
}

func itemIsDirWithError(source, pkg, item string, err error) itemState {
	return itemState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeDir,
		fmode:  fs.ModeDir | 0o755,
		errArg: err,
	}
}

func itemIsFile(source, pkg, item string) itemState {
	return itemState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeFile,
		fmode:  0o644,
	}
}

type testAnalysis struct {
	target   string      // The target part of the item's name.
	item     string      // The item part of the item's name.
	state    file.State  // State to return from State.
	err      error       // Error to return from State.
	gotName  string      // Name passed to SetState.
	gotState *file.State // State passed to SetState.
}

func targetDoesNotExist(target, item string) testAnalysis {
	return testAnalysis{target: target, item: item, state: file.NoFileState()}
}

func targetIsFile(target, item string) testAnalysis {
	return testAnalysis{target: target, item: item, state: file.FileState()}
}

func targetIsDir(target, item string) testAnalysis {
	return testAnalysis{target: target, item: item, state: file.DirState()}
}

func targetIsLink(target, item, dest string, destType file.Type) testAnalysis {
	return testAnalysis{target: target, item: item, state: file.LinkState(dest, destType)}
}

func targetHasError(target, item string, err error) testAnalysis {
	return testAnalysis{target: target, item: item, err: err}
}

func (ta *testAnalysis) Target() string {
	return ta.target
}
func (ta *testAnalysis) State(name string, _ *slog.Logger) (file.State, error) {
	ta.gotName = name
	return ta.state, ta.err
}

func (ta *testAnalysis) SetState(name string, state file.State, _ *slog.Logger) {
	ta.gotName = name
	ta.gotState = &state
}

func (ta *testAnalysis) checkSetState(t *testing.T, wantName string, wantState *file.State) {
	t.Helper()
	if wantState == nil {
		if ta.gotState != nil {
			t.Errorf("unwanted call to index.SetState():\n name: %q\nstate: %s",
				ta.gotName, ta.gotState)
		}
		return
	}
	if wantName != ta.gotName {
		t.Errorf("index.SetState() name arg: got %q, want %q", ta.gotName, wantName)
	}
	if diff := cmp.Diff(wantState, ta.gotState); diff != "" {
		t.Errorf("index.SetState() state arg:\n%s", diff)
	}
}

type testItemAnalyzer struct {
	state     file.State  // State to return from AnalyzeItem.
	err       error       // Error to return from AnalyzeItem.
	gotSource *SourceItem // SourceItem passed to AnalyzeItem.
	gotTarget *TargetItem // TargetItem passed to AnalyzeItem.
}

func (tia *testItemAnalyzer) Goal() Goal {
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
