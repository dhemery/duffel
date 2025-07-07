package plan

import (
	"io/fs"
	"maps"
	"path"
	"slices"

	"github.com/dhemery/duffel/internal/file"
)

type ItemOp interface {
	Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type Planner struct {
	FS     fs.FS
	Target string
	Source string
}

func (p Planner) Plan(ops []PkgOp, fileStater Stater) (Plan, error) {
	index := NewStateCache(fileStater)

	for _, op := range ops {
		walkDir := path.Join(p.Source, op.Pkg)
		err := fs.WalkDir(p.FS, walkDir, op.VisitFunc(p.Source, index))
		if err != nil {
			return Plan{}, err
		}
	}

	tasks := make([]Task, 0)
	for _, item := range slices.Sorted(maps.Keys(index.states)) {
		state := index.states[item]
		if state == nil {
			continue
		}

		task := Task{Item: item, State: *state}
		tasks = append(tasks, task)
	}
	return Plan{Target: p.Target, Tasks: tasks}, nil
}

type PkgOp struct {
	Pkg    string
	ItemOp ItemOp
}

type Stater interface {
	State(name string) (*file.State, error)
}

type Index interface {
	Stater
	SetState(item string, state *file.State)
}

func (po PkgOp) VisitFunc(source string, index Index) fs.WalkDirFunc {
	pkgDir := path.Join(source, po.Pkg)
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == pkgDir {
			// Skip the dir being walked. It is not an item.
			return nil
		}

		item := name[len(pkgDir)+1:]
		oldState, err := index.State(item)
		if err != nil {
			return err
		}

		newState, err := po.ItemOp.Apply(po.Pkg, item, entry, oldState)

		if err == nil || err == fs.SkipDir {
			index.SetState(item, newState)
		}

		return err
	}
}
