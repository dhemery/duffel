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

type ItemOp int

const (
	OpInstall = ItemOp(iota)
	OpRemove  = iota
)

func NewAnalyst(fsys fs.FS, source, target string, index *index, logger *slog.Logger) *analyst {
	a := &analyst{
		fsys:   fsys,
		target: target,
		index:  index,
	}
	finder := NewPkgFinder(fsys)
	merger := NewMerger(finder, a, logger)
	a.install = NewInstallOp(source, target, merger, logger)
	return a
}

type analyst struct {
	fsys    fs.FS
	target  string
	index   *index
	install *installOp
}

func (a *analyst) Analyze(pops ...*PkgOp) (*index, error) {
	for _, pop := range pops {
		err := fs.WalkDir(a.fsys, pop.walkDir, pop.VisitFunc(a.target, a.index, a.install.Apply))
		if err != nil {
			return nil, err
		}
	}
	return a.index, nil
}
