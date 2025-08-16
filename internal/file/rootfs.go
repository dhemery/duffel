package file

import (
	"io/fs"
)

type Root interface {
	Name() string
	FS() fs.FS
	Mkdir(string, fs.FileMode) error
	Remove(string) error
	Symlink(string, string) error
}

type rootFS struct {
	fs.ReadLinkFS
	r Root
}

func RootFS(r Root) *rootFS {
	return &rootFS{
		ReadLinkFS: r.FS().(fs.ReadLinkFS),
		r:          r,
	}
}

func (f *rootFS) Name() string {
	return f.r.Name()
}

func (f *rootFS) Mkdir(name string, perm fs.FileMode) error {
	return f.r.Mkdir(name, perm)
}

func (f *rootFS) Remove(name string) error {
	return f.r.Remove(name)
}

func (f *rootFS) Symlink(oldname, newname string) error {
	return f.r.Symlink(oldname, newname)
}
