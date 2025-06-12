package duffel

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
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
		target        = "path/to/target"
		source        = "path/to/source"
		pkg           = "pkg"
		item          = "item"
		dirReadable   = fs.ModeDir | 0o755
		dirUnreadable = fs.ModeDir | 0o311
		fileReadable  = 0o644
	)

	customWalkError := errors.New("error passed to VisitPath")

	type advisorCall struct {
		pkgArg         string      // The pkg passed to the advisor
		itemArg        string      // The item passed to the advisor
		entryArg       fs.DirEntry // The entry passed to the advisor
		priorAdviceArg *FileState  // The prior advice passed to the advisor
		adviceResult   *FileState  // The advice returned by the advisor
		errResult      error       // The error returned by the advisor
	}
	tests := map[string]struct {
		files           fstest.MapFS // Files on the file system
		priorFileGap    *FileGap     // The recorded file gap for the path before VisitPath
		walkPath        string       // The path passed to VisitPath
		walkEntry       fs.DirEntry  // The dir entry passed to VisitPath
		walkErr         error        // The error passed to VisitPath
		wantFileGap     *FileGap     // The recorded file gap for the path after VisitPath
		wantAdvisorCall *advisorCall // Wanted call to item advisor
		wantErr         error        // Error returned by VisitPath
	}{
		"pkg dir and walk error": {
			walkPath:        path.Join(source, pkg),
			walkErr:         customWalkError,
			wantFileGap:     nil,             // Do not record a file gap for the pkg dir
			wantAdvisorCall: nil,             // Do seek advice for the pkg dir
			wantErr:         customWalkError, // Return the walk error
		},
		"pkg dir and no walk error": {
			walkPath:        path.Join(source, pkg),
			walkErr:         nil,
			wantAdvisorCall: nil, // Do not seek advice for the pkg dir
			wantFileGap:     nil, // Do not record a file gap for the pkg dir
			wantErr:         nil,
		},
		"item and walk error": {
			walkPath:        path.Join(source, pkg, item),
			walkErr:         customWalkError,
			wantAdvisorCall: nil,             // Do seek advice about an item with a walk error
			wantFileGap:     nil,             // Do not record a file gap for an item with a walk error
			wantErr:         customWalkError, // Return the walk error
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotAdvisorCall := false

			advisor := adviseFunc(func(pkg, item string, d fs.DirEntry, s *FileState) (*FileState, error) {
				call := test.wantAdvisorCall
				if call == nil {
					return nil, fmt.Errorf("advisor: unwanted call with pkg %q, item %q", pkg, item)
				}
				gotAdvisorCall = true
				if pkg != call.pkgArg {
					t.Errorf("advisor: want pkg %q, got %q", call.pkgArg, pkg)
				}
				if item != call.itemArg {
					t.Errorf("advisor:, want item %q, got %q", call.itemArg, item)
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
		})
	}
}
