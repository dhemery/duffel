package testfs

import (
	"io/fs"
	"path"
	"testing/fstest"
)

const (
	readAccess   = 0o444
	searchAccess = 0o111
	writeAccess  = 0o222
)

func New() FS {
	return FS{fstest.MapFS{}}
}

type FS struct {
	M fstest.MapFS
}

func (f FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	if err := f.checkDirAccess("readdir", name, readAccess); err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}
	return f.M.ReadDir(name)
}

func (f FS) Symlink(oldname, newname string) error {
	if !fs.ValidPath(newname) {
		return &fs.PathError{Op: "symlink", Path: newname, Err: fs.ErrInvalid}
	}
	if err := f.checkDirAccess("symlink", path.Dir(newname), writeAccess); err != nil {
		return &fs.PathError{Op: "symlink", Path: newname, Err: err}
	}
	f.M[newname] = LinkEntry(oldname)
	return nil
}

func DirEntry(perm fs.FileMode) *fstest.MapFile {
	return &fstest.MapFile{
		Mode: fs.ModeDir | perm.Perm(),
	}
}

func FileEntry(data string, perm fs.FileMode) *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(data),
		Mode: perm.Perm(),
	}
}

func LinkEntry(oldname string) *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(oldname),
		Mode: fs.ModeSymlink,
	}
}

func (f FS) checkDirAccess(op, dir string, want fs.FileMode) error {
	if err := f.checkSearchDir(op, path.Dir(dir)); err != nil {
		return &fs.PathError{Op: op, Path: dir, Err: err}
	}
	info, err := f.M.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &fs.PathError{Op: op, Path: dir, Err: fs.ErrInvalid}
	}
	got := info.Mode().Perm()
	if got&want == 0 {
		return &fs.PathError{Op: op, Path: dir, Err: fs.ErrPermission}
	}
	return nil
}

func (f FS) checkSearchDir(op, lookupDir string) error {
	if lookupDir == "." {
		return nil
	}
	if err := f.checkSearchDir(op, path.Dir(lookupDir)); err != nil {
		return err
	}
	info, err := f.M.Stat(lookupDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &fs.PathError{Op: op, Path: lookupDir, Err: fs.ErrInvalid}
	}
	perm := info.Mode().Perm()
	if perm&0o111 == 0 {
		return &fs.PathError{Op: op, Path: lookupDir, Err: fs.ErrPermission}
	}
	return nil
}
