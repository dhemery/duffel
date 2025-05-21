package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

type dirFS string

func (f dirFS) Lstat(path string) (fs.FileInfo, error) {
	return os.Lstat(filepath.Join(string(f), path))
}

func (f dirFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(filepath.Join(string(f), path), perm)
}

func (f dirFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(filepath.Join(string(f), path))
}

func (f dirFS) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, filepath.Join(string(f), newname))
}
