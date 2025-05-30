package duffel

import (
	"io"
	"io/fs"
)

type FS interface {
	fs.ReadDirFS
	// Lstat(path string) (fs.FileInfo, error)
	// Mkdir(path string, perm fs.FileMode) error
	Symlink(old, new string) error
}

type Request struct {
	Stdout io.Writer
	FS     FS
	Source string
	Target string
	Pkgs   []string
	DryRun bool
}
