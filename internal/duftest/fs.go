package duftest

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing/fstest"
)

const (
	permRead   = 0o444
	permSearch = 0o111
	permWrite  = 0o222
	// fsOp is the prefix for op names in PathError errors returned by FS methods.
	fsOp = "duftest."
)

func NewFS() FS {
	return FS{M: fstest.MapFS{}}
}

type FS struct {
	M fstest.MapFS
}

func (f FS) Open(name string) (fs.File, error) {
	const op = fsOp + "open"
	if _, err := f.check(name, permRead); err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	file, err := f.M.Open(name)
	if err != nil {
		err = &fs.PathError{Op: op, Path: name, Err: err}
	}
	return file, err
}

func (f FS) ReadDir(name string) ([]fs.DirEntry, error) {
	const op = fsOp + "readdir"
	if _, err := f.check(name, fs.ModeDir|permRead); err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	entries, err := f.M.ReadDir(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	return entries, err
}

func (f FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + "stat"
	info, err := f.check(name, permRead)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	return info, nil
}

func (f FS) Symlink(oldname, newname string) error {
	const op = fsOp + "symlink"
	if _, err := f.check(path.Dir(newname), fs.ModeDir|permWrite); err != nil {
		return &fs.PathError{Op: op, Path: newname, Err: err}
	}
	f.M[newname] = &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte(oldname)}
	return nil
}

type searchError struct {
	Path string
	Elem string
	Err  error
}

func (ce searchError) Error() string {
	return fmt.Sprintf("%s ancestor %s unsearchable: %s", ce.Path, ce.Elem, ce.Err)
}

func (ce searchError) Unwrap() error {
	return ce.Err
}

func (f FS) check(name string, want fs.FileMode) (fs.FileInfo, error) {
	if err := f.checkPath(name); err != nil {
		return nil, err
	}
	return f.checkMode(name, want)
}

func (f FS) checkPath(name string) error {
	// The path must be lexically valid for use with fs.FS
	if !fs.ValidPath(name) {
		return fs.ErrInvalid
	}
	// Each ancestor must be a searchable dir
	elems := strings.Split(name, "/")
	for i := range len(elems) - 1 {
		ancestor := path.Join(elems[:i+1]...)
		_, err := f.checkMode(ancestor, fs.ModeDir|permSearch)
		if err != nil {
			return searchError{Path: name, Elem: ancestor, Err: err}
		}
	}
	return nil
}

func (f FS) checkMode(name string, want fs.FileMode) (fs.FileInfo, error) {
	info, err := f.M.Stat(name)
	if err != nil {
		return nil, err
	}

	mode := info.Mode()
	if mode.Type()&want.Type() != want.Type() {
		return nil, modeError{Path: name, Want: want, Got: mode, Err: fs.ErrInvalid}
	}
	if mode.Perm()&want.Perm() == 0 {
		return nil, modeError{Path: name, Want: want, Got: mode, Err: fs.ErrPermission}
	}
	return info, nil
}

type modeError struct {
	Path string
	Want fs.FileMode
	Got  fs.FileMode
	Err  error
}

func (me modeError) Error() string {
	return fmt.Sprintf("%s want mode %s, got %s: %s", me.Path, me.Want, me.Got, me.Err.Error())
}

func (me modeError) Unwrap() error {
	return me.Err
}
