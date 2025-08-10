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

// Analyze applies packageOps
// to identify the current and desired states
// of files in the target tree
// that correspond to the items in the packages.
func Analyze(fsys fs.FS, target string, packageOps []*PackageOp, logger *slog.Logger) (*index, error) {
	stater := file.NewStater(fsys)
	index := NewIndex(stater, logger)
	analyst := NewAnalyst(fsys, target, index)
	return analyst.Analyze(logger, packageOps...)
}

func NewAnalyst(fsys fs.FS, target string, index *index) *Analyst {
	a := &Analyst{
		fsys:   fsys,
		target: target,
		index:  index,
	}
	itemizer := NewItemizer(fsys)
	merger := NewMerger(itemizer, a)
	a.install = NewInstall(merger)
	return a
}

type Analyst struct {
	fsys    fs.FS
	target  string
	index   *index
	install *Install
}

func (a *Analyst) Analyze(l *slog.Logger, pops ...*PackageOp) (*index, error) {
	for _, pop := range pops {
		err := fs.WalkDir(a.fsys, pop.walkRoot.String(), pop.VisitFunc(a.target, a.index, a.install.Apply, l))
		if err != nil {
			return nil, err
		}
	}
	return a.index, nil
}
