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

func TestInstall(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	targetToSource, _ := filepath.Rel(target, source)

	tests := map[string]struct {
		item        string          // Item being analyzed, relative to pkg dir
		status      *Status         // Item status before Analyze
		targetEntry *fstest.MapFile // File entry for the item in target dir
		wantStatus  Status          // Item status after Analyze
		wantErr     error           // Error returned Analyze
	}{
		"no status, no target file": {
			item:        "item",
			status:      nil, // Not yet analyzed
			targetEntry: nil, // No target file
			wantStatus: Status{
				// Does not set a current state because no target file
				Current: nil,
				// Proposes linking to pkg item
				Desired: &State{
					Mode: fs.ModeSymlink,
					Dest: path.Join(targetToSource, pkg, "item"),
				},
			},
			wantErr: nil,
		},
		"desired state, no target file": {
			item:        "item",
			status:      &Status{Desired: &State{Dest: "desired/dest"}},
			targetEntry: nil,
			wantStatus:  Status{Desired: &State{Dest: "desired/dest"}}, // Unchanged
			wantErr:     &ErrConflict{},
		},
		"target file with no status": {
			item:        "item",
			status:      nil,                          // Not yet analyzed
			targetEntry: &fstest.MapFile{Mode: 0o644}, // Plain file
			wantStatus: Status{
				// Records the current state of the target file
				Current: &State{Mode: 0o644},
				// Proposes to leave the target in its current state
				Desired: &State{Mode: 0o644},
			},
			wantErr: &ErrConflict{},
		},
		"current state links to foreign dest": {
			item: "item",
			// Current state set by earlier analysis
			status: &Status{
				Current: &State{Dest: "current/foreign/dest"},
				Desired: &State{Dest: "current/foreign/dest"},
			},
			// Does not change the status
			wantStatus: Status{
				Current: &State{Dest: "current/foreign/dest"},
				Desired: &State{Dest: "current/foreign/dest"},
			},
			wantErr: &ErrConflict{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fsys := duftest.NewFS()
			fsys.M[target] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}
			targetItem := path.Join(target, test.item)
			fsys.M[targetItem] = test.targetEntry

			tree := TargetTree{}
			if test.status != nil {
				tree[test.item] = *test.status
			}

			install := Install{
				fsys:           fsys,
				source:         source,
				target:         target,
				targetToSource: targetToSource,
				tree:           tree,
			}

			gotErr := install.Analyze(pkg, test.item, nil)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			gotStatus, _ := tree.Status(test.item)
			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
				t.Errorf("status:\nwant %s\ngot  %s", test.wantStatus, gotStatus)
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
