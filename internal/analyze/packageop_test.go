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
		return file.State{}, fs.ErrInvalid
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
	anItemFuncError := errors.New("error returned from item op")

	tests := map[string]struct {
		indexState    file.State // Initial state of the item in the index.
		itemFuncState file.State // State returned by ItemFunc.
		itemFuncError error      // Error returned by ItemFunc.
		wantErr       error      // Error returned by visit func.
		wantState     file.State // Index's state for the item after visit func.
	}{
		"no index state, item op returns state": {
			indexState:    file.State{},
			itemFuncState: file.State{Type: file.TypeSymlink, Dest: "dest/from/item/op"},
			wantState:     file.State{Type: file.TypeSymlink, Dest: "dest/from/item/op"},
		},
		"no index state, item op returns state and SkipDir error": {
			indexState:    file.State{},
			itemFuncState: file.State{Type: file.TypeSymlink, Dest: "dest/from/item/op"},
			itemFuncError: fs.SkipDir,
			wantErr:       fs.SkipDir,
			wantState:     file.State{Type: file.TypeSymlink, Dest: "dest/from/item/op"},
		},
		"no index state, item op reports error": {
			indexState:    file.State{},
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
			wantState:     file.State{},
		},
		"index state is dir, item op returns state": {
			indexState:    file.State{Type: file.TypeDir},
			itemFuncState: file.State{Type: file.TypeSymlink, Dest: "dest/from/item/op"},
			wantState:     file.State{Type: file.TypeSymlink, Dest: "dest/from/item/op"},
		},
		"index state is link, item op returns state": {
			indexState:    file.State{Type: file.TypeSymlink, Dest: "dest/from/index"},
			itemFuncState: file.State{Type: file.TypeDir},
			wantState:     file.State{Type: file.TypeDir},
		},
		"index state is file, item op reports error": {
			indexState:    file.State{Type: 0},
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
			wantState:     file.State{Type: 0},
		},
		"index state is dir, item op reports error": {
			indexState:    file.State{Type: file.TypeDir},
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
			wantState:     file.State{Type: file.TypeDir},
		},
		"index state is link, item op reports error": {
			indexState:    file.State{Type: file.TypeSymlink, Dest: "dest/from/index"},
			itemFuncState: file.State{},
			itemFuncError: anItemFuncError,
			wantErr:       anItemFuncError,
			wantState:     file.State{Type: file.TypeSymlink, Dest: "dest/from/index"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var logbuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logbuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

			sourceEntry := errfs.DirEntry(item, fs.ModeDir|0o755)
			sourcePath := NewSourcePath(source, pkg, item)
			sourceItem := SourceItem{sourcePath, file.TypeOf(sourceEntry.Type())}
			targetPath := NewTargetPath(target, item)
			targetItem := TargetItem{targetPath, test.indexState}

			var gotItemFuncCall bool
			fakeItemFunc := func(gotSourceItem SourceItem, gotTargetItem TargetItem, l *slog.Logger) (file.State, error) {
				gotItemFuncCall = true
				if diff := cmp.Diff(gotSourceItem, sourceItem); diff != "" {
					t.Errorf("item op: source item:\n%s", diff)
				}
				if diff := cmp.Diff(gotTargetItem, targetItem); diff != "" {
					t.Errorf("item op: target item:\n%s", diff)
				}
				return test.itemFuncState, test.itemFuncError
			}

			testIndex := testIndex{targetItem.Path.String(): indexValue{state: test.indexState}}

			pkgOp := NewPackageOp(source, pkg, GoalInstall)

			visit := pkgOp.VisitFunc(target, testIndex, fakeItemFunc, logger)

			gotErr := visit(sourceItem.Path.String(), sourceEntry, nil)

			if !gotItemFuncCall {
				t.Errorf("no call to item op")
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
// and preclude setting the desired state or calling the item op.
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
			logger := slog.New(slog.NewTextHandler(&logbuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

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
