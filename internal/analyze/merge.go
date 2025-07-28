package analyze

import (
	"fmt"
	"log/slog"
	"path"
)

type PkgFinder interface {
	FindPkg(name string) (PkgItem, error)
}

type MergeError struct {
	Name string
	Err  error
}

func (me *MergeError) Error() string {
	return fmt.Sprintf("merge %q: %s", me.Name, me.Err)
}

func (me *MergeError) Unwrap() error {
	return me.Err
}

func NewMerger(pkgFinder PkgFinder, analyst *analyst, logger *slog.Logger) *merger {
	return &merger{
		pkgFinder: pkgFinder,
		analyst:   analyst,
		log:       logger,
	}
}

type merger struct {
	pkgFinder PkgFinder
	analyst   *analyst
	log       *slog.Logger
}

func (m merger) Merge(dir, target string) error {
	pkgItem, err := m.pkgFinder.FindPkg(dir)
	if err != nil {
		return &MergeError{Name: dir, Err: err}
	}

	m.log.Info("merging", "dir", dir, "target", target, "details", pkgItem)
	sourcePkg := path.Join(pkgItem.Source, pkgItem.Pkg)
	pkgOp := NewMergePkgOp(sourcePkg, pkgItem.Item, OpInstall, m.log)
	analyst := NewAnalyst(m.analyst.fsys, pkgItem.Source, target, m.analyst.index, m.log)

	_, err = analyst.Analyze(pkgOp)
	return err
}
