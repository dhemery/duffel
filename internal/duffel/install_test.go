package duffel

import (
	"errors"
	"io/fs"
	"path"
	"slices"
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
	customError := errors.New("custom error for visit")

	tests := map[string]struct {
		walkPath  string
		visitPath string
		visitErr  error
		wantTasks []Task
		wantErr   error
	}{
		"visit pkg dir": {
			walkPath:  path.Join(source, pkg),
			visitPath: path.Join(source, pkg),
			wantErr:   nil,
			wantTasks: nil,
		},
		"visit pkg dir with error": {
			walkPath:  path.Join(source, pkg),
			visitPath: path.Join(source, pkg),
			visitErr:  customError,
			wantErr:   customError,
			wantTasks: nil,
		},
		"visit item": {
			walkPath:  path.Join(source, pkg),
			visitPath: path.Join(source, pkg, item),
			visitErr:  nil,
			wantErr:   nil,
			wantTasks: []Task{
				CreateLink{
					Item:   item,
					Action: "link",
					Dest:   path.Join(targetToSource, pkg, item),
				},
			},
		},
		"visit item with error": {
			walkPath:  path.Join(source, pkg),
			visitPath: path.Join(source, pkg, item),
			visitErr:  customError,
			wantErr:   customError,
			wantTasks: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			planner := NewPlanner("", targetToSource)

			visit := PlanInstallPackage(planner, test.walkPath, pkg)

			err := visit(test.visitPath, nil, test.visitErr)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("want error %v, got %v", test.wantErr, err)
			}

			gotTasks := planner.Plan.Tasks
			if !slices.Equal(gotTasks, test.wantTasks) {
				t.Errorf("want tasks %#v, got %#v", test.wantTasks, gotTasks)
			}
		})
	}
}
