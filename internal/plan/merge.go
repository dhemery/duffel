package plan

import (
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type PkgFinder interface {
	FindPkg(name string) (file.PkgItem, error)
}

func NewMerger(pkgFinder PkgFinder, analyzer Analyzer) merger {
	return merger{pkgFinder, analyzer}
}

type merger struct {
	pkgFinder PkgFinder
	analyzer  Analyzer
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
