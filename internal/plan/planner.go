// Package plan creates a plan to change the target tree
// to realize a series of package operations.
package plan

import (
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// NewPlanner returns a planner that plans how to change the tree rooted at target.
func NewPlanner(fsys fs.FS, target string) *planner {
	stater := file.NewStater(fsys)
	index := NewIndex(stater)
	analyst := NewAnalyst(fsys, target, index)
	return &planner{fsys, target, analyst}
}

type planner struct {
	fsys    fs.FS
	target  string
	analyst *Analyst
}

// Plan creates a plan to realize ops in p's target tree.
func (p planner) Plan(ops []*PackageOp, l *slog.Logger) (Plan, error) {
	for _, op := range ops {
		if err := p.analyst.Analyze(op, l); err != nil {
			return Plan{}, err
		}

	}
	return NewPlan(p.target, p.analyst.index), nil
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
