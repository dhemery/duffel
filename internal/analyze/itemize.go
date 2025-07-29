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

func NewItemizer(fsys fs.FS) itemizer {
	return itemizer{fsys}
}

type itemizer struct {
	fsys fs.FS
}

// Itemize returns a [SourcePath] describing the named file.
// The file must be an item in a package in a duffel source.
// A duffel source is a directory that has an entry named .duffel.
// A package is a directory child of a duffel source.
func (pf itemizer) Itemize(name string) (SourcePath, error) {
	source, err := pf.findSource(name)
	if err != nil {
		return SourcePath{}, err
	}

	if name == source {
		return SourcePath{}, ErrIsSource
	}

	pkgItem := name[len(source)+1:]
	pkg, item, found := strings.Cut(pkgItem, "/")
	if !found {
		return SourcePath{}, ErrIsPackage
	}

	return SourcePath{source, pkg, item}, nil
}

func (pf itemizer) findSource(name string) (string, error) {
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
