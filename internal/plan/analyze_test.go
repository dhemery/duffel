package plan

import (
	"errors"
	"io/fs"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/item"
)

type adviseFunc func(string, string, fs.DirEntry, *file.State) (*file.State, error)

func (af adviseFunc) Advise(pkg, itemName string, d fs.DirEntry, priorGoal *file.State) (*file.State, error) {
	return af(pkg, itemName, d, priorGoal)
}

type fileStateFS map[string]file.State

func (f fileStateFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "fileStateFS.open", Path: name, Err: errors.ErrUnsupported}
}

func (f fileStateFS) Stat(name string) (fs.FileInfo, error) {
	state, ok := f[name]
	if !ok {
		return nil, &fs.PathError{Op: "fileStateFS.stat", Path: name, Err: fs.ErrNotExist}
	}
	if state.Mode&fs.ModeSymlink != 0 {
		return f.Stat(state.Dest)
	}
	return &fileStateInfo{name: path.Base(name), state: state}, nil
}

func (f fileStateFS) Lstat(name string) (fs.FileInfo, error) {
	state, ok := f[name]
	if !ok {
		return nil, &fs.PathError{Op: "fileStateFS.lstat", Path: name, Err: fs.ErrNotExist}
	}
	return &fileStateInfo{name: path.Base(name), state: state}, nil
}

func (f fileStateFS) ReadLink(name string) (string, error) {
	state, ok := f[name]
	if !ok {
		return "", &fs.PathError{Op: "fileStateFS.readlink", Path: name, Err: fs.ErrNotExist}
	}
	if state.Mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: "fileStateFS.readlink", Path: name, Err: fs.ErrInvalid}
	}
	return state.Dest, nil
}

type fileStateInfo struct {
	name  string
	state file.State
}

func (f *fileStateInfo) IsDir() bool {
	return f.Mode().IsDir()
}

func (f *fileStateInfo) ModTime() time.Time {
	return time.Now()
}

func (f *fileStateInfo) Mode() fs.FileMode {
	return f.state.Mode
}

func (f *fileStateInfo) Name() string {
	return f.name
}

func (f *fileStateInfo) Size() int64 {
	return 0
}

func (f *fileStateInfo) Sys() any {
	return nil
}

