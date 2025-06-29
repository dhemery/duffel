package plan

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

func (i testIndex) Get(name string) (*file.State, error) {
	v, ok := i[name]
	if !ok {
		return nil, fs.ErrInvalid
	}
	return v.state, v.err
}

func (i testIndex) Set(name string, state *file.State) {
	i[name] = indexValue{state: state}
}

func TestPkgOpApplyItemOp(t *testing.T) {
	const (
		target         = "path/to/target"
		source         = "path/to/source"
		targetToSource = "../source"
		pkg            = "pkg"
		itemName       = "item"
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
			itemOpState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
			wantState:   &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"no index state, item op returns state and SkipDir error": {
			indexState:  nil,
			itemOpState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
			itemOpError: fs.SkipDir,
			wantErr:     fs.SkipDir,
			wantState:   &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"no index state, item op reports error": {
			indexState:  nil,
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   nil,
		},
		"index state is dir, item op returns state": {
			indexState:  &file.State{Mode: fs.ModeDir | 0o755},
			itemOpState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
			wantState:   &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"index state is link, item op returns state": {
			indexState:  &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/index"},
			itemOpState: &file.State{Mode: fs.ModeDir | 0o755},
			wantState:   &file.State{Mode: fs.ModeDir | 0o755},
		},
		"index state is file, item op reports error": {
			indexState:  &file.State{Mode: 0o644},
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   &file.State{Mode: 0o644},
		},
		"index state is dir, item op reports error": {
			indexState:  &file.State{Mode: fs.ModeDir | 0o755},
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   &file.State{Mode: fs.ModeDir | 0o755},
		},
		"index state is link, item op reports error": {
			indexState:  &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/index"},
			itemOpState: nil,
			itemOpError: anItemOpError,
			wantErr:     anItemOpError,
			wantState:   &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/index"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotItemOpCall := false

			itemOp := func(gotPkg, gotItem string, gotEntry fs.DirEntry, gotState *file.State) (*file.State, error) {
				gotItemOpCall = true
				if gotPkg != pkg {
					t.Errorf("item op: got pkg %q, want %q", gotPkg, pkg)
				}
				if gotItem != itemName {
					t.Errorf("item op: got item %q, want %q", gotItem, itemName)
				}
				if !cmp.Equal(gotState, test.indexState) {
					t.Errorf("item op: got state %v, want %v", gotState, test.indexState)
				}
				return test.itemOpState, test.itemOpError
			}

			testIndex := testIndex{itemName: indexValue{state: test.indexState}}

			pkgOp := PkgOp{Pkg: pkg, Apply: itemOp}

			visit := pkgOp.VisitFunc(source, testIndex)

			visitPath := path.Join(source, pkg, itemName)
			gotErr := visit(visitPath, nil, nil)

			if !gotItemOpCall {
				t.Errorf("no call to item op")
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\n got%v\nwant %v", gotErr, test.wantErr)
			}

			var gotState *file.State
			if v, ok := testIndex[itemName]; ok {
				gotState = v.state
			}
			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("recorded state:\n got%v\nwant %v", gotState, test.wantState)
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

			testIndex := testIndex{test.item: indexValue{err: test.indexErr}}

			pkgOp := PkgOp{Pkg: pkg, Apply: nil}

			visit := pkgOp.VisitFunc(source, testIndex)

			visitPath := path.Join(source, pkg, test.item)
			gotErr := visit(visitPath, nil, test.walkErr)

			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("error: got %q, want %q", gotErr, test.wantError)
			}

			var gotState *file.State
			if v, ok := testIndex[test.item]; ok {
				gotState = v.state
			}
			if gotState != nil {
				t.Errorf("recorded state:\n got %v\nwant nil", gotState)
			}
		})
	}
}
