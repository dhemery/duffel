package files

import (
	"io/fs"
)

// ReadLinkFS anticipates the fs.ReadLinkFS to be released in Go 1.25.
type ReadLinkFS interface {
	fs.FS
	Lstat(name string) (fs.FileInfo, error)
	ReadLink(name string) (string, error)
}

func ReadLink(fsys fs.FS, name string) (string, error) {
	sym, ok := fsys.(ReadLinkFS)
	if !ok {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: fs.ErrInvalid}
	}
	return sym.ReadLink(name)
}

func Lstat(fsys fs.FS, name string) (fs.FileInfo, error) {
	sym, ok := fsys.(ReadLinkFS)
	if !ok {
		return fs.Stat(fsys, name)
	}
	return sym.Lstat(name)
}
