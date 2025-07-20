package plan

import (
	"io/fs"
	"iter"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type Analyst interface {
	Analyzer
	Specs() iter.Seq2[string, Spec]
}

type Analyzer interface {
	Analyze(op PkgOp, target string) error
}

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
}

type SpecIndex interface {
	Index
	Specs() iter.Seq2[string, Spec]
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
	return New(p.target, p.analyst.Specs()), nil
}

func NewAnalyst(fsys fs.FS, index SpecIndex) analyst {
	return analyst{fsys: fsys, index: index}
}

type analyst struct {
	fsys  fs.FS
	index SpecIndex
}

func (a analyst) Analyze(op PkgOp, target string) error {
	return fs.WalkDir(a.fsys, op.WalkDir(), op.VisitFunc(target, a.index))
}

func (a analyst) Specs() iter.Seq2[string, Spec] {
	return a.index.Specs()
}

func NewPkgOp(pkgDir string, itemOp ItemOp) pkgOp {
	return pkgOp{
		walkDir: pkgDir,
		pkgDir:  pkgDir,
		itemOp:  itemOp,
	}
}

func NewMergePkgOp(pkgDir, mergeItem string, itemOp ItemOp) pkgOp {
	return pkgOp{
		pkgDir:  pkgDir,
		walkDir: path.Join(pkgDir, mergeItem),
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
