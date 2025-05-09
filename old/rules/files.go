package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

const (
	fsRoot = "."
)

var (
	ErrCannotExecute = errors.New("cannot execute")
	ErrCannotRead    = errors.New("cannot read")
	ErrCannotWrite   = errors.New("cannot write")
	ErrIsFile        = errors.New("is existing file")
	ErrNotDir        = errors.New("is not a directory")
	ErrNotExist      = errors.New("does not exist")
	ErrNotFile       = errors.New("is not a regular file")
	ErrNotFileOrDir  = errors.New("is not a file or directory")
)

func checkCanCreate(f fs.FS, p string) error {
	parent := path.Dir(p)
	info, err := fs.Stat(f, parent)
	if !errors.Is(err, fs.ErrNotExist) {
		return checkCanCreate(f, parent)
	}
	if err != nil {
		return err
	}
	return checkCanWrite(info)
}

func checkReadableDir(info fs.FileInfo) error {
	if !info.IsDir() {
		return ErrNotDir
	}
	return checkCanRead(info)
}

const (
	readBits  = 0444
	writeBits = 0222
	execBits  = 0111
)

func checkCanRead(info fs.FileInfo) error {
	perm := info.Mode().Perm()
	if perm&readBits == 0 {
		return fmt.Errorf("%w: perm %04o", ErrCannotRead, perm)
	}
	return nil
}

func checkCanWrite(info fs.FileInfo) error {
	perm := info.Mode().Perm()
	if perm&writeBits == 0 {
		return fmt.Errorf("%w: perm %04o", ErrCannotWrite, perm)
	}
	return nil
}

func checkCanExecute(info fs.FileInfo) error {
	perm := info.Mode().Perm()
	if perm&execBits == 0 {
		return fmt.Errorf("%w: perm %04o", ErrCannotExecute, perm)
	}
	return nil
}
