// Package errfs provides a file system that can be customized
// to return specified errors.
package errfs

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
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
	return &FS{root: NewDir("", 0o755)}
}

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

// FS is a tree of [*File].
type FS struct {
	root *File
}

func (fsys *FS) add(f *File) error {
	const op = fsOp + "add"

	if f.Name == "." {
		return &fs.PathError{Op: op, Path: f.Name, Err: fs.ErrInvalid}
	}

	dir := path.Dir(f.Name)
	parent, err := fsys.find(dir)
	if errors.Is(err, fs.ErrNotExist) {
		parent = NewDir(dir, 0o755)
		err = fsys.add(parent)
	}
	if err != nil {
		return &fs.PathError{Op: op, Path: f.Name, Err: err}
	}

	if _, ok := parent.entries[f.Name]; ok {
		return &fs.PathError{Op: op, Path: f.Name, Err: fs.ErrExist}
	}

	e := f.entry()
	parent.entries[e.Name()] = e

	return nil
}

func (fsys *FS) find(name string) (*File, error) {
	if name == "." {
		return fsys.root, nil
	}

	parent, err := fsys.find(path.Dir(name))
	if err != nil {
		return nil, err
	}

	if !parent.Mode.IsDir() {
		return nil, fs.ErrInvalid
	}

	base := path.Base(name)
	for _, e := range parent.entries {
		if e.Name() == base {
			return e.file, nil
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

	return file.info(), nil
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

	if !file.Mode.IsDir() {
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

	if file.Mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.errors[ErrReadLink] {
		return "", &fs.PathError{Op: op, Path: name, Err: ErrReadLink}
	}

	return file.Dest, nil
}

// Symlink creates a new symlink with the given name and destination.
// TODO: Do not use fsys.add. Instead find the parent, and fail if error.
// TODO: Return an error if the parent was created with ErrCreate.
func (fsys *FS) Symlink(dest, name string) error {
	const op = fsOp + "symlink"
	if err := fsys.add(NewLink(name, dest)); err != nil {
		return &os.LinkError{Op: op, Old: dest, New: name, Err: err}
	}
	return nil
}

// Stat returns a [fs.FileInfo] that describes the named file.
// This implementation of Stat does not follow symlinks.
// If the file was created with ErrStat,
// that error is returned instead.
func (fsys *FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + "stat"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrStat}
	}

	return Info{file}, nil
}

func (fsys *FS) String() string {
	var out strings.Builder
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		f, _ := fsys.find(name)
		out.WriteRune('"')
		out.WriteString(name)
		out.WriteString(`" `)
		out.WriteString(f.String())
		out.WriteRune('\n')
		return nil
	})
	if err != nil {
		return err.Error()
	}
	return out.String()
}

type File struct {
	Name    string           // The full name of the file
	Mode    fs.FileMode      // The file mode
	Dest    string           // The link destination if the file is a symlink
	entries map[string]Entry // The dir entries if the file is dir
	errors  map[Error]bool   // Errors to return from corresponding methods
}

func (f *File) entry() Entry {
	return Entry{f}
}

func (f *File) info() Info {
	return Info{f}
}

func newFile(name string, mode fs.FileMode, dest string, errs ...Error) *File {
	f := &File{
		Name:    name,
		Mode:    mode,
		Dest:    dest,
		entries: map[string]Entry{},
		errors:  map[Error]bool{},
	}
	for _, e := range errs {
		f.errors[e] = true
	}
	return f
}

// Close implements fs.File.
// It does nothing.
func (f *File) Close() error {
	return nil
}

// Read implements fs.File.
// It always returns 0, [errors.ErrUnsupported].
func (f *File) Read([]byte) (int, error) {
	const op = fileOp + "read"
	return 0, &fs.PathError{Op: op, Path: f.Name, Err: errors.ErrUnsupported}
}

// Stat implements fs.File.
// It does not follow links.
func (f *File) Stat() (fs.FileInfo, error) {
	const op = fileOp + "stat"
	if f.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: f.Name, Err: ErrStat}
	}
	return Info{f}, nil
}

func (f *File) String() string {
	var out strings.Builder
	out.WriteString(f.Mode.String())
	if f.Mode.IsDir() {
		var children []string
		for _, e := range f.entries {
			children = append(children, e.Name())
		}
		out.WriteString(" [")
		out.WriteString(strings.Join(children, ", "))
		out.WriteRune(']')
	}
	if f.Mode&fs.ModeSymlink != 0 {
		out.WriteString(` "`)
		out.WriteString(f.Dest)
		out.WriteRune('"')
	}
	var errors []string
	for e, enabled := range f.errors {
		if enabled {
			errors = append(errors, e.Error())
		}
	}
	if len(errors) > 1 {
		out.WriteString(" [")
		out.WriteString(strings.Join(errors, ", "))
		out.WriteRune(']')
	}
	return out.String()
}

type Info struct {
	file *File
}

// IsDir implements fs.DirEntry and fs.FileInfo.
func (i Info) IsDir() bool {
	return i.Mode().IsDir()
}

// ModTime implements fs.FileInfo.
// It always returns the zero time.
func (i Info) ModTime() time.Time {
	return time.Time{}
}

// Mode implements fs.FileInfo.
func (i Info) Mode() fs.FileMode {
	return i.file.Mode
}

// Name implements fs.FileInfo.
func (i Info) Name() string {
	return path.Base(i.file.Name)
}

// Size implements fs.FileInfo.
// It always returns 0.
func (i Info) Size() int64 {
	return 0
}

// Sys implements fs.FileInfo.
// It returns the full name of the file.
func (i Info) Sys() any {
	return i.Name
}

type Entry struct {
	file *File
}

// Info implements fs.DirEntry.
func (e Entry) Info() (fs.FileInfo, error) {
	return e.file.info(), nil
}

// IsDir implements fs.DirEntry.
func (e Entry) IsDir() bool {
	return e.Type().IsDir()
}

// Name implements fs.DirEntry.
func (e Entry) Name() string {
	return path.Base(e.file.Name)
}

// Type implements fs.DirEntry.
func (e Entry) Type() fs.FileMode {
	return e.file.Mode.Type()
}
