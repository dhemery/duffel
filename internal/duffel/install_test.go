package duffel

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/testfs"
)

type dirEntry struct {
	name string
	mode fs.FileMode
	info fs.FileInfo
}

func (d dirEntry) IsDir() bool {
	return d.mode.IsDir()
}

func (d dirEntry) Info() (fs.FileInfo, error) {
	if d.info == nil {
		return nil, fs.ErrNotExist
	}
	return d.info, nil
}

func (d dirEntry) Name() string {
	return d.name
}

func (d dirEntry) Type() fs.FileMode {
	return d.mode & fs.ModeType
}

func TestInstallAnalyze(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	var (
		targetToSource, _ = filepath.Rel(target, source)
		visitError        = errors.New("error passed to visit")
	)

	tests := map[string]struct {
		item        string          // Item being analyzed, relative to pkg dir
		walkError   error           // Error passed to visit by fs.WalkDir
		status      Status          // Item status before Analyze
		targetEntry *fstest.MapFile // File entry for the item in target dir
		wantStatus  Status          // Item status after Analyze
		wantErr     error           // Error returned Analyze
		skip        string          // Reason for skipping this test
	}{
		"new target item with no status": {
			item:        "item",
			targetEntry: nil,
			status:      Status{},
			wantStatus: Status{
				// Analyze did not record a current state
				Current: nil,
				// Analyze proposed a desired state
				Desired: &State{Dest: path.Join(targetToSource, pkg, "item")},
			},
			wantErr: nil,
		},
		"new target item with desired state": {
			item:        "item",
			targetEntry: nil,
			status:      Status{Desired: &State{Dest: "desired/dest"}},
			wantStatus:  Status{Desired: &State{Dest: "desired/dest"}}, // Unchanged
			wantErr:     &ErrConflict{},
		},
		"current target file first visit": {
			item:        "item",
			targetEntry: testfs.FileEntry("content", 0o644),
			// First visit, so no  status
			status: Status{},
			wantStatus: Status{
				// Analyze recorded the state of the current target file
				Current: &State{Mode: 0o644},
				// Analyze did not propose a desired state
				Desired: nil,
			},
			wantErr: &ErrConflict{},
			skip:    "not yet implemented",
		},
		"current target file already visited": {
			item: "item",
			// Current state recorded on earlier visit
			status: Status{Current: &State{Dest: "current/dest"}},
			// Does not change the status
			wantStatus: Status{Current: &State{Dest: "current/dest"}},
			wantErr:    &ErrConflict{},
			skip:       "not yet implemented",
		},
		"visit pkg dir": {
			item:       ".",
			walkError:  nil,
			wantErr:    nil,      // Succesfully...
			wantStatus: Status{}, // ... does not set the status
			skip:       "responsibility moved to PkgAnalyst",
		},
		"visit pkg dir that gave walk error": {
			item:       ".",
			walkError:  visitError,
			wantStatus: Status{},   // Does not set the status
			wantErr:    visitError, // Returns the given error
			skip:       "responsibility moved to PkgAnalyst",
		},
		"visit item that gave walk error": {
			item:       "item",
			walkError:  visitError,
			wantStatus: Status{},   // Does not set the status
			wantErr:    visitError, // Returns the given error
			skip:       "responsibility moved to PkgAnalyst",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}
			sourcePkg := path.Join(source, pkg)
			sourcePkgItem := path.Join(sourcePkg, test.item)

			fsys := testfs.New()
			fsys.M[target] = testfs.DirEntry(0o755)
			if test.targetEntry != nil {
				fsys.M[sourcePkgItem] = test.targetEntry
			}

			image := Image{}
			image[test.item] = test.status

			v := Install{
				source:         source,
				target:         target,
				targetToSource: targetToSource,
				image:          image,
			}

			gotErr := v.Analyze(pkg, test.item, nil)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %#v\ngot  %#v", test.wantErr, gotErr)
			}

			gotStatus, _ := image.Status(test.item)
			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
				t.Errorf("status:\nwant %s\ngot  %s", test.wantStatus, gotStatus)
			}
		})
	}
}
