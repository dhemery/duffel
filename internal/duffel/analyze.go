package duffel

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
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

func (pa PkgAnalyst) VisitPath(p string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if p == pa.SourcePkg {
		// Source pkg is not an item.
		return nil
	}
	item, _ := filepath.Rel(pa.SourcePkg, p)
	itemGap, ok := pa.TargetGap[item]
	if !ok {
		targetItem := path.Join(pa.Target, item)
		info, err := fs.Stat(pa.FS, targetItem)
		switch {
		case err == nil:
			itemGap = NewFileGap(info.Mode(), "")
		case !errors.Is(err, fs.ErrNotExist):
			// TODO: Record the error in the file gap
			return err
		}
		pa.TargetGap[item] = itemGap
	}

	advice, err := pa.Advisor.Advise(pa.Pkg, item, entry, itemGap.Desired)
	if err != nil {
		return err
	}

	itemGap.Desired = advice
	pa.TargetGap[item] = itemGap
	return nil
}
