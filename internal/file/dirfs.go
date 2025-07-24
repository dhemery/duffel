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

func (f dirFS) join(name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", fmt.Errorf("path %s: %w", name, fs.ErrInvalid)
	}
	return filepath.Join(f.Dir, name), nil
}

func (f dirFS) Mkdir(name string, perm fs.FileMode) error {
	full, err := f.join(name)
	if err != nil {
		return &fs.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return os.Mkdir(full, perm)
}

func (f dirFS) Remove(name string) error {
	full, err := f.join(name)
	if err != nil {
		return &fs.PathError{Op: "remove", Path: name, Err: err}
	}
	return os.Remove(full)
}

func (f dirFS) Symlink(oldname, newname string) error {
	full, err := f.join(newname)
	if err != nil {
		return &LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	return os.Symlink(oldname, full)
}
