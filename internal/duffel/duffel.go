package duffel

import (
	"io"
	"io/fs"
)

type FS interface {
	fs.ReadDirFS
	Symlink(oldname, newname string) error
}

type Request struct {
	Stdout io.Writer
	FS     FS
	Source string
	Target string
	Pkgs   []string
	DryRun bool
}
