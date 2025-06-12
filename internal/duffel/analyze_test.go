package duffel

import (
	"errors"
	"io/fs"
	"path"
	"reflect"
	"testing"
	"testing/fstest"
)

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

type adviseFunc func(string, string, fs.DirEntry, *FileState) (*FileState, error)

func (af adviseFunc) Advise(pkg, item string, d fs.DirEntry, priorGoal *FileState) (*FileState, error) {
	return af(pkg, item, d, priorGoal)
}

func TestPkgAnalystVisitPathTargetItemState(t *testing.T) {
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
			skip:             "not yet implemented: use Lstat instead of Stat",
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
			skip:             "not yet implemented: use Lstat instead of Stat",
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

			fsys := fstest.MapFS{}
			if test.targetItemState != nil {
				targetItem := path.Join(target, item)
				targetItemFile := &fstest.MapFile{
					Mode: test.targetItemState.Mode,
					Data: []byte(test.targetItemState.Dest),
				}
				fsys[targetItem] = targetItemFile
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

func TestPkgAnalystVisitPathSpecialCases(t *testing.T) {
	aWalkError := errors.New("error passed to VisitPath")

	tests := map[string]struct {
		walkPath  string
		walkErr   error
		wantError error
	}{
		"pkg dir with no walk error": {
			walkPath:  path.Join(source, pkg),
			walkErr:   nil,
			wantError: nil,
		},
		"pkg dir with walk error": {
			walkPath:  path.Join(source, pkg),
			walkErr:   aWalkError,
			wantError: aWalkError,
		},
		"item with walk error": {
			walkPath:  path.Join(source, pkg, item),
			walkErr:   aWalkError,
			wantError: aWalkError,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Do not want calls to FS, target gap, or item advisor.
			pa := NewPkgAnalyst(nil, "", source, pkg, nil, nil)

			// Do not want calls to entry.
			err := pa.VisitPath(test.walkPath, nil, test.walkErr)

			if err != test.wantError {
				t.Errorf("want error %q, got %q", test.wantError, err)
			}
		})
	}
}

type statErrorFS struct {
	StatErr error
}

func (f statErrorFS) Open(path string) (fs.File, error) {
	panic("ErrFS.open called: " + path)
}

func (f statErrorFS) Stat(path string) (fs.FileInfo, error) {
	return nil, f.StatErr
}

func TestPkgAnalystVisitPathStatError(t *testing.T) {
	statError := errors.New("wanted stat error")
	fsys := statErrorFS{StatErr: statError}

	// No recorded gaps, forces stat of target item
	targetGap := TargetGap{}

	// Do not want call to item advisor.
	pa := NewPkgAnalyst(fsys, target, source, pkg, targetGap, nil)

	gotErr := pa.VisitPath(path.Join(source, pkg, item), nil, nil)

	if !errors.Is(gotErr, statError) {
		t.Errorf("want error %q, got %q", statError, gotErr)
	}
}
