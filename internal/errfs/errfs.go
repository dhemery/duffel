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
	return &FS{root: Dir("", 0755)}
}

// FS is a tree of [*file].
type FS struct {
	root *ErrFile
}

func Dir(name string, perm fs.FileMode, errs ...Error) *ErrFile {
	return newFile(name, fs.ModeDir|perm.Perm(), "", errs...)
}

func File(name string, perm fs.FileMode, errs ...Error) *ErrFile {
	return newFile(name, perm.Perm(), "", errs...)
}

func Link(name string, dest string, errs ...Error) *ErrFile {
	return newFile(name, fs.ModeSymlink, dest, errs...)
}

func (fsys *FS) Add(f *ErrFile) error {
	const op = fsOp + "add"

	if f.name == "." {
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrInvalid}
	}

	dir := path.Dir(f.name)
	parent, err := fsys.Find(dir)
	if errors.Is(err, fs.ErrNotExist) {
		parent = Dir(dir, 0o755)
		err = fsys.Add(parent)
	}
	if err != nil {
		return &fs.PathError{Op: op, Path: f.name, Err: err}
	}

	if _, ok := parent.entries[f.name]; ok {
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrExist}
	}

	parent.entries[f.Name()] = f

	return nil
}

func (fsys *FS) Find(name string) (*ErrFile, error) {
	if name == "." {
		return fsys.root, nil
	}

	parent, err := fsys.Find(path.Dir(name))
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

	f, err := fsys.Find(name)
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
	file, err := fsys.Find(name)
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
	file, err := fsys.Find(name)
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
	file, err := fsys.Find(name)
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
	if err := fsys.Add(Link(newname, oldname)); err != nil {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: err}
	}
	return nil
}

// Stat returns a [fs.FileInfo] that describes the named file.
// This implementation of Stat does not follow symlinks.
// If the file was created with ErrStat,
// Lstat returns that error instead of the file info.
func (fsys *FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + "stat"
	file, err := fsys.Find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrStat}
	}

	return file, nil
}

type ErrFile struct {
	name    string              // The full name of the file
	mode    fs.FileMode         // The file mode
	dest    string              // The link destination if the file is a symlink
	entries map[string]*ErrFile // The dir entries if the file is dir
	errors  map[Error]bool      // Errors to return from corresponding methods
}

func newFile(name string, mode fs.FileMode, dest string, errs ...Error) *ErrFile {
	f := &ErrFile{
		name:    name,
		mode:    mode,
		dest:    dest,
		entries: map[string]*ErrFile{},
		errors:  map[Error]bool{},
	}
	for _, e := range errs {
		f.errors[e] = true
	}
	return f
}

func (f *ErrFile) FullName() string {
	return f.name
}

// Close implements fs.File.
// It does nothing.
func (f *ErrFile) Close() error {
	return nil
}

// Info implements fs.DirEntry.
func (f *ErrFile) Info() (fs.FileInfo, error) {
	return f, nil
}

// IsDir implements fs.DirEntry and fs.FileInfo.
func (f *ErrFile) IsDir() bool {
	return f.mode&fs.ModeDir != 0
}

// ModTime implements fs.FileInfo.
// It always returns the zero time.
func (f *ErrFile) ModTime() time.Time {
	return time.Time{}
}

// Mode implements fs.FileInfo.
func (f *ErrFile) Mode() fs.FileMode {
	return f.mode
}

// Name implements fs.DirEntry and fs.FileInfo.
func (f *ErrFile) Name() string {
	return path.Base(f.name)
}

// Read implements fs.File.
// It always returns 0, [errors.ErrUnsupported].
func (f *ErrFile) Read([]byte) (int, error) {
	const op = fileOp + "read"
	return 0, &fs.PathError{Op: op, Path: f.name, Err: errors.ErrUnsupported}
}

// Size implements fs.FileInfo.
// It always returns 0.
func (f *ErrFile) Size() int64 {
	return 0
}

// Stat implements fs.File.
// It does not follow links.
func (f *ErrFile) Stat() (fs.FileInfo, error) {
	const op = fileOp + "stat"
	if f.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: f.name, Err: ErrStat}
	}
	return f, nil
}

// Sys implements fs.FileInfo.
// It returns the full name of the file.
func (f *ErrFile) Sys() any {
	return f.name
}

// Type implements fs.DirEntry.
func (f *ErrFile) Type() fs.FileMode {
	return f.mode.Type()
}

func (fsys *FS) String() string {
	var out strings.Builder
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		f, _ := fsys.Find(name)
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

func (f *ErrFile) Equal(o *ErrFile) bool {
	fNil := f == nil
	oNil := o == nil
	if fNil != oNil {
		return false
	}
	if fNil {
		return true
	}
	return f.name == o.name &&
		f.mode == o.mode &&
		f.dest == o.dest
}

func (f *ErrFile) String() string {
	var out strings.Builder
	out.WriteString(f.mode.String())
	if f.IsDir() {
		var children []string
		for _, e := range f.entries {
			children = append(children, path.Base(e.name))
		}
		out.WriteString(" [")
		out.WriteString(strings.Join(children, ", "))
		out.WriteRune(']')
	}
	if f.mode&fs.ModeSymlink != 0 {
		out.WriteString(` "`)
		out.WriteString(f.dest)
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
