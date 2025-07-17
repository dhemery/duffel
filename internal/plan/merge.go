package plan

import (
	"path"
)

type PkgFinder interface {
	FindPkg(name string) (string, error)
}

func NewMerger(pkgFinder PkgFinder, analyzer Analyzer) merger {
	return merger{pkgFinder, analyzer}
}

type merger struct {
	pkgFinder PkgFinder
	analyzer  Analyzer
}

func (m merger) Merge(name, target string) error {
	sourcePkg, err := m.pkgFinder.FindPkg(name)
	if err != nil {
		return err
	}

	source := path.Dir(sourcePkg)
	install := NewInstallOp(source, target, m)
	pkgOp := NewForeignPkgOp(sourcePkg, name, install)

	return m.analyzer.Analyze(pkgOp, target)
}
