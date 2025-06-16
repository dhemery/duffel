package plan

import (
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type Advisor interface {
	Advise(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type states interface {
	Desired(item string) (*file.State, error)
	SetDesired(item string, state *file.State)
}

type PkgAnalyst struct {
	FS        fs.FS
	Target    string
	Pkg       string
	SourcePkg string
	States    states
	Advisor   Advisor
}

func NewPkgAnalyst(fsys fs.FS, target, source, pkg string, states states, advisor Advisor) PkgAnalyst {
	return PkgAnalyst{
		FS:        fsys,
		SourcePkg: path.Join(source, pkg),
		Pkg:       pkg,
		States:    states,
		Advisor:   advisor,
	}
}

func (pa PkgAnalyst) Analyze() error {
	return fs.WalkDir(pa.FS, pa.SourcePkg, pa.VisitPath)
}

func (pa PkgAnalyst) VisitPath(name string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if name == pa.SourcePkg {
		// Source pkg is not an item.
		return nil
	}

	item := name[len(pa.SourcePkg)+1:]
	state, err := pa.States.Desired(item)
	if err != nil {
		return err
	}

	advice, err := pa.Advisor.Advise(pa.Pkg, item, entry, state)
	if err != nil {
		return err
	}

	pa.States.SetDesired(item, advice)
	return nil
}
