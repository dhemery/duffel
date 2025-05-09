package rules

import (
	"errors"
	"io/fs"
)

func CheckPackagePath(f fs.FS, p string) error {
	info, err := fs.Stat(f, p)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotExist
	}
	if err != nil {
		return err
	}

	return checkReadableDir(info)
}
