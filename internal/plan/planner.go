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
	analyst := NewAnalyzer(fsys, target, index)
	return &planner{fsys, target, analyst}
}

type planner struct {
	fsys    fs.FS
	target  string
	analyst *analyzer
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

func NewAnalyzer(fsys fs.FS, target string, index *index) *analyzer {
	analyst := &analyzer{
		fsys:   fsys,
		target: target,
		index:  index,
	}
	itemizer := NewItemizer(fsys)
	merger := NewMerger(itemizer, analyst)
	analyst.install = &installer{merger}
	return analyst
}

type analyzer struct {
	fsys    fs.FS
	target  string
	index   *index
	install *installer
}

func (a *analyzer) Analyze(op *PackageOp, l *slog.Logger) error {
	return fs.WalkDir(a.fsys, op.walkRoot.String(), op.VisitFunc(a.target, a.index, a.install.Analyze, l))
}
