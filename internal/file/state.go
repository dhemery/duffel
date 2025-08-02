package file

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
)

type Type int

const (
	TypeNoFile Type = iota
	TypeFile
	TypeDir
	TypeSymlink
)

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
	}
	return fmt.Sprintf("<unknown file type %o>", t)
}

// A State represents the state of an existing or planned file.
type State struct {
	Type     Type   `json:"type"`
	Dest     string `json:"dest"`
	DestType Type   `json:"desttype"`
}

// Equal reports whether o represents the same state as s.
func (s *State) Equal(o *State) bool {
	sNil := s == nil
	oNil := o == nil
	if sNil != oNil {
		return false
	}
	if sNil {
		return true
	}
	return s.Type == o.Type &&
		s.Dest == o.Dest &&
		s.DestType == o.DestType
}

// LogValue implements [slog.LogValuer].
func (s *State) LogValue() slog.Value {
	if s == nil {
		return slog.AnyValue(nil)
	}
	return slog.GroupValue(
		slog.String("type", s.Type.String()),
		slog.String("dest", s.Dest),
		slog.String("desttype", s.DestType.String()),
	)
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
func (s Stater) State(name string) (*State, error) {
	t, err := s.StatType(name)
	if err != nil {
		return nil, err
	}
	state := &State{Type: t}
	if t == TypeSymlink {
		dest, err := fs.ReadLink(s.FS, name)
		if err != nil {
			return nil, err
		}
		fullDest := path.Join(path.Dir(name), dest)
		dt, err := s.StatType(fullDest)
		if err != nil {
			return nil, err
		}
		state.Dest = dest
		state.DestType = dt
	}
	return state, nil
}

func (s Stater) StatType(name string) (Type, error) {
	info, err := fs.Lstat(s.FS, name)
	if errors.Is(err, fs.ErrNotExist) {
		return TypeNoFile, nil
	}
	if err != nil {
		return TypeNoFile, err
	}
	modetype := info.Mode().Type()
	switch modetype {
	case 0:
		return TypeFile, nil
	case fs.ModeDir:
		return TypeDir, nil
	case fs.ModeSymlink:
		return TypeSymlink, nil
	default:
		return TypeNoFile, fs.ErrInvalid
	}
}

func TypeOf(m fs.FileMode) Type {
	switch m.Type() {
	case 0:
		return TypeFile
	case fs.ModeDir:
		return TypeDir
	case fs.ModeSymlink:
		return TypeSymlink
	default:
		return Type(m)
	}
}

func NoFile() *State {
	return &State{Type: TypeNoFile}
}

func DirState() *State {
	return &State{Type: TypeDir}
}

func FileState() *State {
	return &State{Type: TypeFile}
}

func LinkState(dest string, destType Type) *State {
	return &State{Type: TypeSymlink, Dest: dest, DestType: destType}
}

// String formats s as a string.
func (s *State) String() string {
	if s == nil {
		return "<nil>"
	}
	if s.Type == TypeSymlink {
		return fmt.Sprintf("%s to %s (%q)", s.Type, s.DestType, s.Dest)
	}
	return s.Type.String()
}
