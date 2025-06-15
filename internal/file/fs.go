package file

import "io/fs"

type symlinker interface {
	Symlink(oldname, newname string) error
}

func Symlink(fsys fs.FS, oldname, newname string) error {
	sym, ok := fsys.(symlinker)
	if !ok {
		return &LinkError{Op: "symlink", Old: oldname, New: newname, Err: fs.ErrInvalid}
	}
	return sym.Symlink(oldname, newname)
}

type dirmaker interface {
	Mkdir(path string, perm fs.FileMode) error
}

func MkDir(fsys fs.FS, dir string, perm fs.FileMode) error {
	md, ok := fsys.(dirmaker)
	if !ok {
		return &fs.PathError{Op: "mkdir", Path: dir, Err: fs.ErrInvalid}
	}
	return md.Mkdir(dir, perm)
}
