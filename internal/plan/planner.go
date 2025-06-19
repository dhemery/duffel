package plan

import (
	"io/fs"
	"iter"
	"path"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/item"
)

type ItemOp func(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)

type ItemStates interface {
	Desired(item string) (*file.State, error)
	SetDesired(item string, state *file.State)
}

type StateSequencer interface {
	ByItem() iter.Seq2[string, item.Spec]
}

type StateIndex interface {
	ItemStates
	StateSequencer
}

type Planner struct {
	FS     fs.FS
	Target string
	Source string
	Index  StateIndex
}

func (p Planner) Plan(ops []PkgOp) (Plan, error) {
	for _, op := range ops {
		walkDir := path.Join(p.Source, op.Pkg)
		err := fs.WalkDir(p.FS, walkDir, op.VisitFunc(p.Source, p.Index))
		if err != nil {
			return Plan{}, err
		}
	}
	return New(p.Target, p.Index), nil
}

type PkgOp struct {
	Pkg   string
	Apply ItemOp
}

func (op PkgOp) VisitFunc(source string, index ItemStates) fs.WalkDirFunc {
	pkgDir := path.Join(source, op.Pkg)
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == pkgDir {
			// Skip the dir being walked. It is not an item.
			return nil
		}

		item := name[len(pkgDir)+1:]
		priorState, err := index.Desired(item)
		if err != nil {
			return err
		}

		newState, err := op.Apply(op.Pkg, item, entry, priorState)
		if err != nil {
			return err
		}

		index.SetDesired(item, newState)
		return nil
	}
}
