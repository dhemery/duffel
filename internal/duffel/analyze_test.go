package duffel

import (
	"errors"
	"io/fs"
	"path"
	"reflect"
	"testing"
	"time"
)

type adviseFunc func(string, string, fs.DirEntry, *FileState) (*FileState, error)

func (af adviseFunc) Advise(pkg, item string, d fs.DirEntry, priorGoal *FileState) (*FileState, error) {
	return af(pkg, item, d, priorGoal)
}

type fileStateFS map[string]FileState

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
	state FileState
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
		item           = "item"
		dirReadable    = fs.ModeDir | 0o755
		dirUnreadable  = fs.ModeDir | 0o311
		fileReadable   = 0o644
	)
	anAdvisorError := errors.New("error returned from advisor")

	tests := map[string]struct {
		targetItemState  *FileState // The state of the item in the target dir
		advisorAdvice    *FileState // The advice returned by the advisor
		advisorError     error      // Error returned by advisor
		wantErr          error      // Error returned by VisitPath
		wantDesiredState *FileState // The recorded desired state for the item after VisitPath
		skip             string     // Reason for skipping test
	}{
		"no target item": {
			targetItemState:  nil,
			advisorAdvice:    &FileState{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
			wantDesiredState: &FileState{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
		},
		"target item is dir, advisor advises": {
			targetItemState:  &FileState{Mode: fs.ModeDir | 0o755},
			advisorAdvice:    &FileState{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
			wantDesiredState: &FileState{Mode: fs.ModeSymlink, Dest: "dest/from/advisor"},
		},
		"target item is link, advisor advises": {
			targetItemState:  &FileState{Mode: fs.ModeSymlink, Dest: "dest/from/stat"},
			advisorAdvice:    &FileState{Mode: fs.ModeDir | 0o755},
			wantDesiredState: &FileState{Mode: fs.ModeDir | 0o755},
		},
		"target item is file, advisor reports error": {
			targetItemState:  &FileState{Mode: 0o644},
			advisorError:     anAdvisorError,
			wantErr:          anAdvisorError,
			wantDesiredState: &FileState{Mode: 0o644},
		},
		"target item is dir, advisor reports error": {
			targetItemState:  &FileState{Mode: fs.ModeDir | 0o755},
			advisorError:     anAdvisorError,
			wantErr:          anAdvisorError,
			wantDesiredState: &FileState{Mode: fs.ModeDir | 0o755},
		},
		"target item is link, advisor reports error": {
			targetItemState:  &FileState{Mode: fs.ModeSymlink, Dest: "dest/from/stat"},
			advisorAdvice:    nil,
			advisorError:     anAdvisorError,
			wantErr:          anAdvisorError,
			wantDesiredState: &FileState{Mode: fs.ModeSymlink, Dest: "dest/from/stat"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}
			gotAdvisorCall := false

			advisor := adviseFunc(func(gotPkg, gotItem string, gotEntry fs.DirEntry, gotState *FileState) (*FileState, error) {
				gotAdvisorCall = true
				if gotPkg != pkg {
					t.Errorf("advisor: want pkg %q, got %q", pkg, gotPkg)
				}
				if gotItem != item {
					t.Errorf("advisor: want item %q, got %q", item, gotItem)
				}
				if !reflect.DeepEqual(gotState, test.targetItemState) {
					t.Errorf("advisor: want prior advice %v, got %v", test.targetItemState, gotState)
				}
				return test.advisorAdvice, test.advisorError
			})

			fsys := fileStateFS{}
			if test.targetItemState != nil {
				targetItem := path.Join(target, item)
				fsys[targetItem] = *test.targetItemState
			}

			targetGap := TargetGap{}

			pa := NewPkgAnalyst(fsys, target, source, pkg, targetGap, advisor)

			sourcePkgItem := path.Join(source, pkg, item)
			gotErr := pa.VisitPath(sourcePkgItem, nil, nil)

			if !gotAdvisorCall {
				t.Errorf("no call to advisor")
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			gotFileGap, ok := targetGap[item]

			if !ok {
				t.Fatalf("did not record file gap")
			}

			gotCurrentState := gotFileGap.Current
			if !reflect.DeepEqual(gotCurrentState, test.targetItemState) {
				t.Errorf("current state\nwant %v\ngot  %v", test.targetItemState, gotCurrentState)
			}

			gotDesiredState := gotFileGap.Desired
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
	info := &fileStateInfo{name: path.Base(name), state: FileState{Mode: results.mode}}
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

			emptyTargetGap := TargetGap{}
			var uncallableAdvisor ItemAdvisor = nil
			pa := NewPkgAnalyst(fsys, target, source, pkg, emptyTargetGap, uncallableAdvisor)

			walkPath := path.Join(source, pkg, test.item)

			gotErr := pa.VisitPath(walkPath, nil, test.walkErr)

			if !errors.Is(gotErr, test.wantError) {
				t.Errorf("want error %q, got %q", test.wantError, gotErr)
			}

			if len(emptyTargetGap) != 0 {
				t.Error("want no file gaps, got:")
				for path, gap := range emptyTargetGap {
					t.Errorf("    %q: %v", path, gap)
				}
			}
		})
	}
}
