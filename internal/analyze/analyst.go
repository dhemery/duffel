package analyze

import (
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

func Analyze(fsys fs.FS, target string, packageOps []*PackageOp, logger *slog.Logger) (*index, error) {
	stater := file.NewStater(fsys)
	index := NewIndex(stater, logger)
	analyst := NewAnalyst(fsys, target, index, logger)
	return analyst.Analyze(packageOps...)
}

func NewAnalyst(fsys fs.FS, target string, index *index, logger *slog.Logger) *Analyst {
	a := &Analyst{
		fsys:   fsys,
		target: target,
		index:  index,
		logger: logger,
	}
	itemizer := NewItemizer(fsys)
	merger := NewMerger(itemizer, a, logger)
	a.install = NewInstall(target, merger, logger)
	return a
}

type Analyst struct {
	fsys    fs.FS
	target  string
	index   *index
	install *Install
	logger  *slog.Logger
}

func (a *Analyst) Analyze(pops ...*PackageOp) (*index, error) {
	for _, pop := range pops {
		err := fs.WalkDir(a.fsys, pop.walkRoot.String(), pop.VisitFunc(a.target, a.index, a.install.Apply, a.logger))
		if err != nil {
			return nil, err
		}
	}
	return a.index, nil
}
