package plan

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
)

func TestInstallOp(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	targetToSource, _ := filepath.Rel(target, source)

	tests := map[string]struct {
		item      string      // Item being analyzed, relative to pkg dir
		stateArg  *file.State // Desired state passed to Apply
		wantState *file.State // Desired state returned by by Apply
		wantErr   error       // Error returned by Apply
	}{
		"no in state": {
			item:     "item",
			stateArg: nil,
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"in state is dir": {
			item:     "item",
			stateArg: &file.State{Mode: fs.ModeDir | 0o755},
			wantErr:  ErrIsDir,
		},
		"in state is file": {
			item:     "item",
			stateArg: &file.State{Mode: 0o644},
			wantErr:  ErrIsFile,
		},
		"in state links to current pkg item": {
			item: "item",
			stateArg: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"in state links to foreign dest": {
			item:      "item",
			stateArg:  &file.State{Mode: fs.ModeSymlink, Dest: "current/foreign/dest"},
			wantState: nil,
			wantErr:   ErrNotPkgItem,
		},
		"in state is not file, dir, or link": {
			item:     "item",
			stateArg: &file.State{Mode: fs.ModeDevice},
			wantErr:  ErrTargetType,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := Install{
				TargetToSource: targetToSource,
			}

			gotAdvice, gotErr := install.Apply(pkg, test.item, nil, test.stateArg)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			if !reflect.DeepEqual(gotAdvice, test.wantState) {
				t.Errorf("item advice:\nwant %#v\ngot  %#v", test.wantState, gotAdvice)
			}
		})
	}
}
