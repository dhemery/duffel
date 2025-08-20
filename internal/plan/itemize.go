package plan

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

var (
	ErrIsPackage    = errors.New("is a duffel package")
	ErrIsSource     = errors.New("is a duffel source")
	ErrNotInPackage = errors.New("not in a duffel package")
)

func NewItemizer(fsys fs.ReadLinkFS) itemizer {
	return itemizer{fsys}
}

type itemizer struct {
	fsys fs.ReadLinkFS
}

// Itemize returns a [SourcePath] describing the named file.
// If the file is not in a duffel source directory,
// the method returns an error.
func (i itemizer) Itemize(name string) (SourcePath, error) {
	source, err := file.SourceDir(i.fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return SourcePath{}, ErrNotInPackage
	}

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

	return NewSourcePath(source, pkg, item), nil
}
