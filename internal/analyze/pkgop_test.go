package analyze

import (
	"errors"
	"io/fs"
	"path"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
)

type indexValue struct {
	state *file.State
	err   error
}
type testIndex map[string]indexValue

func (i testIndex) State(name string) (*file.State, error) {
	v, ok := i[name]
	if !ok {
		return nil, fs.ErrInvalid
	}
	return v.state, v.err
}

func (i testIndex) SetState(name string, state *file.State) {
	i[name] = indexValue{state: state}
}

type itemOpFunc func(name string, entry fs.DirEntry, inState *file.State) (*file.State, error)

func (f itemOpFunc) Apply(name string, entry fs.DirEntry, inState *file.State) (*file.State, error) {
	return f(name, entry, inState)
}

func TestPkgOpApplyItemOp(t *testing.T) {
	const (
		target         = "path/to/target"
		source         = "path/to/source"
		targetToSource = "../source"
		pkg            = "pkg"
		item           = "item"
	)
	anItemOpError := errors.New("error returned from item op")

	tests := map[string]struct {
		indexState  *file.State // Initial state of the item in the index
		itemOpState *file.State // State returned by the item op
		itemOpError error       // Error returned by item op
		wantErr     error       // Error returned by visit func
		wantState   *file.State // Index's state for the item after visit func
	}{
		"no index state, item op returns state": {
			indexState:  nil,
			itemOpState: &file.State{Type: fs.ModeSymlink, Dest: "dest/from/item/op"},
			wantState:   &file.State{Type: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"no index state, item op returns state and SkipDir error": {
			indexState:  nil,
			itemOpState: &file.State{Type: fs.ModeSymlink, Dest: "dest/from/item/op"},
			itemOpError: fs.SkipDir,
			wantErr:     fs.SkipDir,
			wantState:   &file.State{Type: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"no index state, item op reports error": {
			indexState:  nil,
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   nil,
		},
		"index state is dir, item op returns state": {
			indexState:  &file.State{Type: fs.ModeDir},
			itemOpState: &file.State{Type: fs.ModeSymlink, Dest: "dest/from/item/op"},
			wantState:   &file.State{Type: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"index state is link, item op returns state": {
			indexState:  &file.State{Type: fs.ModeSymlink, Dest: "dest/from/index"},
			itemOpState: &file.State{Type: fs.ModeDir},
			wantState:   &file.State{Type: fs.ModeDir},
		},
		"index state is file, item op reports error": {
			indexState:  &file.State{Type: 0},
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   &file.State{Type: 0},
		},
		"index state is dir, item op reports error": {
			indexState:  &file.State{Type: fs.ModeDir},
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   &file.State{Type: fs.ModeDir},
		},
		"index state is link, item op reports error": {
			indexState:  &file.State{Type: fs.ModeSymlink, Dest: "dest/from/index"},
			itemOpState: nil,
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   &file.State{Type: fs.ModeSymlink, Dest: "dest/from/index"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sourcePkg := path.Join(source, pkg)
			sourcePkgItem := path.Join(sourcePkg, item)

			var gotItemOpCall bool
			itemOp := itemOpFunc(func(gotName string, gotEntry fs.DirEntry, gotState *file.State) (*file.State, error) {
				gotItemOpCall = true
				if gotName != sourcePkgItem {
					t.Errorf("item op: got name %q, want %q", gotName, sourcePkgItem)
				}
				if !cmp.Equal(gotState, test.indexState) {
					t.Errorf("item op: got state %v, want %v", gotState, test.indexState)
				}
				return test.itemOpState, test.itemOpError
			})

			targetItem := path.Join(target, item)

			testIndex := testIndex{targetItem: indexValue{state: test.indexState}}

			pkgOp := NewPkgOp(sourcePkg, itemOp)

			visit := pkgOp.VisitFunc(target, testIndex)

			gotErr := visit(sourcePkgItem, nil, nil)

			if !gotItemOpCall {
				t.Errorf("no call to item op")
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\n got%v\nwant %v", gotErr, test.wantErr)
			}

			var gotState *file.State
			if v, ok := testIndex[targetItem]; ok {
				gotState = v.state
			}
			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("index[%q] after visit:\n got%v\nwant %v",
					targetItem, gotState, test.wantState)
			}
		})
	}
}

// Tests of situations that produce errors
// and preclude setting the desired state or calling the item op.
func TestPkgOpWalkFuncError(t *testing.T) {
	var (
		anIndexError = errors.New("error returned from index.Desired")
		aWalkError   = errors.New("error passed to visit func")
	)

	tests := map[string]struct {
		item      string // The item being visited, or . to visit the pkg dir
		walkErr   error  // The error passed to visit func
		indexErr  error  // The error returned from index.Desired
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

			targetItem := path.Join(target, test.item)
			sourcePkg := path.Join(source, pkg)
			sourcePkgItem := path.Join(sourcePkg, test.item)

			testIndex := testIndex{targetItem: indexValue{err: test.indexErr}}

			pkgOp := NewPkgOp(sourcePkg, nil)

			visit := pkgOp.VisitFunc(target, testIndex)

			gotErr := visit(sourcePkgItem, nil, test.walkErr)

			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("error:\n got: %v\nwant: %v", gotErr, test.wantError)
			}

			var gotState *file.State
			if v, ok := testIndex[test.item]; ok {
				gotState = v.state
			}
			if gotState != nil {
				t.Errorf("recorded state:\n got: %v\nwant: %v", gotState, nil)
			}
		})
	}
}
