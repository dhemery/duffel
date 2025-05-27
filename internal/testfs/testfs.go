package testfs

import (
	"fmt"
	"io/fs"
	"path"
	"testing/fstest"
)

func New() FS {
	return FS{fstest.MapFS{}}
}

type FS struct {
	fstest.MapFS
}

func (fsys FS) Symlink(oldname, newname string) error {
	if !fs.ValidPath(newname) {
		return &fs.PathError{Op: "symlink", Path: newname, Err: fs.ErrInvalid}
	}
	if _, ok := fsys.MapFS[newname]; ok {
		return &fs.PathError{Op: "symlink", Path: newname, Err: fs.ErrExist}
	}
	p, ok := fsys.MapFS[path.Dir(newname)]
	if !ok {
		err := fmt.Errorf("parent %w", fs.ErrNotExist)
		return &fs.PathError{Op: "symlink", Path: newname, Err: err}
	}
	if !p.Mode.IsDir() {
		err := fmt.Errorf("parent %w", fs.ErrInvalid)
		return &fs.PathError{Op: "symlink", Path: newname, Err: err}
	}
	// TODO: Check permission to create the link
	fsys.MapFS[newname] = LinkEntry(oldname)
	return nil
}

func DirEntry(perm fs.FileMode) *fstest.MapFile {
	return &fstest.MapFile{
		Mode: fs.ModeDir | perm,
	}
}

func FileEntry(data string, perm fs.FileMode) *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(data),
		Mode: perm,
	}
}

func LinkEntry(oldname string) *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(oldname),
		Mode: fs.ModeSymlink,
	}
}
