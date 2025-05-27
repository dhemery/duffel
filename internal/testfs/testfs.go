package testfs

import (
	"io/fs"
	"testing/fstest"
)

func New() FS {
	return FS{fstest.MapFS{}}
}

type FS struct {
	fstest.MapFS
}

func (fsys FS) Symlink(oldname, newname string) error {
	entry := LinkEntry(oldname)
	fsys.MapFS[newname] = entry
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
