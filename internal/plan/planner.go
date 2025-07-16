package plan

import (
	"io/fs"
	"iter"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type Analyst interface {
	Analyzer
	States() iter.Seq2[string, *file.State]
}
type Analyzer interface {
	Analyze(op PkgOp) error
}

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
	All() iter.Seq2[string, *file.State]
}

type ItemOp interface {
	Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type PkgOp interface {
	WalkDir() string
	VisitFunc(index Index) fs.WalkDirFunc
}

func NewPlanner(target string, analyst Analyst) planner {
	return planner{
		target:  target,
		analyst: analyst,
	}
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

func NewAnalyst(fsys fs.FS, index Index) analyst {
	return analyst{fsys: fsys, index: index}
}

type analyst struct {
	fsys  fs.FS
	index Index
}

func (a analyst) Analyze(op PkgOp) error {
	return fs.WalkDir(a.fsys, op.WalkDir(), op.VisitFunc(a.index))
}

func (a analyst) States() iter.Seq2[string, *file.State] {
	return a.index.All()
}

func NewForeignPkgOp(pkgDir, walkDir string, itemOp ItemOp) pkgOp {
	return pkgOp{
		walkDir: walkDir,
		pkgDir:  pkgDir,
		pkg:     path.Base(pkgDir),
		itemOp:  itemOp,
	}
}

func NewPkgOp(source, pkg string, itemOp ItemOp) pkgOp {
	pkgDir := path.Join(source, pkg)
	return pkgOp{
		walkDir: pkgDir,
		pkgDir:  pkgDir,
		pkg:     pkg,
		itemOp:  itemOp,
	}
}

type pkgOp struct {
	target  string
	walkDir string
	pkgDir  string
	pkg     string
	itemOp  ItemOp
}

func (po pkgOp) WalkDir() string {
	return po.walkDir
}

func (po pkgOp) VisitFunc(index Index) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == po.walkDir {
			// Skip the dir being walked. It is not an item.
			return nil
		}

		item := name[len(po.walkDir)+1:]
		oldState, err := index.State(item)
		if err != nil {
			return err
		}

		newState, err := po.itemOp.Apply(po.pkg, item, entry, oldState)

		if err == nil || err == fs.SkipDir {
			index.SetState(item, newState)
		}

		return err
	}
}
