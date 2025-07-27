package plan

import (
	"io/fs"
	"iter"
	"path"
)

type Analyst interface {
	Analyze(ops ...PkgOp) (Specs, error)
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
	p.analyst.Analyze(ops...)
	specs, err := p.analyst.Analyze(ops...)
	if err != nil {
		return Plan{}, err
	}
	return New(p.target, specs), nil
}

type SpecsIndex interface {
	Index
	Specs
}

func NewAnalyst(fsys fs.FS, target string, specs SpecsIndex) analyst {
	return analyst{fsys: fsys, target: target, SpecsIndex: specs}
}

type analyst struct {
	fsys   fs.FS
	target string
	SpecsIndex
}

func (a analyst) Analyze(ops ...PkgOp) (Specs, error) {
	for _, op := range ops {
		err := fs.WalkDir(a.fsys, op.WalkDir(), op.VisitFunc(a.target, a))
		if err != nil {
			return nil, err
		}
	}
	return a.SpecsIndex, nil
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