func TestPkgAnalystVisitPath(t *testing.T) {
	const (
		target         = "path/to/target"
		source         = "path/to/source"
		targetToSource = "../source"
		pkg            = "pkg"
		itemName       = "item"
		dirReadable    = fs.ModeDir | 0o755
		dirUnreadable  = fs.ModeDir | 0o311
		fileReadable   = 0o644
	)
	anAdvisorError := errors.New("error returned from advisor")

	tests := map[string]struct {
		targetItemState  *file.State // The state of the item in the target dir
		advisorAdvice    *file.State // The advice returned by the advisor
		advisorError     error       // Error returned by advisor
		wantErr          error       // Error returned by VisitPath
		wantDesiredState *file.State // The recorded desired state for the item after VisitPath
		skip             string      // Reason for skipping test
	}{
		"no target item, advisor advises": {
			targetItemState:  nil,
			advisorAdvice:    &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
			wantDesiredState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
		},
		"no target item, reports error": {
			targetItemState:  nil,
			advisorError:     anAdvisorError,
			wantErr:          anAdvisorError,
			wantDesiredState: nil,
		},
		"target item is dir, advisor advises": {
			targetItemState:  &file.State{Mode: fs.ModeDir | 0o755},
			advisorAdvice:    &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
			wantDesiredState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
		},
		"target item is link, advisor advises": {
			targetItemState:  &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/stat"},
			advisorAdvice:    &file.State{Mode: fs.ModeDir | 0o755},
			wantDesiredState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"target item is file, advisor reports error": {
			targetItemState:  &file.State{Mode: 0o644},
			advisorError:     anAdvisorError,
			wantErr:          anAdvisorError,
			wantDesiredState: &file.State{Mode: 0o644},
		},
		"target item is dir, advisor reports error": {
			targetItemState:  &file.State{Mode: fs.ModeDir | 0o755},
			advisorError:     anAdvisorError,
			wantErr:          anAdvisorError,
			wantDesiredState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"target item is link, advisor reports error": {
			targetItemState:  &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/stat"},
			advisorAdvice:    nil,
			advisorError:     anAdvisorError,
			wantErr:          anAdvisorError,
			wantDesiredState: &file.State{Mode: fs.ModeSymlink, Dest: "dest/from/stat"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}
			gotAdvisorCall := false

			advisor := adviseFunc(func(gotPkg, gotItem string, gotEntry fs.DirEntry, gotState *file.State) (*file.State, error) {
				gotAdvisorCall = true
				if gotPkg != pkg {
					t.Errorf("advisor: want pkg %q, got %q", pkg, gotPkg)
				}
				if gotItem != itemName {
					t.Errorf("advisor: want item %q, got %q", itemName, gotItem)
				}
				if !reflect.DeepEqual(gotState, test.targetItemState) {
					t.Errorf("advisor: want prior advice %v, got %v", test.targetItemState, gotState)
				}
				return test.advisorAdvice, test.advisorError
			})

			fsys := fileStateFS{}
			if test.targetItemState != nil {
				targetItem := path.Join(target, itemName)
				fsys[targetItem] = *test.targetItemState
			}

			index := item.Index{}

			pa := NewPkgAnalyst(fsys, target, source, pkg, index, advisor)

			sourcePkgItem := path.Join(source, pkg, itemName)
			gotErr := pa.VisitPath(sourcePkgItem, nil, nil)

			if !gotAdvisorCall {
				t.Errorf("no call to advisor")
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			gotSpec, ok := index[itemName]

			if !ok {
				t.Fatalf("did not record spec")
			}

			gotCurrentState := gotSpec.Current
			if !reflect.DeepEqual(gotCurrentState, test.targetItemState) {
				t.Errorf("current state\nwant %v\ngot  %v", test.targetItemState, gotCurrentState)
			}

			gotDesiredState := gotSpec.Desired
			if !reflect.DeepEqual(gotDesiredState, test.wantDesiredState) {
				t.Errorf("desired state\nwant %v\ngot  %v", test.wantDesiredState, gotDesiredState)
			}
		})
	}
}

type errFSResults struct {
	mode        fs.FileMode
	lstatErr    error
	readLinkErr error
}

type errFS map[string]*errFSResults

func (fsys errFS) Open(path string) (fs.File, error) {
	return nil, &fs.PathError{Op: "shortCircuit.open", Path: path, Err: errors.ErrUnsupported}
}

func (fsys errFS) Stat(path string) (fs.FileInfo, error) {
	return nil, &fs.PathError{Op: "shortCircuit.stat", Path: path, Err: errors.ErrUnsupported}
}

func (fsys errFS) Lstat(name string) (fs.FileInfo, error) {
	results, ok := fsys[name]
	if !ok {
		return nil, &fs.PathError{Op: "shortCircuit.stat", Path: name, Err: fs.ErrNotExist}
	}
	info := &fileStateInfo{name: path.Base(name), state: file.State{Mode: results.mode}}
	return info, results.lstatErr
}

func (fsys errFS) ReadLink(path string) (string, error) {
	results, ok := fsys[path]
	if !ok {
		return "", &fs.PathError{Op: "shortCircuit.readlink", Path: path, Err: fs.ErrNotExist}
	}
	return "", results.readLinkErr
}

// Tests of situations that produce errors
// and preclude recording the target file state or calling the item advisor.
func TestPkgAnalystVisitPathError(t *testing.T) {
	var (
		anLstatError   = errors.New("error returned from lstat")
		aReadLinkError = errors.New("error returned from readlink")
		aWalkError     = errors.New("error passed to VisitPath")
	)

	tests := map[string]struct {
		item           string      // The item being visited, or . to visit the pkg dir
		walkErr        error       // The error passed to VisitPath
		targetItemMode fs.FileMode // The mode of the target item file
		lstatErr       error       // Error returned from Lstat
		readLinkErr    error       // Error returned from ReadLink
		wantError      error
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
		"target item lstat error": {
			item:      "item",
			lstatErr:  anLstatError,
			wantError: anLstatError,
		},
		"target item readlink error": {
			item:           "item",
			targetItemMode: fs.ModeSymlink,
			readLinkErr:    aReadLinkError,
			wantError:      aReadLinkError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			const (
				source = "path/to/source"
				target = "path/to/target"
				pkg    = "pkg"
			)

			fsys := errFS{}
			if test.lstatErr != nil || test.readLinkErr != nil {
				targetItem := path.Join(target, test.item)
				result := &errFSResults{
					mode:        test.targetItemMode,
					lstatErr:    test.lstatErr,
					readLinkErr: test.readLinkErr,
				}
				fsys[targetItem] = result
			}

			emptyIndex := item.Index{}
			var uncallableAdvisor Advisor = nil
			pa := NewPkgAnalyst(fsys, target, source, pkg, emptyIndex, uncallableAdvisor)

			walkPath := path.Join(source, pkg, test.item)

			gotErr := pa.VisitPath(walkPath, nil, test.walkErr)

			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("want error %q, got %q", test.wantError, gotErr)
			}

			if len(emptyIndex) != 0 {
				t.Error("want no specs, got:")
				for item, spec := range emptyIndex {
					t.Errorf("    %q: %v", item, spec)
				}
			}
		})
	}
}
