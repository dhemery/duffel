package file

import (
	"errors"
	"io/fs"
	"path"
)

func SourceDir(fsys fs.ReadLinkFS, name string) (string, error) {
	dfName := path.Join(name, ".duffel")
	_, err := fsys.Lstat(dfName)

	if err == nil {
		return name, nil
	}

	if dfName == ".duffel" {
		return "", fs.ErrNotExist
	}

	if errors.Is(err, fs.ErrNotExist) {
		return SourceDir(fsys, path.Dir(name))
	}

	return "", err
}
