package file

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type osfs interface {
	fs.ReadDirFS
	fs.ReadLinkFS
}

type dirFS struct {
	osfs
	Dir string
}

type LinkError = os.LinkError

// DirFS returns a file system for the tree of files rooted at dir.
// It implements [fs.ReadDirFS] and [fs.ReadLinkFS].
func DirFS(dir string) dirFS {
	return dirFS{
		osfs: os.DirFS(dir).(osfs),
		Dir:  dir,
	}
}

func (f dirFS) join(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", fmt.Errorf("path %s: %w", path, fs.ErrInvalid)
	}
	return filepath.Join(f.Dir, path), nil
}

func (f dirFS) Sub(dir string) (fs.FS, error) {
	full, err := f.join(dir)
	if err != nil {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: err}
	}
	return DirFS(full), nil
}

func (f dirFS) Mkdir(path string, perm fs.FileMode) error {
	full, err := f.join(path)
	if err != nil {
		return &fs.PathError{Op: "mkdir", Path: path, Err: err}
	}
	return os.Mkdir(full, perm)
}

func (f dirFS) Symlink(oldname, newname string) error {
	full, err := f.join(newname)
	if err != nil {
		return &LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	return os.Symlink(oldname, full)
}
