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
	Analyze(op PkgOp, target string) error
}

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
	All() iter.Seq2[string, *file.State]
}

type ItemOp interface {
	Apply(name string, entry fs.DirEntry, indexState *file.State) (*file.State, error)
}

type PkgOp interface {
	WalkDir() string
	VisitFunc(target string, index Index) fs.WalkDirFunc
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
		err := p.analyst.Analyze(op, p.target)
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

func (a analyst) Analyze(op PkgOp, target string) error {
	return fs.WalkDir(a.fsys, op.WalkDir(), op.VisitFunc(target, a.index))
}

func (a analyst) States() iter.Seq2[string, *file.State] {
	return a.index.All()
}

func NewForeignPkgOp(pkgDir, walkDir string, itemOp ItemOp) pkgOp {
	return pkgOp{
		pkgDir:  pkgDir,
		walkDir: walkDir,
		itemOp:  itemOp,
	}
}

func NewPkgOp(pkgDir string, itemOp ItemOp) pkgOp {
	return pkgOp{
		walkDir: pkgDir,
		pkgDir:  pkgDir,
		itemOp:  itemOp,
	}
}

type pkgOp struct {
	pkgDir  string
	walkDir string
	itemOp  ItemOp
}

func (po pkgOp) WalkDir() string {
	return po.walkDir
}

func (po pkgOp) VisitFunc(target string, index Index) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == po.walkDir {
			// Skip the dir being walked.
			return nil
		}

		item := name[len(po.pkgDir)+1:]
		targetItem := path.Join(target, item)
		oldState, err := index.State(targetItem)
		if err != nil {
			return err
		}

		newState, err := po.itemOp.Apply(name, entry, oldState)

		if err == nil || err == fs.SkipDir {
			index.SetState(targetItem, newState)
		}

		return err
	}
}
