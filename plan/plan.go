package plan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type FS interface {
	fs.ReadDirFS
	Lstat(name string) (fs.FileInfo, error)
	ReadLink(name string) (string, error)
}

type Advisor interface {
	fmt.Stringer
}

func Plan(target FS, source FS, advisor Advisor, packages []string) error {
	fmt.Println("planning", advisor, packages, "from", source, "to", target)
	return nil
}

type DirFS struct {
	fs.ReadDirFS
	Dir string
}

func NewDirFS(dir string) FS {
	dirFS := os.DirFS(dir).(fs.ReadDirFS)
	return &DirFS{
		ReadDirFS: dirFS,
		Dir:       dir,
	}
}

func (fsys *DirFS) Lstat(name string) (fs.FileInfo, error) {
	path := filepath.Join(fsys.Dir, name)
	return os.Lstat(path)
}

func (fsys *DirFS) ReadLink(name string) (string, error) {
	path := filepath.Join(fsys.Dir, name)
	return os.Readlink(path)
}

func (f *DirFS) String() string {
	return f.Dir
}
