package file

import (
	"errors"
	"io/fs"
	"path"
	"strings"
)

var (
	ErrIsPackage    = errors.New("is a duffel package")
	ErrIsSource     = errors.New("is a duffel source")
	ErrNotInPackage = errors.New("not in a duffel package")
)

func NewPkgFinder(fsys fs.FS) pkgFinder {
	return pkgFinder{fsys}
}

type pkgFinder struct {
	fsys fs.FS
}

// FindPkg returns the package directory that contains the named file.
// A package directory is a directory in a duffel source.
// A duffel source is a directory that contains an entry named .duffel.
func (pf pkgFinder) FindPkg(name string) (string, error) {
	source, err := pf.findSource(name)
	if err != nil {
		return "", err
	}

	if name == source {
		return "", ErrIsSource
	}

	pkgItem := name[len(source)+1:]
	pkg, _, found := strings.Cut(pkgItem, "/")
	if !found {
		return "", ErrIsPackage
	}

	sourcePkg := path.Join(source, pkg)
	return sourcePkg, nil
}

func (pf pkgFinder) findSource(name string) (string, error) {
	if name == "." {
		return "", ErrNotInPackage
	}

	dfName := path.Join(name, ".duffel")
	_, err := fs.Lstat(pf.fsys, dfName)

	if errors.Is(err, fs.ErrNotExist) {
		return pf.findSource(path.Dir(name))
	}

	if err != nil {
		return "", err
	}

	return name, nil
}
