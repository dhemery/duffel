package plan

import (
	"errors"
	"io/fs"
	"path"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
)

type adviseFunc func(string, string, fs.DirEntry, *file.State) (*file.State, error)

func (af adviseFunc) Apply(pkg, itemName string, d fs.DirEntry, priorGoal *file.State) (*file.State, error) {
	return af(pkg, itemName, d, priorGoal)
}

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

func TestPkgAnalystVisitPath(t *testing.T) {
	const (
		target         = "path/to/target"
		source         = "path/to/source"
		targetToSource = "../source"
		pkg            = "pkg"
		itemName       = "item"
	)
	anAdvisorError := errors.New("error returned from advisor")

	tests := map[string]struct {
		indexState        *file.State // The initial state of the item in the index
		advisedState      *file.State // The advice returned by the advisor
		advisorError      error       // Error returned by advisor
		wantErr           error       // Error returned by VisitPath
		wantRecordedState *file.State // The recorded state for the item after VisitPath
		skip              string      // Reason for skipping test
	}{
		"no index state, advisor advises": {
			indexState:        nil,
			advisedState:      &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
			wantRecordedState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
		},
		"no index state, advisor reports error": {
			indexState:        nil,
			advisorError:      anAdvisorError,
			wantErr:           anAdvisorError,
			wantRecordedState: nil,
		},
		"index state is dir, advisor advises": {
			indexState:        &file.State{Mode: fs.ModeDir | 0o755},
			advisedState:      &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
			wantRecordedState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
		},
		"index state is link, advisor advises": {
			indexState:        &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/index"},
			advisedState:      &file.State{Mode: fs.ModeDir | 0o755},
			wantRecordedState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"index state is file, advisor reports error": {
			indexState:        &file.State{Mode: 0o644},
			advisorError:      anAdvisorError,
			wantErr:           anAdvisorError,
			wantRecordedState: nil,
		},
		"index state is dir, advisor reports error": {
			indexState:        &file.State{Mode: fs.ModeDir | 0o755},
			advisorError:      anAdvisorError,
			wantErr:           anAdvisorError,
			wantRecordedState: nil,
		},
		"index state is link, advisor reports error": {
			indexState:        &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/index"},
			advisedState:      nil,
			advisorError:      anAdvisorError,
			wantErr:           anAdvisorError,
			wantRecordedState: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}
			gotAdvisorCall := false

			testAdvisor := adviseFunc(func(gotPkg, gotItem string, gotEntry fs.DirEntry, gotState *file.State) (*file.State, error) {
				gotAdvisorCall = true
				if gotPkg != pkg {
					t.Errorf("advisor: want pkg %q, got %q", pkg, gotPkg)
				}
				if gotItem != itemName {
					t.Errorf("advisor: want item %q, got %q", itemName, gotItem)
				}
				if !reflect.DeepEqual(gotState, test.indexState) {
					t.Errorf("advisor: want state %v, got %v", test.indexState, gotState)
				}
				return test.advisedState, test.advisorError
			})

			testIndex := &testIndex{
				t:            t,
				name:         itemName,
				initialState: test.indexState,
			}

			pa := NewPkgWalker(nil, target, source, pkg, testIndex, testAdvisor)

			sourcePkgItem := path.Join(source, pkg, itemName)
			gotErr := pa.VisitPath(sourcePkgItem, nil, nil)

			if !gotAdvisorCall {
				t.Errorf("no call to advisor")
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
// and preclude setting the desired state or calling the item advisor.
func TestPkgAnalystVisitPathError(t *testing.T) {
	var (
		anIndexError = errors.New("error returned from index.Desired")
		aWalkError   = errors.New("error passed to VisitPath")
	)

	tests := map[string]struct {
		item      string // The item being visited, or . to visit the pkg dir
		walkErr   error  // The error passed to VisitPath
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
		"cannot get desired state": {
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

			pa := NewPkgWalker(nil, target, source, pkg, testIndex, nil)

			walkPath := path.Join(source, pkg, test.item)

			gotErr := pa.VisitPath(walkPath, nil, test.walkErr)

			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("want error %q, got %q", test.wantError, gotErr)
			}

			if testIndex.recordedState != nil {
				t.Errorf("recorded state:\nwant nil\n got %v", testIndex.recordedState)
			}
		})
	}
}
