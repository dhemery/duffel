package file

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

const (
	TypeInvalid Type = -1 + iota // Unknown file type.
	TypeNoFile                   // The file does not exist.
	TypeFile                     // The file is a regular file.
	TypeDir                      // The file is a directory.
	TypeSymlink                  // The file is a symbolic link.
)

// Type is the type of an existing or planned file.
type Type int

// TypeOf returns the [Type] associated with file mode m.
func TypeOf(m fs.FileMode) (Type, error) {
	switch m.Type() {
	case 0:
		return TypeFile, nil
	case fs.ModeDir:
		return TypeDir, nil
	case fs.ModeSymlink:
		return TypeSymlink, nil
	default:
		return TypeInvalid, fmt.Errorf("unknown file mode %s", m)
	}
}

// IsDir reports whether t is the type of a directory.
func (t Type) IsDir() bool {
	return t == TypeDir
}

// IsRegular reports whether t is the type of a regular file.
func (t Type) IsRegular() bool {
	return t == TypeFile
}

// IsLink reports whether t is the type of a symbolic link.
func (t Type) IsLink() bool {
	return t == TypeSymlink
}

// IsNoFile reports whether t is the type of a non-existent file.
func (t Type) IsNoFile() bool {
	return t == TypeNoFile
}

// String formats t as a string.
func (t Type) String() string {
	switch t {
	case TypeNoFile:
		return "<no file>"
	case TypeFile:
		return "file"
	case TypeDir:
		return "directory"
	case TypeSymlink:
		return "symlink"
	case TypeInvalid:
		return "<invalid file type>"
	}
	return fmt.Sprintf("<unknown file type %o>", t)
}

// A State represents the state of an existing or planned file.
type State struct {
	Type     Type   `json:"type"`
	Dest     string `json:"dest"`
	DestType Type   `json:"desttype"`
}

// String formats s as a string.
func (s State) String() string {
	if s.Type == TypeSymlink {
		return fmt.Sprintf("%s to %s (%q)", s.Type, s.DestType, s.Dest)
	}
	return s.Type.String()
}

// NewStater creates a [Stater] that reads file states from fsys.
func NewStater(fsys fs.FS) Stater {
	return Stater{fsys}
}

// A Stater describes the states of files in a file system.
type Stater struct {
	FS fs.FS
}

// State returns the state of the named file.
func (s Stater) State(name string) (State, error) {
	t, err := s.statType(name)
	if err != nil {
		return State{}, err
	}
	state := State{Type: t}
	if t == TypeSymlink {
		dest, err := fs.ReadLink(s.FS, name)
		if err != nil {
			return State{}, err
		}
		fullDest := path.Join(path.Dir(name), dest)
		dt, err := s.statType(fullDest)
		if err != nil {
			return State{}, err
		}
		state.Dest = dest
		state.DestType = dt
	}
	return state, nil
}

// statType returns the [Type] of the file.
func (s Stater) statType(name string) (Type, error) {
	info, err := fs.Lstat(s.FS, name)
	if errors.Is(err, fs.ErrNotExist) {
		return TypeNoFile, nil
	}
	if err != nil {
		return TypeNoFile, err
	}
	return TypeOf(info.Mode())
}

var (
	dirState    = State{Type: TypeDir}
	fileState   = State{Type: TypeFile}
	noFileState = State{Type: TypeNoFile}
)

// DirState returns a [Stete] with type [TypeDir].
func DirState() State {
	return dirState
}

// FileState returns a [Stete] with type [TypeFile].
func FileState() State {
	return fileState
}

// LinkState returns a [State] with type [TypeLink]
// and the given destination and destination type.
func LinkState(dest string, destType Type) State {
	return State{Type: TypeSymlink, Dest: dest, DestType: destType}
}

// NoFileState returns a [Stete] with type [TypeNoFile].
func NoFileState() State {
	return noFileState
}
