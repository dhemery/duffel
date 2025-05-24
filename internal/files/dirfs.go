package files

import (
	"io/fs"
	"os"
	"path/filepath"
)

type dirFS string

// DirFS returns a file system for the tree of files rooted at dir.
// It implements [duffel.FS].
func DirFS(dir string) dirFS {
	return dirFS(dir)
}

func (f dirFS) Join(path string) string {
	return filepath.Join(string(f), path)
}

func (f dirFS) Lstat(path string) (fs.FileInfo, error) {
	return os.Lstat(f.Join(path))
}

func (f dirFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(f.Join(path), perm)
}

func (f dirFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(f.Join(path))
}

func (f dirFS) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, f.Join(newname))
}
