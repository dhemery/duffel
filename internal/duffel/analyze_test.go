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

type adviseFunc func(string, string, fs.DirEntry, *FileState) (*FileState, error)

func (af adviseFunc) Advise(pkg, item string, d fs.DirEntry, priorGoal *FileState) (*FileState, error) {
	return af(pkg, item, d, priorGoal)
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

	var (
		customWalkError    = errors.New("error passed to VisitPath")
		customAdvisorError = errors.New("error returned from advisor")
	)

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
		walkEntry       fs.DirEntry  // The dir entry passed to VisitPath
		walkErr         error        // The error passed to VisitPath
		wantAdvisorCall *advisorCall // Wanted call to item advisor
		wantErr         error        // Error returned by VisitPath
		wantFileGap     *FileGap     // The recorded file gap for the path after VisitPath
	}{
		"returns walk error for pkg dir with walk error": {
			walkPath:        path.Join(source, pkg),
			walkErr:         customWalkError,
			wantFileGap:     nil,             // Do not record a file gap for the pkg dir
			wantAdvisorCall: nil,             // Do seek advice for the pkg dir
			wantErr:         customWalkError, // Return the walk error
		},
		"does nothing if pkg dir": {
			walkPath:        path.Join(source, pkg),
			walkErr:         nil,
			wantAdvisorCall: nil, // Do not seek advice for the pkg dir
			wantFileGap:     nil, // Do not record a file gap for the pkg dir
			wantErr:         nil,
		},
		"returns walk error for item with walk error": {
			walkPath:        path.Join(source, pkg, item),
			walkErr:         customWalkError,
			wantAdvisorCall: nil,             // Do seek advice for an item with a walk error
			wantFileGap:     nil,             // Do not record a file gap for an item with a walk error
			wantErr:         customWalkError, // Return the walk error
		},
		"records item advice if no target file and no prior advice": {
			walkPath:     path.Join(source, pkg, item),
			files:        fstest.MapFS{}, // No target file for item
			priorFileGap: nil,
			walkEntry:    nil,
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

			gotErr := pa.VisitPath(test.walkPath, test.walkEntry, test.walkErr)

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
