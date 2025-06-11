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

func TestInstallAnalyze(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	targetToSource, _ := filepath.Rel(target, source)

	tests := map[string]struct {
		item        string          // Item being analyzed, relative to pkg dir
		itemGap     *FileGap        // Item gap before Analyze
		targetEntry *fstest.MapFile // File entry for the item in target dir
		wantItemGap FileGap         // Item gap after Analyze
		wantErr     error           // Error returned Analyze
	}{
		"no gap, no target file": {
			item:        "item",
			itemGap:     nil, // Not yet analyzed
			targetEntry: nil, // No target file
			wantItemGap: FileGap{
				// Does not set a current state because no target file
				Current: nil,
				// Proposes linking to pkg item
				Desired: &FileState{
					Mode: fs.ModeSymlink,
					Dest: path.Join(targetToSource, pkg, "item"),
				},
			},
			wantErr: nil,
		},
		"desired state, no target file": {
			item:        "item",
			itemGap:     &FileGap{Desired: &FileState{Dest: "desired/dest"}},
			targetEntry: nil,
			wantItemGap: FileGap{Desired: &FileState{Dest: "desired/dest"}}, // Unchanged
			wantErr:     &ErrConflict{},
		},
		"target file with no gap": {
			item:        "item",
			itemGap:     nil,                          // Not yet analyzed
			targetEntry: &fstest.MapFile{Mode: 0o644}, // Plain file
			wantItemGap: FileGap{
				// Records the current state of the target file
				Current: &FileState{Mode: 0o644},
				// Proposes to leave the target in its current state
				Desired: &FileState{Mode: 0o644},
			},
			wantErr: &ErrConflict{},
		},
		"current state links to foreign dest": {
			item: "item",
			// Current state set by earlier analysis
			itemGap: &FileGap{
				Current: &FileState{Dest: "current/foreign/dest"},
				Desired: &FileState{Dest: "current/foreign/dest"},
			},
			// Does not change the gap
			wantItemGap: FileGap{
				Current: &FileState{Dest: "current/foreign/dest"},
				Desired: &FileState{Dest: "current/foreign/dest"},
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

			targetGap := TargetGap{}
			if test.itemGap != nil {
				targetGap[test.item] = *test.itemGap
			}

			install := Install{
				FS:             fsys,
				Target:         target,
				TargetToSource: targetToSource,
				TargetGap:      targetGap,
			}

			gotErr := install.Analyze(pkg, test.item, nil)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			gotItemGap := targetGap[test.item]
			if !reflect.DeepEqual(gotItemGap, test.wantItemGap) {
				t.Errorf("item gap:\nwant %s\ngot  %s", test.wantItemGap, gotItemGap)
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
