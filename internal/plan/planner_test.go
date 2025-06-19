package plan

import (
	"errors"
	"io/fs"
	"path"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
)

type testIndex struct {
	t             *testing.T
	name          string
	initialState  *file.State
	err           error
	recordedState *file.State
}

func (s *testIndex) Desired(name string) (*file.State, error) {
	if name != s.name {
		s.t.Errorf("desired: want name %q, got %q", s.name, name)
	}
	return s.initialState, s.err
}

func (s *testIndex) SetDesired(name string, state *file.State) {
	if name != s.name {
		s.t.Errorf("set desired: want name %q, got %q", s.name, name)
	}
	s.recordedState = state
}

func TestPkgOpApplyItemFunc(t *testing.T) {
	const (
		target         = "path/to/target"
		source         = "path/to/source"
		targetToSource = "../source"
		pkg            = "pkg"
		itemName       = "item"
	)
	anItemOpError := errors.New("error returned from item op")

	tests := map[string]struct {
		indexState        *file.State // Initial state of the item in the index
		itemOpState       *file.State // State returned by the item op
		itemOpError       error       // Error returned by item op
		wantErr           error       // Error returned by visit func
		wantRecordedState *file.State // Index's state for the item after visit func
		skip              string      // Reason for skipping test
	}{
		"no index state, item op returns state": {
			indexState:        nil,
			itemOpState:       &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
			wantRecordedState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"no index state, item op reports error": {
			indexState:        nil,
			itemOpError:       anItemOpError,
			wantErr:           anItemOpError,
			wantRecordedState: nil,
		},
		"index state is dir, item op returns state": {
			indexState:        &file.State{Mode: fs.ModeDir | 0o755},
			itemOpState:       &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
			wantRecordedState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/item/op"},
		},
		"index state is link, item op returns state": {
			indexState:        &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/index"},
			itemOpState:       &file.State{Mode: fs.ModeDir | 0o755},
			wantRecordedState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"index state is file, item op reports error": {
			indexState:        &file.State{Mode: 0o644},
			itemOpError:       anItemOpError,
			wantErr:           anItemOpError,
			wantRecordedState: nil,
		},
		"index state is dir, item op reports error": {
			indexState:        &file.State{Mode: fs.ModeDir | 0o755},
			itemOpError:       anItemOpError,
			wantErr:           anItemOpError,
			wantRecordedState: nil,
		},
		"index state is link, item op reports error": {
			indexState:        &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/index"},
			itemOpState:       nil,
			itemOpError:       anItemOpError,
			wantErr:           anItemOpError,
			wantRecordedState: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}
			gotItemOpCall := false

			itemOp := func(gotPkg, gotItem string, gotEntry fs.DirEntry, gotState *file.State) (*file.State, error) {
				gotItemOpCall = true
				if gotPkg != pkg {
					t.Errorf("item op: want pkg %q, got %q", pkg, gotPkg)
				}
				if gotItem != itemName {
					t.Errorf("item op: want item %q, got %q", itemName, gotItem)
				}
				if !reflect.DeepEqual(gotState, test.indexState) {
					t.Errorf("item op: want state %v, got %v", test.indexState, gotState)
				}
				return test.itemOpState, test.itemOpError
			}

			testIndex := &testIndex{
				t:            t,
				name:         itemName,
				initialState: test.indexState,
			}

			pkgOp := PkgOp{Pkg: pkg, Apply: itemOp}

			visit := pkgOp.VisitFunc(source, testIndex)

			visitPath := path.Join(source, pkg, itemName)
			gotErr := visit(visitPath, nil, nil)

			if !gotItemOpCall {
				t.Errorf("no call to item op")
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			if !reflect.DeepEqual(testIndex.recordedState, test.wantRecordedState) {
				t.Errorf("desired state:\nwant %v\ngot  %v", test.wantRecordedState, testIndex.recordedState)
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

			testIndex := &testIndex{
				t:             t,
				name:          test.item,
				err:           test.indexErr,
				initialState:  &file.State{Mode: fs.ModeSymlink, Dest: "bogus/initial/state"},
				recordedState: nil,
			}

			pkgOp := PkgOp{Pkg: pkg, Apply: nil}

			visit := pkgOp.VisitFunc(source, testIndex)

			visitPath := path.Join(source, pkg, test.item)
			gotErr := visit(visitPath, nil, test.walkErr)

			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("want error %q, got %q", test.wantError, gotErr)
			}

			if testIndex.recordedState != nil {
				t.Errorf("state in index:\nwant nil\n got %v", testIndex.recordedState)
			}
		})
	}
}
