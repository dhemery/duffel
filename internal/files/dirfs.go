package files

import (
	"fmt"
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

func (f dirFS) join(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", fmt.Errorf("path %s: %w", path, fs.ErrInvalid)
	}
	return filepath.Join(f.Dir, path), nil
}

func (f dirFS) Lstat(path string) (fs.FileInfo, error) {
	full, err := f.join(path)
	if err != nil {
		return nil, &fs.PathError{Op: "lstat", Path: path, Err: err}
	}
	return os.Lstat(full)
}

func (f dirFS) Mkdir(path string, perm fs.FileMode) error {
	full, err := f.join(path)
	if err != nil {
		return  &fs.PathError{Op: "mkdir", Path: path, Err: err}
	}
	return os.Mkdir(full, perm)
}

func (f dirFS) Symlink(oldname, newname string) error {
	full, err := f.join(newname)
	if err != nil {
		return  &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	return os.Symlink(oldname, full)
}
