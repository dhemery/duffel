// Package plan identifies the current and planned states
// of each file in the target tree
// that will have to change in order to achieve
// a given sequence of package goals.
package plan

import (
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

func NewPlanner(fsys fs.FS, target string) *Planner {
	stater := file.NewStater(fsys)
	index := NewIndex(stater)
	analyst := NewAnalyst(fsys, target, index)
	return &Planner{fsys, target, analyst}
}

type Planner struct {
	FS      fs.FS
	Target  string
	analyst *Analyst
}

func (p Planner) Plan(pkgOps []*PackageOp, l *slog.Logger) (Plan, error) {
	for _, pop := range pkgOps {
		err := p.analyst.Analyze(pop, l)
		if err != nil {
			return Plan{}, err
		}

	}
	return NewPlan(p.Target, p.analyst.index), nil
}

func NewAnalyst(fsys fs.FS, target string, index *index) *Analyst {
	analyst := &Analyst{
		fsys:   fsys,
		target: target,
		index:  index,
	}
	itemizer := NewItemizer(fsys)
	merger := NewMerger(itemizer, analyst)
	analyst.install = NewInstall(merger)
	return analyst
}

type Analyst struct {
	fsys    fs.FS
	target  string
	index   *index
	install *Install
}

func (a *Analyst) Analyze(pop *PackageOp, l *slog.Logger) error {
	return fs.WalkDir(a.fsys, pop.walkRoot.String(), pop.VisitFunc(a.target, a.index, a.install.Apply, l))
}
