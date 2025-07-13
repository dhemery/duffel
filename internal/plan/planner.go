package plan

import (
	"io/fs"
	"iter"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type analyzer interface {
	Analyze(ops []PkgOp) error
}

type states interface {
	Sorted() iter.Seq2[string, *file.State]
}

type Planner struct {
	Target   string
	Analyzer analyzer
	States   states
}

func (p Planner) Plan(ops ...PkgOp) (Plan, error) {
	err := p.Analyzer.Analyze(ops)
	if err != nil {
		return Plan{}, err
	}
	return New(p.Target, p.States), nil
}

type Analyst struct {
	FS fs.FS
}

func (a Analyst) Analyze(ops []PkgOp) error {
	for _, op := range ops {
		err := fs.WalkDir(a.FS, op.WalkDir(), op.VisitFunc())
		if err != nil {
			return err
		}
	}
	return nil
}

type itemOp interface {
	Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
}

type PkgOp struct {
	Source string
	Pkg    string
	ItemOp itemOp
	Index  index
}

func (po PkgOp) WalkDir() string {
	return path.Join(po.Source, po.Pkg)
}

func (po PkgOp) VisitFunc() fs.WalkDirFunc {
	walkDir := po.WalkDir()
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == walkDir {
			// Skip the dir being walked. It is not an item.
			return nil
		}

		item := name[len(walkDir)+1:]
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
