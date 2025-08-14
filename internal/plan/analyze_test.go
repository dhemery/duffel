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
	desc           string          // Description of the test
	index          targetState     // The state of the target item in the index before analyzing.
	source         sourceState     // The state of the source item to analyze.
	itemFunc       itemFuncResults // The results returned by itemFunc.
	wantErr        error           // Error result.
	wantIndexState file.State      // The state of the target item in the index after analyzing.
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
			desc:           "index has no file, item func returns state",
			index:          targetNoFile("target", "item"),
			source:         sourceFile("source", "pkg", "item"),
			itemFunc:       itemFuncResults{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantIndexState: file.LinkState("item/func/dest", file.TypeFile),
		},
		{
			desc:           "index has no file, item func returns state and SkipDir",
			index:          targetNoFile("target", "item"),
			source:         sourceDir("source", "pkg", "item"),
			itemFunc:       itemFuncResults{state: file.LinkState("item/func/dest", file.TypeFile), err: fs.SkipDir},
			wantErr:        fs.SkipDir,
			wantIndexState: file.LinkState("item/func/dest", file.TypeFile),
		},
		{
			desc:           "index has no file, item func reports error",
			index:          targetNoFile("target", "item"),
			source:         sourceFile("source", "pkg", "item"),
			itemFunc:       itemFuncResults{err: errFromItemFunc},
			wantErr:        errFromItemFunc,
			wantIndexState: file.NoFileState(),
		},
		{
			desc:           "index has dir, item func returns state",
			index:          targetDir("target", "item"),
			source:         sourceFile("source", "pkg", "item"),
			itemFunc:       itemFuncResults{state: file.LinkState("item/func/dest", file.TypeFile)},
			wantIndexState: file.LinkState("item/func/dest", file.TypeFile),
		},
		{
			desc:           "index state is link, item func returns state",
			index:          targetLink("target", "item", "index/dest", file.TypeFile),
			source:         sourceFile("source", "pkg", "item"),
			itemFunc:       itemFuncResults{state: file.DirState()},
			wantIndexState: file.DirState(),
		},
		{
			desc:           "index state is file, item func reports error",
			index:          targetFile("target", "item"),
			source:         sourceFile("source", "pkg", "item"),
			itemFunc:       itemFuncResults{err: errFromItemFunc},
			wantErr:        errFromItemFunc,
			wantIndexState: file.FileState(),
		},
		{
			desc:           "index has dir, item func reports error",
			index:          targetDir("target", "item"),
			source:         sourceFile("source", "pkg", "item"),
			itemFunc:       itemFuncResults{err: errFromItemFunc},
			wantErr:        errFromItemFunc,
			wantIndexState: file.DirState(),
		},
		{
			desc:           "index has link, item func reports error",
			index:          targetLink("target", "item", "index/dest", file.TypeFile),
			source:         sourceFile("source", "pkg", "item"),
			itemFunc:       itemFuncResults{err: errFromItemFunc},
			wantErr:        errFromItemFunc,
			wantIndexState: file.LinkState("index/dest", file.TypeFile),
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
			index:   targetError("target", "item", errFromIndexState),
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

		testItemFunc := func(gotSource SourceItem, gotTarget TargetItem, l *slog.Logger) (file.State, error) {
			wantSource := test.source.sourceItem()
			if diff := cmp.Diff(wantSource, gotSource); diff != "" {
				t.Errorf("item func source arg:\n%s", diff)
			}
			wantTarget := test.index.targetItem()
			if diff := cmp.Diff(wantTarget, gotTarget); diff != "" {
				t.Errorf("item func target arg:\n%s", diff)
			}
			return test.itemFunc.state, test.itemFunc.err
		}

		testIndex := testIndex{test.index.path(): indexValue{state: test.index.state, err: test.index.err}}

		pkgOp := NewInstallOp(test.source.source, test.source.pkg)

		visit := pkgOp.VisitFunc(test.index.target, testIndex, testItemFunc, logger)

		gotErr := visit(test.source.nameArg(), test.source.entryArg(), test.source.errArg)

		if !errors.Is(gotErr, test.wantErr) {
			t.Fatalf("error:\n got: %v\nwant: %v", gotErr, test.wantErr)
		}

		var gotState file.State
		if v, ok := testIndex[test.index.path()]; ok {
			gotState = v.state
		}
		if diff := cmp.Diff(test.wantIndexState, gotState); diff != "" {
			t.Errorf("index[%q] after visit:\n%s", test.index.path(), gotState)
		}
	})
}

type indexValue struct {
	state file.State
	err   error
}
type testIndex map[string]indexValue

func (i testIndex) State(name string, l *slog.Logger) (file.State, error) {
	v, ok := i[name]
	if !ok {
		return file.NoFileState(), fs.ErrInvalid
	}
	return v.state, v.err
}

func (i testIndex) SetState(name string, state file.State, l *slog.Logger) {
	i[name] = indexValue{state: state}
}

type sourceState struct {
	source string      // The source part of the source item's name.
	pkg    string      // The package part of the source item's name.
	item   string      // The item part of the source item's name.
	ftype  file.Type   // The source item's file type. Must match fmode.
	fmode  fs.FileMode // The source item's file mode. Must match ftype.
	errArg error       // Error passed to PackageOp's visit func.
}

func (s sourceState) sourceItem() SourceItem {
	return NewSourceItem(s.source, s.pkg, s.item, s.ftype)
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

func (s sourceState) nameArg() string {
	return path.Join(s.source, s.pkg, s.item)
}

func (s sourceState) entryArg() fs.DirEntry {
	return errfs.DirEntry(s.item, s.fmode)
}

type targetState struct {
	target string     // The target part of the target item's name.
	item   string     // The item part of the target item's name.
	state  file.State // State to return from index.State.
	err    error      // Error to return from index.State.
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

func (t targetState) path() string {
	return path.Join(t.target, t.item)
}

func (t targetState) targetItem() TargetItem {
	return NewTargetItem(t.target, t.item, t.state)
}

type itemFuncResults struct {
	state file.State
	err   error
}
