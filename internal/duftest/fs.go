package duftest

import (
	"errors"
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
	M   fstest.MapFS
	Dir string
}

func (f FS) Open(name string) (fs.File, error) {
	const op = fsOp + "open"
	full := path.Join(f.Dir, name)
	return nil, &fs.PathError{Op: op, Path: full, Err: errors.ErrUnsupported}
}

func (f FS) ReadDir(name string) ([]fs.DirEntry, error) {
	const op = fsOp + "readdir"
	full := path.Join(f.Dir, name)
	if _, err := f.check(full, fs.ModeDir|permRead); err != nil {
		return nil, &fs.PathError{Op: op, Path: full, Err: err}
	}
	entries, err := f.M.ReadDir(full)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: full, Err: err}
	}
	return entries, err
}

func (f FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + "stat"
	full := path.Join(f.Dir, name)
	info, err := f.check(full, permRead)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: full, Err: err}
	}
	return info, nil
}

func (f FS) Lstat(name string) (fs.FileInfo, error) {
	const op = fsOp + "lstat"
	full := path.Join(f.Dir, name)
	info, err := f.check(full, permRead)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: full, Err: err}
	}
	return info, nil
}

func (f FS) Symlink(oldname, newname string) error {
	const op = fsOp + "symlink"
	full := path.Join(f.Dir, newname)
	if _, err := f.check(path.Dir(full), fs.ModeDir|permWrite); err != nil {
		return &fs.PathError{Op: op, Path: full, Err: err}
	}
	f.M[full] = &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte(oldname)}
	return nil
}

func (f FS) Sub(dir string) (fs.FS, error) {
	const op = fsOp + "sub"
	full := path.Join(f.Dir, dir)
	if !fs.ValidPath(full) {
		return nil, &fs.PathError{Op: op, Path: full, Err: fs.ErrInvalid}
	}
	return FS{M: f.M, Dir: full}, nil
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
