// Package errfs provides a file system that can be customized
// to return specified errors.
package errfs

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"time"
)

const (
	fsOp   = "errfs."
	fileOp = "errfs.file."
)

// Errors to return from corresponding FS and file methods.
var (
	ErrClose    = Error{"ErrClose"}
	ErrLstat    = Error{"ErrLstat"}
	ErrOpen     = Error{"ErrOpen"}
	ErrRead     = Error{"ErrRead"}
	ErrReadDir  = Error{"ErrReadDir"}
	ErrReadLink = Error{"ErrReadLink"}
	ErrStat     = Error{"ErrStat"}
)

type Error struct {
	s string
}

func (e Error) Error() string {
	return fsOp + e.s
}

// New returns a new FS.
func New() *FS {
	return &FS{root: newFile("", fs.ModeDir|0o775, "")}
}

// FS is a tree of [*file].
type FS struct {
	root *file
}

// AddFile adds a regular file to fsys
// with the given name, permissions, and prepared errors.
// Any missing ancestor directories are also added.
func (fsys *FS) AddFile(name string, perm fs.FileMode, errs ...Error) error {
	return fsys.Add(name, perm.Perm(), "", errs...)
}

// AddDir adds a directory to fsys
// with the given name, permissions, and prepared errors.
// Any missing ancestor directories are also added.
func (fsys *FS) AddDir(name string, perm fs.FileMode, errs ...Error) error {
	return fsys.Add(name, fs.ModeDir|perm.Perm(), "", errs...)
}

// AddLink adds a symlink to fsys
// with the given name, destination, and prepared errors.
// Any missing ancestor directories are also added.
func (fsys *FS) AddLink(name string, dest string, errs ...Error) error {
	return fsys.Add(name, fs.ModeSymlink, dest, errs...)
}

// Add adds a new [*file] to fsys
// with the given name, mode, destination, and prepared errors.
// Any missing ancestor directories are also added.
func (fsys *FS) Add(name string, mode fs.FileMode, dest string, errs ...Error) error {
	_, err := fsys.addFile(name, mode, dest, errs...)
	return err
}

func (fsys *FS) addFile(name string, mode fs.FileMode, dest string, errs ...Error) (*file, error) {
	const op = fsOp + "add"

	if name == "." {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	dir := path.Dir(name)
	parent, err := fsys.find(dir)
	if errors.Is(err, fs.ErrNotExist) {
		parent, err = fsys.addFile(dir, fs.ModeDir|0o755, "")
	}
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if _, ok := parent.entries[name]; ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}

	f := newFile(name, mode, dest, errs...)

	parent.entries[f.Name()] = f

	return f, nil
}

func (fsys *FS) find(name string) (*file, error) {
	if name == "." {
		return fsys.root, nil
	}

	parent, err := fsys.find(path.Dir(name))
	if err != nil {
		return nil, err
	}

	if !parent.IsDir() {
		return nil, fs.ErrInvalid
	}

	for _, e := range parent.entries {
		if e.name == name {
			return e, nil
		}
	}
	return nil, fs.ErrNotExist
}

// Open returnes the named file.
// If the file was created with ErrOpen,
// that error is returned instead.
func (fsys *FS) Open(name string) (fs.File, error) {
	const op = fsOp + "open"

	f, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	if f.errors[ErrOpen] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrOpen}
	}
	return f, nil
}

// Lstat returns a [fs.FileInfo] that describes the named file.
// Lstat does not follow symlinks.
// If the file was created with ErrLstat,
// that error is returned instead.
func (fsys *FS) Lstat(name string) (fs.FileInfo, error) {
	const op = fsOp + "lstat"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.errors[ErrLstat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrLstat}
	}

	return file, nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
// If the directory was created with ErrReadDir,
// that error is returned instead.
func (fsys *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	const op = fsOp + "readdir"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if !file.IsDir() {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.errors[ErrReadDir] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrReadDir}
	}

	var entries []fs.DirEntry
	for _, key := range slices.Sorted(maps.Keys(file.entries)) {
		entries = append(entries, file.entries[key])
	}

	return entries, nil
}

// ReadLink returns the Dest of the named symlink file.
// If the file was created with ErrReadLink,
// that error is returned instead.
func (fsys *FS) ReadLink(name string) (string, error) {
	const op = fsOp + "readlink"
	file, err := fsys.find(name)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.Mode()&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.errors[ErrReadLink] {
		return "", &fs.PathError{Op: op, Path: name, Err: ErrReadLink}
	}

	return file.dest, nil
}

func (fsys *FS) Symlink(oldname, newname string) error {
	const op = fsOp + "symlink"
	return fsys.Add(newname, fs.ModeSymlink, oldname)
}

// Stat returns a [fs.FileInfo] that describes the named file.
// This implementation of Stat does not follow symlinks.
// If the file was created with ErrStat,
// Lstat returns that error instead of the file info.
func (fsys *FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + "stat"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrStat}
	}

	return file, nil
}

type file struct {
	name    string           // The full name of the file
	mode    fs.FileMode      // The file mode
	dest    string           // The link destination if the file is a symlink
	entries map[string]*file // The dir entries if the file is dir
	errors  map[Error]bool   // Errors to return from corresponding methods
}

func newFile(name string, mode fs.FileMode, dest string, errs ...Error) *file {
	f := &file{
		name:    name,
		mode:    mode,
		dest:    dest,
		entries: map[string]*file{},
		errors:  map[Error]bool{},
	}
	for _, e := range errs {
		f.errors[e] = true
	}
	return f
}

// Close implements fs.File.
// It does nothing.
func (f *file) Close() error {
	return nil
}

// Info implements fs.DirEntry.
func (f *file) Info() (fs.FileInfo, error) {
	return f, nil
}

// IsDir implements fs.DirEntry and fs.FileInfo.
func (f *file) IsDir() bool {
	return f.mode&fs.ModeDir != 0
}

// ModTime implements fs.FileInfo.
// It always returns the zero time.
func (f *file) ModTime() time.Time {
	return time.Time{}
}

// Mode implements fs.FileInfo.
func (f *file) Mode() fs.FileMode {
	return f.mode
}

// Name implements fs.DirEntry and fs.FileInfo.
func (f *file) Name() string {
	return path.Base(f.name)
}

// Read implements fs.File.
// It always returns 0, [errors.ErrUnsupported].
func (f *file) Read([]byte) (int, error) {
	const op = fileOp + "read"
	return 0, &fs.PathError{Op: op, Path: f.name, Err: errors.ErrUnsupported}
}

// Size implements fs.FileInfo.
// It always returns 0.
func (f *file) Size() int64 {
	return 0
}

// Stat implements fs.File.
// It does not follow links.
func (f *file) Stat() (fs.FileInfo, error) {
	const op = fileOp + "stat"
	if f.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: f.name, Err: ErrStat}
	}
	return f, nil
}

func (f *file) String() string {
	return fmt.Sprintf("%q %s %q %v %v",
		f.name, f.mode, f.dest, f.entries, f.errors)
}

// Sys implements fs.FileInfo.
// It returns the full name of the file.
func (f *file) Sys() any {
	return f.name
}

// Type implements fs.DirEntry.
func (f *file) Type() fs.FileMode {
	return f.mode.Type()
}
