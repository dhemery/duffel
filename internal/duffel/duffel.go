package duffel

import (
	"io/fs"
)

type FS interface {
	fs.ReadDirFS
	Symlink(oldname, newname string) error
}

type Request struct {
	FS     FS
	Source string
	Target string
	Pkgs   []string
}
