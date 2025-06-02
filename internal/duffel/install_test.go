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
			planner := NewPlanner("", "")

			visit := PlanInstallPackage(planner, targetToSource, test.walkPath, pkg)

			err := visit(test.visitPath, nil, test.visitErr)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("want error %v, got %v", test.wantErr, err)
			}

			gotTasks := planner.Tasks()
			if !slices.Equal(gotTasks, test.wantTasks) {
				t.Errorf("want tasks %#v, got %#v", test.wantTasks, gotTasks)
			}
		})
	}
}

func TestInstallVisitPlannedStatus(t *testing.T) {
	const (
		source         = "path/to/source"
		pkg            = "pkg"
		item           = "item"
		targetToSource = "target/to/source"
	)
	sourcePkg := path.Join(source, pkg)
	sourcePkgItem := path.Join(sourcePkg, item)

	tests := map[string]struct {
		status   bool
		wantErr  error
		wantTask Task
	}{
		"no planned status": {
			status:  false,
			wantErr: nil,
			wantTask: CreateLink{
				Action: "link",
				Item:   item,
				Dest:   path.Join(targetToSource, pkg, item),
			},
		},
		"existing item": {
			status:   true,
			wantErr:  &ErrConflict{},
			wantTask: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			planner := NewPlanner("", "")
			planner.Statuses[item] = test.status

			visit := PlanInstallPackage(planner, targetToSource, sourcePkg, pkg)

			err := visit(sourcePkgItem, nil, nil)

			if test.wantErr == nil {
				if err != nil {
					t.Error(err)
				}
			} else {
				var gotWrapped *ErrConflict
				if !errors.As(err, &gotWrapped) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
			}

			gotTasks := planner.Tasks()

			if test.wantTask == nil {
				if len(gotTasks) > 0 {
					t.Fatalf("want no tasks, got %#v", gotTasks)
				}
				return
			}
			if len(gotTasks) == 0 {
				t.Fatalf("want task %#v, got none", test.wantTask)
			}

			gotTask := gotTasks[0]
			gotLinkTask, ok := gotTask.(CreateLink)
			if !ok {
				t.Fatalf("want CreateLink task %#v, got %#v", test.wantTask, gotTask)
			}
			if gotLinkTask != test.wantTask {
				t.Errorf("want task %#v, got %#v", test.wantTask, gotLinkTask)
			}

			if len(gotTasks) > 1 {
				t.Errorf("want 1 task %#v, got %d extra: %#v",
					test.wantTask, len(gotTasks)-1, gotTasks[1:])
			}
		})
	}
}
