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

func TestPackageOp(t *testing.T) {
	itemFuncSuite.run(t)
	earlyExitSuite.run(t)
}

type packageOpTest struct {
	desc      string      // Description of the test
	index     indexItem   // The state of the item in the index before analyzing.
	source    sourceState // The state of the source item to analyze.
	itemFunc  itemFunc    // The results returned by itemFunc.
	wantErr   error       // Error result.
	wantState *file.State // State passed to index.SetState.
}

var (
	errFromItemFunc      = errors.New("error returned from item func")
	errFromIndexState    = errors.New("error returned from index.State")
	errPassedToVisitFunc = errors.New("error passed to visit func")
)

var itemFuncSuite = packageOpSuite{
	name: "ItemFunc",
	tests: []packageOpTest{
		{
			desc:      "index has no file, item func returns state",
			index:     indexNoFile("target", "item"),
			source:    sourceFile("source", "pkg", "item"),
			itemFunc:  itemFunc{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantState: ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:      "index has no file, item func returns state and SkipDir",
			index:     indexNoFile("target", "item"),
			source:    sourceDir("source", "pkg", "item"),
			itemFunc:  itemFunc{state: file.LinkState("item/func/dest", file.TypeFile), err: fs.SkipDir},
			wantErr:   fs.SkipDir,
			wantState: ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:      "index has no file, item func reports error",
			index:     indexNoFile("target", "item"),
			source:    sourceFile("source", "pkg", "item"),
			itemFunc:  itemFunc{err: errFromItemFunc},
			wantErr:   errFromItemFunc,
			wantState: nil,
		},
		{
			desc:      "index has dir, item func returns state",
			index:     indexDir("target", "item"),
			source:    sourceFile("source", "pkg", "item"),
			itemFunc:  itemFunc{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantState: ptr(file.LinkState("item/func/dest", file.TypeFile)),
		},
		{
			desc:      "index state is link, item func returns state",
			index:     indexLink("target", "item", "index/dest", file.TypeFile),
			source:    sourceFile("source", "pkg", "item"),
			itemFunc:  itemFunc{state: file.DirState()},
			wantState: ptr(file.DirState()),
		},
		{
			desc:      "index state is file, item func reports error",
			index:     indexFile("target", "item"),
			source:    sourceFile("source", "pkg", "item"),
			itemFunc:  itemFunc{err: errFromItemFunc},
			wantErr:   errFromItemFunc,
			wantState: nil,
		},
		{
			desc:      "index has dir, item func reports error",
			index:     indexDir("target", "item"),
			source:    sourceFile("source", "pkg", "item"),
			itemFunc:  itemFunc{err: errFromItemFunc},
			wantErr:   errFromItemFunc,
			wantState: nil,
		},
		{
			desc:      "index has link, item func reports error",
			index:     indexLink("target", "item", "index/dest", file.TypeFile),
			source:    sourceFile("source", "pkg", "item"),
			itemFunc:  itemFunc{err: errFromItemFunc},
			wantErr:   errFromItemFunc,
			wantState: nil,
		},
	},
}

var earlyExitSuite = packageOpSuite{
	name: "EarlyExit",
	tests: []packageOpTest{
		{
			desc:    "visit package dir with no walk error",
			source:  sourceDir("source", "pkg", "."),
			wantErr: nil,
		},
		{
			desc:    "visit package dir with walk error",
			source:  sourceDirError("source", "pkg", ".", errPassedToVisitFunc),
			wantErr: errPassedToVisitFunc,
		},
		{
			desc:    "visit item with walk error",
			source:  sourceDirError("source", "pkg", "item", errPassedToVisitFunc),
			wantErr: errPassedToVisitFunc,
		},
		{
			desc:    "visit item with index error",
			index:   indexError("target", "item", errFromIndexState),
			source:  sourceDir("source", "pkg", "item"),
			wantErr: errFromIndexState,
		},
	},
}

type packageOpSuite struct {
	name  string
	tests []packageOpTest
}

func (s packageOpSuite) run(t *testing.T) {
	t.Run(s.name, func(t *testing.T) {
		for _, test := range s.tests {
			test.run(t)
		}
	})
}

func (test packageOpTest) run(t *testing.T) {
	t.Run(test.desc, func(t *testing.T) {
		var logbuf bytes.Buffer
		logger := log.Logger(&logbuf, duftest.LogLevel)
		defer duftest.Dump(t, "log", &logbuf)

		pkgOp := NewInstallOp(test.SourceDir(), test.Package())

		visit := pkgOp.VisitFunc(test.TargetDir(), &test.index, test.itemFunc.Call, logger)

		gotErr := visit(test.NameArg(), test.EntryArg(), test.ErrArg())

		if !errors.Is(gotErr, test.wantErr) {
			t.Fatalf("error:\n got: %v\nwant: %v", gotErr, test.wantErr)
		}

		test.index.checkSetState(t, test.TargetName(), test.wantState)
		test.itemFunc.checkCall(t, test.SourceItem(), test.TargetItem())
	})
}

func (test packageOpTest) EntryArg() fs.DirEntry {
	return errfs.DirEntry(test.source.item, test.source.fmode)
}
func (test packageOpTest) ErrArg() error {
	return test.source.errArg
}

func (test packageOpTest) NameArg() string {
	return path.Join(test.source.source, test.source.pkg, test.source.item)
}

func (test packageOpTest) Package() string {
	return test.source.pkg
}

func (test packageOpTest) SourceDir() string {
	return test.source.source
}

func (test packageOpTest) TargetItem() TargetItem {
	return NewTargetItem(test.index.target, test.index.item, test.index.state)
}

func (test packageOpTest) TargetDir() string {
	return test.index.target
}

func (test packageOpTest) TargetName() string {
	return path.Join(test.index.target, test.index.item)
}

func (test packageOpTest) SourceItem() SourceItem {
	return NewSourceItem(test.source.source, test.source.pkg, test.source.item, test.source.ftype)
}

type sourceState struct {
	source string      // The source part of the source item's name.
	pkg    string      // The package part of the source item's name.
	item   string      // The item part of the source item's name.
	ftype  file.Type   // The source item's file type. Must match fmode.
	fmode  fs.FileMode // The source item's file mode. Must match ftype.
	errArg error       // Error passed to PackageOp's visit func.
}

func sourceDir(source, pkg, item string) sourceState {
	return sourceState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeDir,
		fmode:  fs.ModeDir | 0755,
	}
}

func sourceDirError(source, pkg, item string, err error) sourceState {
	return sourceState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeDir,
		fmode:  fs.ModeDir | 0755,
		errArg: err,
	}
}

