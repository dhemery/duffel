package rules

import (
	"errors"
	"fmt"
	"io/fs"
)

func CheckInstallPath(f fs.FS, p string) error {
	info, err := fs.Stat(f, p)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotExist
	}
	if err != nil {
		return err
	}
	if err = checkReadableDir(info); err != nil {
		fmt.Println("error from", p, err.Error())
		return err
	}
	return checkNotInDuffelDir(f, p)
}
