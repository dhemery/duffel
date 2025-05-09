package rules

import (
	"errors"
	"io/fs"
)

// CheckTargetPath returns whether it is okay to create a link at path,
// and an error if path is not a valid target.
// This method assumes that all ancestor paths are valid targets.
func CheckTargetPath(f fs.FS, path string) (bool, error) {
	info, err := fs.Stat(f, path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	mode := info.Mode()
	if mode.IsRegular() {
		return false, ErrIsFile
	}
	return false, nil
}
