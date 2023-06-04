package main

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

const (
	fsRoot       = "."
	farmFileName = ".farm"
)

func CheckPackagePath(f fs.FS, p string) error {
	info, err := fs.Stat(f, p)
	if err != nil {
		return err
	}

	return checkReadableDir(info)
}

func CheckInstallPath(f fs.FS, p string) error {
	info, err := fs.Stat(f, p)
	if err != nil {
		return err
	}
	if err = checkReadableDir(info); err != nil {
		return err
	}
	return checkNotInFarm(f, p)
}

func checkNotInFarm(f fs.FS, p string) error {
	if err := checkNotFarm(f, p); err != nil {
		return err
	}
	if p == fsRoot {
		return nil
	}
	parent := path.Dir(p)
	return checkNotInFarm(f, parent)
}

func checkNotFarm(f fs.FS, p string) error {
	farmFilePath := path.Join(p, farmFileName)
	_, err := fs.Stat(f, farmFilePath)
	if err == nil {
		return fmt.Errorf("in farm %s: %w", p, fs.ErrPermission)
	}
	return nil
}

func checkCanCreate(f fs.FS, p string) error {
	parent := path.Dir(p)
	info, err := fs.Stat(f, parent)
	if errors.Is(err, fs.ErrNotExist) {
		return checkCanCreate(f, parent)
	}
	if err != nil {
		return err
	}
	mode := info.Mode()
	if !isWriteable(mode) {
		return fmt.Errorf("%s: cannot write (mode %o): %w", p, mode, fs.ErrPermission)
	}
	return nil
}

func checkReadableDir(info fs.FileInfo) error {
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %w", fs.ErrInvalid)
	}
	mode := info.Mode()
	if !isReadable(mode) {
		return fmt.Errorf("cannot read (mode %o): %w", mode, fs.ErrPermission)
	}
	return nil
}

func isReadable(m fs.FileMode) bool {
	const readBits = 0444
	return m&readBits != 0
}

func isWriteable(m fs.FileMode) bool {
	const writeBits = 0222
	return m&writeBits != 0
}
