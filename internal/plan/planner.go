package plan

import (
	"io/fs"
	"iter"
	"path"
)

type Analyst interface {
	Analyze(op PkgOp, target string) error
}

type Index interface {
	State(name string) (*State, error)
	SetState(item string, state *State)
}

type Specs interface {
	All() iter.Seq2[string, Spec]
}

type PkgOp interface {
	WalkDir() string
	VisitFunc(target string, index Index) fs.WalkDirFunc
}

type SpecsAnalyst interface {
	Analyst
	Specs
}

func NewPlanner(target string, analyzer SpecsAnalyst) planner {
	return planner{
		target:  target,
		analyst: analyzer,
	}
}

type planner struct {
	target  string
	analyst SpecsAnalyst
}

func (p planner) Plan(ops []PkgOp) (Plan, error) {
	for _, op := range ops {
		err := p.analyst.Analyze(op, p.target)
		if err != nil {
			return Plan{}, err
		}
	}
	return New(p.target, p.analyst), nil
}

type SpecsIndex interface {
	Index
	Specs
}

func NewAnalyst(fsys fs.FS, specs SpecsIndex) analyst {
	return analyst{fsys: fsys, SpecsIndex: specs}
}

type analyst struct {
	fsys fs.FS
	SpecsIndex
}

func (a analyst) Analyze(op PkgOp, target string) error {
	return fs.WalkDir(a.fsys, op.WalkDir(), op.VisitFunc(target, a))
}

type ItemOp interface {
	Apply(name string, entry fs.DirEntry, indexState *State) (*State, error)
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
