package plan

import (
	"path"
)

type PkgFinder interface {
	FindPkg(name string) (PkgItem, error)
}

func NewMerger(pkgFinder PkgFinder, analyzer Analyst) merger {
	return merger{pkgFinder, analyzer}
}

type merger struct {
	pkgFinder PkgFinder
	analyzer  Analyst
}

func (m merger) Merge(name, target string) error {
	pkgItem, err := m.pkgFinder.FindPkg(name)
	if err != nil {
		return err
	}

	install := NewInstallOp(pkgItem.Source, target, m)
	sourcePkg := path.Join(pkgItem.Source, pkgItem.Pkg)
	pkgOp := NewMergePkgOp(sourcePkg, pkgItem.Item, install)

	return m.analyzer.Analyze(pkgOp, target)
}
