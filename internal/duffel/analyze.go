package duffel

import (
	"errors"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/files"
)

type ItemAdvisor interface {
	Advise(pkg, item string, entry fs.DirEntry, priorAdvice *FileState) (*FileState, error)
}

type PkgAnalyst struct {
	FS        fs.FS
	Target    string
	Pkg       string
	SourcePkg string
	TargetGap TargetGap
	Advisor   ItemAdvisor
}

func NewPkgAnalyst(fsys fs.FS, target, source, pkg string, tg TargetGap, advisor ItemAdvisor) PkgAnalyst {
	return PkgAnalyst{
		FS:        fsys,
		Target:    target,
		SourcePkg: path.Join(source, pkg),
		Pkg:       pkg,
		TargetGap: tg,
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
	fileGap, ok := pa.TargetGap[item]
	if !ok {
		targetItem := path.Join(pa.Target, item)
		info, err := files.Lstat(pa.FS, targetItem)
		switch {
		case err == nil:
			state := &FileState{Mode: info.Mode()}
			if info.Mode()&fs.ModeSymlink != 0 {
				dest, err := files.ReadLink(pa.FS, targetItem)
				if err != nil {
					return err
				}
				state.Dest = dest
			}
			fileGap.Current = state
			fileGap.Desired = state
		case !errors.Is(err, fs.ErrNotExist):
			// TODO: Record the error in the file gap
			return err
		}
		pa.TargetGap[item] = fileGap
	}

	advice, err := pa.Advisor.Advise(pa.Pkg, item, entry, fileGap.Desired)
	if err != nil {
		return err
	}

	fileGap.Desired = advice
	pa.TargetGap[item] = fileGap
	return nil
}
