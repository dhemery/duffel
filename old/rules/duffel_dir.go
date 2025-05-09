package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

const (
	duffelFileName = ".duffel"
)

var (
	ErrNotDuffelDir = errors.New("is not a duffel dir")
	ErrIsDuffelDir  = errors.New("is a duffel dir")
)

func CheckIsDuffelDir(f fs.FS, p string) error {
	duffelInfo, err := fs.Stat(f, p)
	if err != nil {
		return ErrNotExist
	}
	if !duffelInfo.IsDir() {
		return ErrNotDir
	}

	duffelFilePath := path.Join(p, duffelFileName)
	duffelFileInfo, err := fs.Stat(f, duffelFilePath)
	if err != nil {
		return ErrNotDuffelDir
	}
	if !duffelFileInfo.Mode().IsRegular() {
		return fmt.Errorf("%s: %w", duffelFilePath, ErrNotFile)
	}

	return nil
}

func checkNotInDuffelDir(f fs.FS, p string) error {
	if err := checkNotDuffelDir(f, p); err != nil {
		return err
	}
	if p == fsRoot {
		return nil
	}
	parent := path.Dir(p)
	return checkNotInDuffelDir(f, parent)
}

func checkNotDuffelDir(f fs.FS, p string) error {
	duffelFilePath := path.Join(p, duffelFileName)
	_, err := fs.Stat(f, duffelFilePath)
	if err == nil {
		return fmt.Errorf("%s: %w", p, ErrIsDuffelDir)
	}
	return nil
}
