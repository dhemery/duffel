// Helper functions that are not part of the file and file system APIs.

package errfs

import (
	"io/fs"
)

// NewDir creates a new directory file.
// Each [Error] configures the associated operation
// on the directory to return that error.
func NewDir(name string, perm fs.FileMode, errs ...Error) *File {
	return newFile(name, fs.ModeDir|perm.Perm(), "", errs)
}

// NewFile creates a new regular file.
// Each [Error] configures the associated operation
// on the file to return that error.
func NewFile(name string, perm fs.FileMode, errs ...Error) *File {
	return newFile(name, perm.Perm(), "", errs)
}

// NewLink creates a new symlink file.
// Each [Error] configures the associated operation
// on the symlink to return that error.
func NewLink(name string, dest string, errs ...Error) *File {
	return newFile(name, fs.ModeSymlink, dest, errs)
}

// FileName returns the name used to create f.
func FileName(f *File) string {
	return f.name
}

// FileDirEntry returns a [fs.DirEntry] that describes f.
func FileDirEntry(f *File) fs.DirEntry {
	return f.entry()
}

// Add adds f to fsys,
// along with any missing ancestor directories.
func Add(fsys *FS, f *File) error {
	_, err := fsys.add(f)
	return err
}

// AddDir creates the specified directory file and adds it to fsys,
// along witn any missing ancestor directories.
// Each [Error] configures the associated operation
// on the directory to return that error.
func AddDir(fsys *FS, name string, perm fs.FileMode, errs ...Error) error {
	return Add(fsys, NewDir(name, perm, errs...))
}

// AddFile creates the specified file and adds it to fsys,
// along witn any missing ancestor directories.
// Each [Error] configures the associated operation
// on the file to return that error.
func AddFile(fsys *FS, name string, perm fs.FileMode, errs ...Error) error {
	return Add(fsys, NewFile(name, perm, errs...))
}

// AddLink creates the specified symlink file and adds it to fsys,
// along witn any missing ancestor directories.
// Each [Error] configures the associated operation
// on the symlink to return that error.
func AddLink(fsys *FS, name string, dest string, errs ...Error) error {
	return Add(fsys, NewLink(name, dest, errs...))
}

// Find returns the named file.
func Find(fsys *FS, name string) (*File, error) {
	node, err := fsys.find(name)
	if err != nil {
		return nil, err
	}
	return node.file, nil
}
