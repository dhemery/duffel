package plan

import (
	"io/fs"
	"iter"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

func NewAnalyst(fsys fs.FS, index Index) analyst {
	return analyst{fsys: fsys, index: index}
}

type PkgOp interface {
	VisitFunc() fs.WalkDirFunc
	WalkDir() string
}

type Analyzer interface {
	Analyze(op PkgOp) error
}

type Analyst interface {
	Analyzer
	States() iter.Seq2[string, *file.State]
}

func NewPlanner(target string, analyst Analyst) planner {
	return planner{
		target:  target,
		analyst: analyst,
	}
}

type ItemOp interface {
	Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
	All() iter.Seq2[string, *file.State]
}

func NewPkgOp(source, pkg string, itemOp ItemOp, index Index) pkgOp {
	return pkgOp{
		source: source,
		pkg:    pkg,
		itemOp: itemOp,
		index:  index,
	}
}

type pkgOp struct {
	source string
	pkg    string
	itemOp ItemOp
	index  Index
}

type planner struct {
	target  string
	analyst Analyst
}

func (p planner) Plan(ops []PkgOp) (Plan, error) {
	for _, op := range ops {
		err := p.analyst.Analyze(op)
		if err != nil {
			return Plan{}, err
		}
	}
	return New(p.target, p.analyst.States()), nil
}

type analyst struct {
	fsys  fs.FS
	index Index
}

func (a analyst) Analyze(op PkgOp) error {
	return fs.WalkDir(a.fsys, op.WalkDir(), op.VisitFunc())
}

func (a analyst) States() iter.Seq2[string, *file.State] {
	return a.index.All()
}

func (po pkgOp) WalkDir() string {
	return path.Join(po.source, po.pkg)
}

func (po pkgOp) VisitFunc() fs.WalkDirFunc {
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
		oldState, err := po.index.State(item)
		if err != nil {
			return err
		}

		newState, err := po.itemOp.Apply(po.pkg, item, entry, oldState)

		if err == nil || err == fs.SkipDir {
			po.index.SetState(item, newState)
		}

		return err
	}
}
