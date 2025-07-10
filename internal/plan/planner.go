package plan

import (
	"io/fs"
	"iter"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
}

type Planner struct {
	FS     fs.FS
	Target string
	Source string
}

type states interface {
	Sorted() iter.Seq2[string, *file.State]
}

func (p Planner) Plan(ops []PkgOp, itemStates states) (Plan, error) {
	for _, op := range ops {
		walkDir := path.Join(p.Source, op.Pkg)
		err := fs.WalkDir(p.FS, walkDir, op.VisitFunc(p.Source))
		if err != nil {
			return Plan{}, err
		}
	}

	tasks := make([]Task, 0)
	for item, state := range itemStates.Sorted() {
		if state == nil {
			continue
		}

		task := Task{Item: item, State: *state}
		tasks = append(tasks, task)
	}
	return Plan{Target: p.Target, Tasks: tasks}, nil
}

type ItemOp interface {
	Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type PkgOp struct {
	Pkg    string
	ItemOp ItemOp
	Index  Index
}

func (po PkgOp) VisitFunc(source string) fs.WalkDirFunc {
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
		oldState, err := po.Index.State(item)
		if err != nil {
			return err
		}

		newState, err := po.ItemOp.Apply(po.Pkg, item, entry, oldState)

		if err == nil || err == fs.SkipDir {
			po.Index.SetState(item, newState)
		}

		return err
	}
}
