// Helper functions that are not part of the file and file system APIs.

package errfs

import "io/fs"

// Add adds file to fsys,
// along with any missing ancestor directories.
// If the file was created with errs,
// each relevant file operation and file system operation
// will return the corresponding error.
func Add(fsys *FS, file *File) error {
	return fsys.add(file)
}

// AddDir creates the specified directory and adds it to fsys,
// along witn any missing ancestor directories.
// If errs is non-empty,
// each relevant file operation and file system operation
// will return the corresponding error.
func AddDir(fsys *FS, name string, perm fs.FileMode, errs ...Error) error {
	return fsys.add(NewDir(name, perm, errs...))
}

// AddFile creates the specified file and adds it to fsys,
// along witn any missing ancestor directories.
// If errs is non-empty,
// each relevant file operation and file system operation
// will return the corresponding error.
func AddFile(fsys *FS, name string, perm fs.FileMode, errs ...Error) error {
	return fsys.add(NewFile(name, perm, errs...))
}

// AddLink creates the specified symlink and adds it to fsys,
// along witn any missing ancestor directories.
// If errs is non-empty,
// each relevant file operation and file system operation
// will return the corresponding error.
func AddLink(fsys *FS, name string, dest string, errs ...Error) error {
	return fsys.add(NewLink(name, dest, errs...))
}

// Find returns the named file.
func Find(fsys *FS, name string) (*File, error) {
	return fsys.find(name)
}

// NewDir creates a new directory file.
// If errs is non-empty,
// each relevant file operation
// will return the corresponding error.
func NewDir(name string, perm fs.FileMode, errs ...Error) *File {
	return newFile(name, fs.ModeDir|perm.Perm(), "", errs...)
}

// NewFile creates a new regular file.
// If errs is non-empty,
// each relevant file operation
// will return the corresponding error.
func NewFile(name string, perm fs.FileMode, errs ...Error) *File {
	return newFile(name, perm.Perm(), "", errs...)
}

// NewLink creates a new symlink file.
// If errs is non-empty,
// each relevant file operation
// will return the corresponding error.
func NewLink(name string, dest string, errs ...Error) *File {
	return newFile(name, fs.ModeSymlink, dest, errs...)
}

// FileToInfo returns an [Info] that describes f.
// The result implements [fs.FileInfo]
func FileToInfo(f *File) Info {
	return f.info()
}

// FileToEntry returns an [Entry] that describes f.
// The result implements [fs.DirEntry]
func FileToEntry(f *File) Entry {
	return f.entry()
}
