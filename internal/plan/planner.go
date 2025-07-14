package plan

import (
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

func NewAnalyzer(fsys fs.FS) analyzer {
	return analyzer{fsys}
}

type PkgOp interface {
	VisitFunc() fs.WalkDirFunc
	WalkDir() string
}

type Analyzer interface {
	Analyze(op PkgOp) error
}

func NewPlanner(target string, analyzer Analyzer, states States) planner {
	return planner{
		target:   target,
		analyzer: analyzer,
		states:   states,
	}
}

type ItemOp interface {
	Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
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
	target   string
	analyzer Analyzer
	states   States
}

func (p planner) Plan(ops ...PkgOp) (Plan, error) {
	for _, op := range ops {
		err := p.analyzer.Analyze(op)
		if err != nil {
			return Plan{}, err
		}
	}
	return New(p.target, p.states), nil
}

type analyzer struct {
	FS fs.FS
}

func (a analyzer) Analyze(op PkgOp) error {
	return fs.WalkDir(a.FS, op.WalkDir(), op.VisitFunc())
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
