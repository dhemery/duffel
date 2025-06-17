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

func TestAdviseInstall(t *testing.T) {
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
		"in state links to foreign dest": {
			item:      "item",
			stateArg:  &file.State{Dest: "current/foreign/dest"},
			wantState: nil,
			wantErr:   &ErrConflict{},
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
