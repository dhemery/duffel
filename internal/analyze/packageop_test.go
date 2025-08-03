package analyze_test

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"path"
	"testing"

	. "github.com/dhemery/duffel/internal/analyze"
	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/log"

	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
)

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

func TestPackageOpItemFunc(t *testing.T) {
	const (
		target         = "path/to/target"
		source         = "path/to/source"
		targetToSource = "../source"
		pkg            = "pkg"
		item           = "item"
	)
	anItemFuncError := errors.New("error returned from item func")

	tests := map[string]struct {
		indexState    file.State // Initial state of the item in the index.
		itemFuncState file.State // State returned by ItemFunc.
		itemFuncError error      // Error returned by ItemFunc.
		wantErr       error      // Error returned by visit func.
		wantState     file.State // Index's state for the item after visit func.
	}{
		"index has no file, item func returns state": {
			indexState:    file.NoFileState(),
			itemFuncState: file.LinkState("item/func/dest", file.TypeFile),
			wantState:     file.LinkState("item/func/dest", file.TypeFile),
		},
		"index has no file, item func returns state and SkipDir": {
			indexState:    file.NoFileState(),
			itemFuncState: file.LinkState("item/func/dest", file.TypeFile),
			itemFuncError: fs.SkipDir,
			wantErr:       fs.SkipDir,
			wantState:     file.LinkState("item/func/dest", file.TypeFile),
		},
		"index has no file, item func reports error": {
			indexState:    file.NoFileState(),
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
		},
		"index has dir, item func returns state": {
			indexState:    file.DirState(),
			itemFuncState: file.LinkState("item/func/dest", file.TypeFile),
			wantState:     file.LinkState("item/func/dest", file.TypeFile),
		},
		"index state is link, item func returns state": {
			indexState:    file.LinkState("index/dest", file.TypeFile),
			itemFuncState: file.DirState(),
			wantState:     file.DirState(),
		},
		"index state is file, item func reports error": {
			indexState:    file.FileState(),
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
			wantState:     file.FileState(),
		},
		"index has dir, item func reports error": {
			indexState:    file.DirState(),
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
			wantState:     file.DirState(),
		},
		"index has link, item func reports error": {
			indexState:    file.LinkState("index/dest", file.TypeFile),
			itemFuncState: file.NoFileState(),
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
			wantState:     file.LinkState("index/dest", file.TypeFile),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var logbuf bytes.Buffer
			logger := log.Logger(&logbuf, slog.LevelInfo)

			sourceEntry := errfs.DirEntry(item, fs.ModeDir|0o755)
			sourcePath := NewSourcePath(source, pkg, item)
			sourceType, _ := file.TypeOf(sourceEntry.Type())
			sourceItem := SourceItem{sourcePath, sourceType}
			targetPath := NewTargetPath(target, item)
			targetItem := TargetItem{targetPath, test.indexState}

			var gotItemFuncCall bool
			fakeItemFunc := func(gotSourceItem SourceItem, gotTargetItem TargetItem, l *slog.Logger) (file.State, error) {
				gotItemFuncCall = true
				if diff := cmp.Diff(gotSourceItem, sourceItem); diff != "" {
					t.Errorf("item func: source item:\n%s", diff)
				}
				if diff := cmp.Diff(gotTargetItem, targetItem); diff != "" {
					t.Errorf("item func: target item:\n%s", diff)
				}
				return test.itemFuncState, test.itemFuncError
			}

			testIndex := testIndex{targetItem.Path.String(): indexValue{state: test.indexState}}

			pkgOp := NewPackageOp(source, pkg, GoalInstall)

			visit := pkgOp.VisitFunc(target, testIndex, fakeItemFunc, logger)

			gotErr := visit(sourceItem.Path.String(), sourceEntry, nil)

			if !gotItemFuncCall {
				t.Errorf("no call to item func")
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\n got%v\nwant %v", gotErr, test.wantErr)
			}

			var gotState file.State
			if v, ok := testIndex[targetPath.String()]; ok {
				gotState = v.state
			}
			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("index[%q] after visit:\n got%v\nwant %v",
					targetPath, gotState, test.wantState)
			}
			if t.Failed() || testing.Verbose() {
				t.Log("log:\n", logbuf.String())
			}
		})
	}
}

// Tests of situations that produce errors
// and preclude setting the desired state or calling the item func.
func TestPackageOpWalkFuncError(t *testing.T) {
	var (
		anIndexError = errors.New("error returned from index.Desired")
		aWalkError   = errors.New("error passed to visit func")
	)

	tests := map[string]struct {
		item      string // The item being visited, or . to visit the pkg dir.
		walkErr   error  // The error passed to visit func.
		indexErr  error  // The error returned from index.Desired.
		wantError error
	}{
		"pkg dir with no walk error": {
			item:      ".",
			walkErr:   nil,
			wantError: nil,
		},
		"pkg dir with walk error": {
			item:      ".",
			walkErr:   aWalkError,
			wantError: aWalkError,
		},
		"item with walk error": {
			item:      "item",
			walkErr:   aWalkError,
			wantError: aWalkError,
		},
		"item with index error": {
			item:      "item",
			indexErr:  anIndexError,
			wantError: anIndexError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			const (
				source = "path/to/source"
				target = "path/to/target"
				pkg    = "pkg"
			)

			var logbuf bytes.Buffer
			logger := log.Logger(&logbuf, slog.LevelInfo)

			targetItem := path.Join(target, test.item)
			sourcePkg := path.Join(source, pkg)
			sourcePkgItem := path.Join(sourcePkg, test.item)

			testIndex := testIndex{targetItem: indexValue{err: test.indexErr}}

			pkgOp := NewPackageOp(source, pkg, GoalInstall)

			visit := pkgOp.VisitFunc(target, testIndex, nil, logger)

			gotErr := visit(sourcePkgItem, errfs.DirEntry("test-entry", 0o644), test.walkErr)

			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("error:\n got: %v\nwant: %v", gotErr, test.wantError)
			}

			if gotState, ok := testIndex[test.item]; ok {
				t.Errorf("recorded state:\n got: %v\nwant: %v", gotState, nil)
			}
			if t.Failed() || testing.Verbose() {
				t.Log("log:\n", logbuf.String())
			}
		})
	}
}
