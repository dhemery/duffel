package duffel

import (
	"errors"
	"io/fs"
	"path"
	"testing"
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

func TestInstallVisitInput(t *testing.T) {
	const (
		source         = "path/to/source"
		pkg            = "pkg"
		item           = "item"
		targetToSource = "target/to/source"
	)
	visitErr := errors.New("error passed to visit")

	tests := map[string]struct {
		item       string
		visitErr   error
		wantStatus Status
		wantErr    error
	}{
		"visit pkg dir": {
			item:       ".",
			wantErr:    nil,
			wantStatus: Status{},
		},
		"visit pkg dir with error": {
			item:       ".",
			visitErr:   visitErr,
			wantErr:    visitErr,
			wantStatus: Status{},
		},
		"visit item": {
			item:       item,
			visitErr:   nil,
			wantErr:    nil,
			wantStatus: Status{Planned: Result{Dest: path.Join(targetToSource, pkg, item)}},
		},
		"visit item with error": {
			item:       item,
			visitErr:   visitErr,
			wantErr:    visitErr,
			wantStatus: Status{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sourcePkg := path.Join(source, pkg)
			visitPath := path.Join(sourcePkg, test.item)

			planner := Planner{}

			visit := PlanInstallPackage(planner, targetToSource, sourcePkg, pkg)

			gotErr := visit(visitPath, nil, test.visitErr)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("want error %v, got %v", test.wantErr, gotErr)
			}

			gotStatus := planner.Status(item)
			if gotStatus != test.wantStatus {
				t.Errorf("want status %#v, got %#v", test.wantStatus, gotStatus)
			}
		})
	}
}

func TestInstallVisitStatus(t *testing.T) {
	const (
		source         = "path/to/source"
		pkg            = "pkg"
		item           = "item"
		targetToSource = "target/to/source"
	)
	sourcePkg := path.Join(source, pkg)
	sourcePkgItem := path.Join(sourcePkg, item)

	tests := map[string]struct {
		status     Status
		wantErr    error
		wantStatus Status
	}{
		"no status": {
			status:     Status{},
			wantErr:    nil,
			wantStatus: Status{Planned: Result{Dest: path.Join(targetToSource, pkg, item)}},
		},
		"prior item": {
			status:     Status{Prior: Result{Dest: "prior/link/dest"}},
			wantErr:    &ErrConflict{},
			wantStatus: Status{Prior: Result{Dest: "prior/link/dest"}}, // Unchanged
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			planner := Planner{}
			planner[item] = test.status

			visit := PlanInstallPackage(planner, targetToSource, sourcePkg, pkg)

			gotErr := visit(sourcePkgItem, nil, nil)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("want error %v, got %v", test.wantErr, gotErr)
			}

			gotStatus := planner.Status(item)
			if gotStatus != test.wantStatus {
				t.Errorf("want status %#v, got %#v", test.wantStatus, gotStatus)
			}
		})
	}
}
