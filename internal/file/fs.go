package file

import (
	"fmt"
	"io/fs"
)

type unsupportedError struct {
	FS fs.FS
}

func (u unsupportedError) Error() string {
	return fmt.Sprintf("op not supported by %T", u.FS)
}

func (u unsupportedError) Is(err error) bool {
	return err == fs.ErrInvalid
}

func Symlink(fsys fs.FS, oldname, newname string) error {
	sym, ok := fsys.(interface{ Symlink(string, string) error })
	if !ok {
		return &LinkError{Op: "symlink", Old: oldname, New: newname, Err: unsupportedError{fsys}}
	}
	return sym.Symlink(oldname, newname)
}

func Mkdir(fsys fs.FS, dir string, perm fs.FileMode) error {
	md, ok := fsys.(interface {
		Mkdir(string, fs.FileMode) error
	})
	if !ok {
		return &fs.PathError{Op: "mkdir", Path: dir, Err: unsupportedError{fsys}}
	}
	return md.Mkdir(dir, perm)
}

func Remove(fsys fs.FS, name string) error {
	rm, ok := fsys.(interface{ Remove(string) error })
	if !ok {
		return &fs.PathError{Op: "remove", Path: name, Err: unsupportedError{fsys}}
	}
	return rm.Remove(name)
}
