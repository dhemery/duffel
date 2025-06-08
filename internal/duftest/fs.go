package duftest

import (
	"fmt"
	"io/fs"
	"path"
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
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	if err := f.checkRead(name); err != nil {
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
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	if err := f.checkDirRead(name); err != nil {
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
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	info, err := f.stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	return info, nil
}

func (f FS) Symlink(oldname, newname string) error {
	const op = fsOp + "symlink"
	if !fs.ValidPath(newname) {
		return &fs.PathError{Op: op, Path: newname, Err: fs.ErrInvalid}
	}
	if err := f.checkDirWrite(path.Dir(newname)); err != nil {
		return &fs.PathError{Op: op, Path: newname, Err: err}
	}
	f.M[newname] = &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte(oldname)}
	return nil
}

func (f FS) stat(name string) (fs.FileInfo, error) {
	if err := f.checkDirSearch(path.Dir(name)); err != nil {
		return nil, err
	}

	return f.M.Stat(name)
}

func (f FS) checkRead(name string) error {
	return f.checkMode(name, permRead)
}

func (f FS) checkDirRead(dir string) error {
	return f.checkMode(dir, fs.ModeDir|permRead)
}

func (f FS) checkDirWrite(dir string) error {
	return f.checkMode(dir, fs.ModeDir|permWrite)
}

func (f FS) checkDirSearch(dir string) error {
	if dir == "." {
		return nil
	}
	return f.checkMode(dir, fs.ModeDir|permSearch)
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

func (f FS) checkMode(name string, want fs.FileMode) error {
	info, err := f.stat(name)
	if err != nil {
		return err
	}

	wantType := want & fs.ModeType
	wantPerm := want & fs.ModePerm

	mode := info.Mode()
	if mode&wantType != wantType {
		return modeError{Path: name, Want: want, Got: mode, Err: fs.ErrInvalid}
	}
	if mode&wantPerm == 0 {
		return modeError{Path: name, Want: want, Got: mode, Err: fs.ErrPermission}
	}
	return nil
}
