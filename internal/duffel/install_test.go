package duffel

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/duftest"
)

func TestAdviseInstall(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	targetToSource, _ := filepath.Rel(target, source)

	tests := map[string]struct {
		item           string     // Item being analyzed, relative to pkg dir
		priorAdviceArg *FileState // Desired state passed to Advise
		wantAdvice     *FileState // Desired state returned by by Advise
		wantErr        error      // Error returned by Advise
	}{
		"no prior advice": {
			item:           "item",
			priorAdviceArg: nil,
			wantAdvice: &FileState{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"prior advice links to foreign dest": {
			item:           "item",
			priorAdviceArg: &FileState{Dest: "current/foreign/dest"},
			wantAdvice:     nil,
			wantErr:        &ErrConflict{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fsys := duftest.NewFS()
			fsys.M[target] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}

			install := Install{
				FS:             fsys,
				TargetToSource: targetToSource,
			}

			gotAdvice, gotErr := install.Advise(pkg, test.item, nil, test.priorAdviceArg)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			if !reflect.DeepEqual(gotAdvice, test.wantAdvice) {
				t.Errorf("item advice:\nwant %#v\ngot  %#v", test.wantAdvice, gotAdvice)
			}

			if t.Failed() {
				t.Log("files in fsys:")
				for fname, entry := range fsys.M {
					t.Logf("    %s: %v", fname, entry)
				}
			}
		})
	}
}
