package plan

import (
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type ItemOp func(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)

type StateIndex interface {
	Desired(item string) (*file.State, error)
	SetDesired(item string, state *file.State)
}

type Planner struct {
	FS     fs.FS
	Source string
	Index  StateIndex
}

func (p Planner) Plan(op PkgOp) error {
	walkDir := path.Join(p.Source, op.Pkg)
	return fs.WalkDir(p.FS, walkDir, op.VisitFunc(p.Source, p.Index))
}

type PkgOp struct {
	Pkg   string
	Apply ItemOp
}

func (op PkgOp) VisitFunc(source string, index StateIndex) fs.WalkDirFunc {
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
