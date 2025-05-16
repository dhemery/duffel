package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
)

type dirFS struct {
	fsys fs.FS
	dir  string
}

// DirFS returns a file system (an fs.FS) for the tree rooted at the directory dir.
// See [os.DirFS] for details.
//
// The result implements [io/fs.StatFS], [io/fs.ReadFileFS],
// [io/fs/ReadDirFS], and the upcoming (go 1.25) [io/fs.ReadLinkFS].
//
// When go 1.25 is released, remove this and use [os.DirFS].
func DirFS(dir string) fs.FS {
	fsys := os.DirFS(dir)
	return &dirFS{
		fsys: fsys,
		dir:  dir,
	}
}

func (f *dirFS) Open(name string) (fs.File, error) {
	return f.fsys.Open(name)
}

func (f *dirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return f.fsys.(fs.ReadDirFS).ReadDir(name)
}

func (f *dirFS) ReadFile(name string) ([]byte, error) {
	return f.fsys.(fs.ReadFileFS).ReadFile(name)
}

func (f *dirFS) Stat(name string) (fs.FileInfo, error) {
	return f.fsys.(fs.StatFS).Stat(name)
}

func (f *dirFS) Lstat(name string) (fs.FileInfo, error) {
	path := filepath.Join(f.dir, name)
	return os.Lstat(path)
}

func (f *dirFS) ReadLink(name string) (string, error) {
	path := filepath.Join(f.dir, name)
	return os.Readlink(path)
}

func (f *dirFS) String() string {
	return f.dir
}
