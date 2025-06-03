package duffel

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
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

func TestVisitInstall(t *testing.T) {
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
		skip        bool
	}{
		"new target item with no status": {
			item:        "item",
			targetEntry: nil,
			status:      Status{},
			wantStatus: Status{
				// Visit did not change existing state
				Existing: State{},
				// Visit planned the link
				Planned: State{Dest: path.Join(targetToSource, pkg, "item")},
			},
			wantErr: nil,
		},
		"new target item with planned state": {
			item:        "item",
			targetEntry: nil,
			status:      Status{Planned: State{Dest: "planned/dest"}},
			wantStatus:  Status{Planned: State{Dest: "planned/dest"}}, // Unchanged
			wantErr:     &ErrConflict{},
		},
		"existing target file first visit": {
			item:        "item",
			targetEntry: testfs.FileEntry("content", 0o644),
			// First visit, so no  status
			status: Status{},
			wantStatus: Status{
				// Visit recorded existing target file
				Existing: State{Mode: 0o644},
				// Visit did not change the planned state
				Planned: State{},
			},
			wantErr: &ErrConflict{},
			skip:    true,
		},
		"existing target file already visited": {
			item: "item",
			// Existing state recorded on earlier visit
			status: Status{Existing: State{Dest: "existing/dest"}},
			// Visit did change the status
			wantStatus: Status{Existing: State{Dest: "existing/dest"}},
			wantErr:    &ErrConflict{},
			skip:       true,
		},
		"visit pkg dir": {
			item:       ".",
			walkError:  nil,
			wantErr:    nil,      // Succesfully...
			wantStatus: Status{}, // ...plan no action
		},
		"visit pkg dir that gave walk error": {
			item:       ".",
			walkError:  visitError,
			wantErr:    visitError, // Visit returned the given error
			wantStatus: Status{},   // Visit did not change the status
		},
		"visit item that gave walk error": {
			item:       "item",
			walkError:  visitError,
			wantErr:    visitError, // Visit returned the given error
			wantStatus: Status{},   // Visit did not change the status
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.skip {
				t.Skip("wip")
			}
			sourcePkg := path.Join(source, pkg)
			sourcePkgItem := path.Join(sourcePkg, test.item)

			fsys := testfs.New()
			fsys.M[target] = testfs.DirEntry(0o755)
			if test.targetEntry != nil {
				fsys.M[sourcePkgItem] = test.targetEntry
			}

			req := &Request{
				FS:     fsys,
				Target: target,
				Source: source,
			}

			image := Image{}
			image[test.item] = test.status
			v := InstallVisitor{
				target:         target,
				targetToSource: targetToSource,
			}
			visit := PlanInstallPackage(req, pkg, v, image)

			gotErr := visit(sourcePkgItem, nil, test.walkError)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %#v\ngot  %#v", test.wantErr, gotErr)
			}

			gotStatus := image.Status(test.item)
			if gotStatus != test.wantStatus {
				t.Errorf("status:\nwant %v\ngot  %v", test.wantStatus, gotStatus)
			}
		})
	}
}
