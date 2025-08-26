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

// itemize returns a [SourcePath] describing the named file.
// If the file is not in a duffel source directory,
// the method returns an error.
func (i itemizer) itemize(name string) (SourcePath, error) {
	source, err := file.SourceDir(i.fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return SourcePath{}, errNotInPackage
	}

	if err != nil {
		return SourcePath{}, err
	}

	if name == source {
		return SourcePath{}, errIsSource
	}

	pkgItem := name[len(source)+1:]
	pkg, item, found := strings.Cut(pkgItem, "/")
	if !found {
		return SourcePath{}, errIsPackage
	}

	return newSourcePath(source, pkg, item), nil
}
