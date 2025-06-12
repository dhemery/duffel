package duffel

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/duftest"
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

func TestPkgAnalystVisitPath(t *testing.T) {
	customAdvisorError := errors.New("error returned from advisor")

	type advisorCall struct {
		pkgArg         string      // The pkg passed to the advisor
		itemArg        string      // The item passed to the advisor
		entryArg       fs.DirEntry // The entry passed to the advisor
		priorAdviceArg *FileState  // The prior advice passed to the advisor
		adviceResult   *FileState  // The advice returned by the advisor
		errResult      error       // The error returned by the advisor
	}
	tests := map[string]struct {
		priorFileGap    *FileGap     // The recorded file gap for the path before VisitPath
		files           fstest.MapFS // Files on the file system
		walkPath        string       // The path passed to VisitPath
		wantAdvisorCall *advisorCall // Wanted call to item advisor
		wantErr         error        // Error returned by VisitPath
		wantFileGap     *FileGap     // The recorded file gap for the path after VisitPath
	}{
		"records item advice if no target file and no prior advice": {
			walkPath:     path.Join(source, pkg, item),
			files:        fstest.MapFS{}, // No target file for item
			priorFileGap: nil,
			wantAdvisorCall: &advisorCall{
				pkgArg:         pkg,
				itemArg:        item,
				entryArg:       nil,
				priorAdviceArg: nil,
				adviceResult: &FileState{
					Mode: fs.ModeSymlink,
					Dest: "path/advised/by/advisor",
				},
				errResult: nil,
			},
			wantErr: nil,
			wantFileGap: &FileGap{
				Desired: &FileState{
					Mode: fs.ModeSymlink,
					Dest: "path/advised/by/advisor",
				},
			},
		},
		"returns advisor error": {
			priorFileGap: nil,
			files: fstest.MapFS{
				path.Join(target, item): &fstest.MapFile{Mode: 0o644}, // Plain file
			},
			walkPath: path.Join(source, pkg, item),
			wantAdvisorCall: &advisorCall{
				pkgArg:         pkg,
				itemArg:        item,
				entryArg:       nil,
				priorAdviceArg: &FileState{Mode: 0o644},
				adviceResult:   nil,
				errResult:      customAdvisorError,
			},
			wantErr: customAdvisorError,
			// Records file state from stat
			wantFileGap: &FileGap{
				Current: &FileState{Mode: 0o644},
				Desired: &FileState{Mode: 0o644},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotAdvisorCall := false

			advisor := adviseFunc(func(pkg, item string, d fs.DirEntry, priorAdvice *FileState) (*FileState, error) {
				call := test.wantAdvisorCall
				if call == nil {
					return nil, fmt.Errorf("advisor: unwanted call with pkg %q, item %q, priorAdvice %v",
						pkg, item, priorAdvice)
				}
				gotAdvisorCall = true
				if pkg != call.pkgArg {
					t.Errorf("advisor: want pkg %q, got %q", call.pkgArg, pkg)
				}
				if item != call.itemArg {
					t.Errorf("advisor: want item %q, got %q", call.itemArg, item)
				}
				if !reflect.DeepEqual(priorAdvice, call.priorAdviceArg) {
					t.Errorf("advisor: want prior advice %v, got %v", call.priorAdviceArg, priorAdvice)
				}
				return call.adviceResult, call.errResult
			})

			fsys := duftest.FS{M: test.files}
			targetGap := TargetGap{}

			pa := NewPkgAnalyst(fsys, target, source, pkg, targetGap, advisor)

			gotErr := pa.VisitPath(test.walkPath, nil, nil)

			if test.wantAdvisorCall != nil && !gotAdvisorCall {
				t.Errorf("no call to advisor, wanted: %#v", test.wantAdvisorCall)
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			gotFileGap, ok := targetGap[item]
			switch {
			case test.wantFileGap == nil && ok:
				t.Errorf("file gap:\n    want: none\n    got : got %#v", gotFileGap)
			case test.wantFileGap != nil && !ok:
				t.Errorf("file gap:\n    want: %v\n    got : none", test.wantFileGap)
			case test.wantFileGap != nil && !reflect.DeepEqual(&gotFileGap, test.wantFileGap):
				t.Errorf("file gap:\n    want: %v\n    got : %v", test.wantFileGap, gotFileGap)

			}
			if t.Failed() {
				t.Error("target gap:")
				for n, g := range targetGap {
					t.Errorf("    %q: %v", n, g)
				}
			}
		})
	}
}

func TestPkgAnalystVisitPathSpecialCases(t *testing.T) {
	aWalkError := errors.New("error passed to VisitPath")

	tests := map[string]struct {
		path      string
		err       error
		wantError error
	}{
		"pkg dir with no walk error": {
			path:      path.Join(source, pkg),
			err:       nil,
			wantError: nil,
		},
		"pkg dir with walk error": {
			path:      path.Join(source, pkg),
			err:       aWalkError,
			wantError: aWalkError,
		},
		"item with walk error": {
			path:      path.Join(source, pkg, item),
			err:       aWalkError,
			wantError: aWalkError,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Do not want calls to FS, target gap, or item advisor.
			pa := NewPkgAnalyst(nil, "", source, pkg, nil, nil)

			// Do not want calls to entry.
			err := pa.VisitPath(test.path, nil, test.err)

			if err != test.wantError {
				t.Errorf("want error %q, got %q", test.wantError, err)
			}
		})
	}
}

func TestPkgAnalystVisitPathStatError(t *testing.T) {
	wantErr := errors.New("wanted stat error")

	fsys := duftest.ErrFS{StatErr: wantErr}

	targetGap := TargetGap{} // No gaps, forces stat of target item

	// Do not want call to item advisor.
	pa := NewPkgAnalyst(fsys, target, source, pkg, targetGap, nil)

	gotErr := pa.VisitPath(path.Join(source, pkg, item), nil, nil)

	if !errors.Is(gotErr, wantErr) {
		t.Errorf("want error %q, got %q", wantErr, gotErr)
	}
}
