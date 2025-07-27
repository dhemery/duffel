package analyze

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

// PkgItem describes an item in a duffel package.
type PkgItem struct {
	Source string // The item's duffel source directory.
	Pkg    string // The name of the item's package.
	Item   string // The name of the item relative to the package directory.
}

func NewPkgFinder(fsys fs.FS) pkgFinder {
	return pkgFinder{fsys}
}

type pkgFinder struct {
	fsys fs.FS
}

// FindPkg returns a [PkgItem] describing the named file.
// The file must be an item in a package in a duffel source.
// A duffel source is a directory that has an entry named .duffel.
// A package is a directory child of a duffel source.
func (pf pkgFinder) FindPkg(name string) (PkgItem, error) {
	source, err := pf.findSource(name)
	if err != nil {
		return PkgItem{}, err
	}

	if name == source {
		return PkgItem{}, ErrIsSource
	}

	pkgItem := name[len(source)+1:]
	pkg, item, found := strings.Cut(pkgItem, "/")
	if !found {
		return PkgItem{}, ErrIsPackage
	}

	return PkgItem{source, pkg, item}, nil
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
