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

func TestInstallVisitor(t *testing.T) {
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
		item        string          // Item being visited, relative to pkg dir
		walkError   error           // Error passed to visit
		status      Status          // Planner status before the visit
		targetEntry *fstest.MapFile // File entry for the item in target dir
		wantStatus  Status          // Planner status after visit
		wantErr     error           // Returned by visit
		skip        string
	}{
		"new target item with no status": {
			item:        "item",
			targetEntry: nil,
			status:      Status{},
			wantStatus: Status{
				// Visit did not record a current state
				Current: nil,
				// Visit proposed a desired state
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
				// Visit recorded the state of the current target file
				Current: &State{Mode: 0o644},
				// Visit did not propose a desired state
				Desired: nil,
			},
			wantErr: &ErrConflict{},
			skip:    "not yet implemented",
		},
		"current target file already visited": {
			item: "item",
			// current state recorded on earlier visit
			status: Status{Current: &State{Dest: "current/dest"}},
			// Visit did change the status
			wantStatus: Status{Current: &State{Dest: "current/dest"}},
			wantErr:    &ErrConflict{},
			skip:       "not yet implemented",
		},
		"visit pkg dir": {
			item:       ".",
			walkError:  nil,
			wantErr:    nil,      // Succesfully...
			wantStatus: Status{}, // ...plan no action
			skip:       "responsibility moved to PkgPlanner",
		},
		"visit pkg dir that gave walk error": {
			item:       ".",
			walkError:  visitError,
			wantErr:    visitError, // Visit returned the given error
			wantStatus: Status{},   // Visit did not change the status
			skip:       "responsibility moved to PkgPlanner",
		},
		"visit item that gave walk error": {
			item:       "item",
			walkError:  visitError,
			wantErr:    visitError, // Visit returned the given error
			wantStatus: Status{},   // Visit did not change the status
			skip:       "responsibility moved to PkgPlanner",
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

			v := InstallVisitor{
				source:         source,
				target:         target,
				targetToSource: targetToSource,
				image:          image,
			}

			gotErr := v.VisitItem(pkg, test.item, nil)

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
