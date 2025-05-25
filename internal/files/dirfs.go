package files

import (
	"io/fs"
	"os"
	"path/filepath"
)

type dirFS struct {
	fs.ReadDirFS
	Dir string
}

// DirFS returns a file system for the tree of files rooted at dir.
// It implements [duffel.FS].
func DirFS(dir string) dirFS {
	return dirFS{
		ReadDirFS: os.DirFS(dir).(fs.ReadDirFS),
		Dir:       dir,
	}
}

func (f dirFS) Join(path string) string {
	return filepath.Join(f.Dir, path)
}

func (f dirFS) Lstat(path string) (fs.FileInfo, error) {
	return os.Lstat(f.Join(path))
}

func (f dirFS) Mkdir(path string, perm fs.FileMode) error {
	return os.Mkdir(f.Join(path), perm)
}

func (f dirFS) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, f.Join(newname))
}
