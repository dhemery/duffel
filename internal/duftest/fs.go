package duftest

import (
	"fmt"
	"io/fs"
	"path"
	"testing/fstest"
)

const (
	readAccess   = 0o444
	searchAccess = 0o111
	writeAccess  = 0o222
	// fsOp is the prefix for op names in PathError errors returned by FS methods
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
	if err := f.checkFileAccess(name, readAccess); err != nil {
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
	if err := f.checkDirAccess(name, readAccess); err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	entries, err := f.M.ReadDir(name)
	if err != nil {
		err = &fs.PathError{Op: op, Path: name, Err: err}
	}
	return entries, err
}

func (f FS) Symlink(oldname, newname string) error {
	const op = fsOp + "symlink"
	if !fs.ValidPath(newname) {
		return &fs.PathError{Op: op, Path: newname, Err: fs.ErrInvalid}
	}
	if err := f.checkDirAccess(path.Dir(newname), writeAccess); err != nil {
		return &fs.PathError{Op: op, Path: newname, Err: err}
	}
	f.M[newname] = &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte(oldname)}
	return nil
}

func (f FS) checkDirAccess(dir string, want fs.FileMode) error {
	const op = fsOp + "checkDirAccess"
	if err := f.checkSearchDir(path.Dir(dir)); err != nil {
		return fmt.Errorf("%s %s: %w", op, dir, err)
	}
	info, err := f.M.Stat(dir)
	if err != nil {
		return fmt.Errorf("%s %s: %w", op, dir, err)
	}
	mode := info.Mode()
	if !mode.IsDir() {
		return fmt.Errorf("%s %s: want dir, got %s: %w", op, dir, mode, fs.ErrInvalid)
	}
	got := mode.Perm()
	if got&want == 0 {
		return fmt.Errorf("%s %s: want perm %s, got mode %s: %w", op, dir, want, mode, fs.ErrPermission)
	}
	return nil
}

func (f FS) checkFileAccess(file string, want fs.FileMode) error {
	const op = fsOp + "checkFileAccess"
	if err := f.checkSearchDir(path.Dir(file)); err != nil {
		return fmt.Errorf("%s %s: %w", op, file, err)
	}
	info, err := f.M.Stat(file)
	if err != nil {
		return fmt.Errorf("%s %s: %w", op, file, err)
	}
	mode := info.Mode()
	got := mode.Perm()
	if got&want == 0 {
		return fmt.Errorf("%s %s: want perm %s, got mode %s: %w", op, file, want, mode, fs.ErrPermission)
	}
	return nil
}

func (f FS) checkSearchDir(lookupDir string) error {
	const op = fsOp + "checkSearchDir"
	if lookupDir == "." {
		return nil
	}
	if err := f.checkSearchDir(path.Dir(lookupDir)); err != nil {
		return fmt.Errorf("%s %s: %w", op, lookupDir, err)
	}
	info, err := f.M.Stat(lookupDir)
	if err != nil {
		return fmt.Errorf("%s %s: %w", op, lookupDir, err)
	}
	mode := info.Mode()
	if !mode.IsDir() {
		return fmt.Errorf("%s %s: want dir, got %s: %w", op, lookupDir, mode, fs.ErrInvalid)
	}
	perm := mode.Perm()
	if perm&0o111 == 0 {
		return fmt.Errorf("%s %s: want searchable, got mode %s: %w", op, lookupDir, mode, fs.ErrPermission)
	}
	return nil
}