func sourceFile(source, pkg, item string) sourceState {
	return sourceState{
		source: source,
		pkg:    pkg,
		item:   item,
		ftype:  file.TypeFile,
		fmode:  0644,
	}
}

type indexItem struct {
	target   string      // The target part of the item's name.
	item     string      // The item part of the item's name.
	state    file.State  // State to return from State.
	err      error       // Error to return from State.
	gotName  string      // Name passed to SetState.
	gotState *file.State // State passed to SetState.
}

func indexNoFile(target, item string) indexItem {
	return indexItem{target: target, item: item, state: file.NoFileState()}
}

func indexFile(target, item string) indexItem {
	return indexItem{target: target, item: item, state: file.FileState()}
}

func indexDir(target, item string) indexItem {
	return indexItem{target: target, item: item, state: file.DirState()}
}

func indexLink(target, item, dest string, destType file.Type) indexItem {
	return indexItem{target: target, item: item, state: file.LinkState(dest, destType)}
}

func indexError(target, item string, err error) indexItem {
	return indexItem{target: target, item: item, err: err}
}

func (s *indexItem) State(string, *slog.Logger) (file.State, error) {
	return s.state, s.err
}

func (s *indexItem) SetState(name string, state file.State, _ *slog.Logger) {
	s.gotName = name
	s.gotState = &state
}

func (s *indexItem) checkSetState(t *testing.T, wantName string, wantState *file.State) {
	if wantState == nil {
		if s.gotState != nil {
			t.Errorf("unwanted call to index.SetState():\n name: %q\nstate: %s",
				s.gotName, s.gotState)
		}
		return
	}
	if wantName != s.gotName {
		t.Errorf("index.SetState() name arg: got %q, want %q", s.gotName, wantName)
	}
	if diff := cmp.Diff(wantState, s.gotState); diff != "" {
		t.Errorf("index.SetState() state arg:\n%s", diff)
	}
}

type itemFunc struct {
	state     file.State  // State returned from itemFunc
	err       error       // Error returned from itemFunc
	gotSource *SourceItem // SourceItem passed to itemFunc.
	gotTarget *TargetItem // TargetItem passed to itemFunc.
}

func (i *itemFunc) Call(gotSource SourceItem, gotTarget TargetItem, l *slog.Logger) (file.State, error) {
	i.gotSource, i.gotTarget = &gotSource, &gotTarget
	return i.state, i.err
}

func (i *itemFunc) checkCall(t *testing.T, wantSource SourceItem, wantTarget TargetItem) {
	if i.gotSource == nil {
		return
	}
	if diff := cmp.Diff(&wantSource, i.gotSource); diff != "" {
		t.Errorf("item func source arg:\n%s", diff)
	}
	if diff := cmp.Diff(&wantTarget, i.gotTarget); diff != "" {
		t.Errorf("item func target arg:\n%s", diff)
	}
}

func ptr[T any](v T) *T {
	return &v
}
