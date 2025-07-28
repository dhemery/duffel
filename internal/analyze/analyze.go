package analyze

import (
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
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
	a.install = NewInstallOp(target, merger, logger)
	return a
}

type Analyst struct {
	fsys    fs.FS
	target  string
	index   *index
	install *installOp
	logger  *slog.Logger
}

func (a *Analyst) Analyze(pops ...*PkgOp) (*index, error) {
	for _, pop := range pops {
		err := fs.WalkDir(a.fsys, pop.root.String(), pop.VisitFunc(a.target, a.index, a.install.Apply, a.logger))
		if err != nil {
			return nil, err
		}
	}
	return a.index, nil
}
