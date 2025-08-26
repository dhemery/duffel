package plan

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

var (
	errIsPackage    = errors.New("is a duffel package")
	errIsSource     = errors.New("is a duffel source")
	errNotInPackage = errors.New("not in a duffel package")
)

type itemizer struct {
	fsys fs.ReadLinkFS
}

// itemize returns a [sourcePath] describing the named file.
// If the file is not in a duffel source directory,
// the method returns an error.
func (i itemizer) itemize(name string) (sourcePath, error) {
	source, err := file.SourceDir(i.fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return sourcePath{}, errNotInPackage
	}

	if err != nil {
		return sourcePath{}, err
	}

	if name == source {
		return sourcePath{}, errIsSource
	}

	pkgItem := name[len(source)+1:]
	pkg, item, found := strings.Cut(pkgItem, "/")
	if !found {
		return sourcePath{}, errIsPackage
	}

	return newSourcePath(source, pkg, item), nil
}
