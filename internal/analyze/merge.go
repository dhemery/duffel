package analyze

import (
	"fmt"
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

func NewMerger(pkgFinder PkgFinder, analyst analyst) merger {
	return merger{pkgFinder, analyst}
}

type merger struct {
	pkgFinder PkgFinder
	analyst   analyst
}

func (m merger) Merge(name, target string) error {
	pkgItem, err := m.pkgFinder.FindPkg(name)
	if err != nil {
		return &MergeError{Name: name, Err: err}
	}

	install := NewInstallOp(pkgItem.Source, target, m)
	sourcePkg := path.Join(pkgItem.Source, pkgItem.Pkg)
	pkgOp := NewMergePkgOp(sourcePkg, pkgItem.Item, install)

	_, err = m.analyst.Analyze(pkgOp)
	return err
}
