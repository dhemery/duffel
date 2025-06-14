package plan

import (
	"errors"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/item"
)

type Advisor interface {
	Advise(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type PkgAnalyst struct {
	FS        fs.FS
	Target    string
	Pkg       string
	SourcePkg string
	Specs     item.Index
	Advisor   Advisor
}

func NewPkgAnalyst(fsys fs.FS, target, source, pkg string, index item.Index, advisor Advisor) PkgAnalyst {
	return PkgAnalyst{
		FS:        fsys,
		Target:    target,
		SourcePkg: path.Join(source, pkg),
		Pkg:       pkg,
		Specs:     index,
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
	spec, ok := pa.Specs[item]
	if !ok {
		targetItem := path.Join(pa.Target, item)
		info, err := file.Lstat(pa.FS, targetItem)
		switch {
		case err == nil:
			state := &file.State{Mode: info.Mode()}
			if info.Mode()&fs.ModeSymlink != 0 {
				dest, err := file.ReadLink(pa.FS, targetItem)
				if err != nil {
					return err
				}
				state.Dest = dest
			}
			spec.Current = state
			spec.Desired = state
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
		pa.Specs[item] = spec
	}

	advice, err := pa.Advisor.Advise(pa.Pkg, item, entry, spec.Desired)
	if err != nil {
		return err
	}

	spec.Desired = advice
	pa.Specs[item] = spec
	return nil
}
