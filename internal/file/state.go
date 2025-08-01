package file

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
)

// A State represents the state of an existing or planned file.
type State struct {
	Type     fs.FileMode `json:"type"`
	Dest     string      `json:"dest"`
	DestType fs.FileMode `json:"desttype"`
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
		slog.String("type", s.Type.String()[:1]),
		slog.String("dest", s.Dest),
		slog.String("desttype", s.DestType.String()[:1]),
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
	info, err := fs.Lstat(s.FS, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state := &State{Type: info.Mode().Type()}
	if info.Mode()&fs.ModeSymlink != 0 {
		dest, err := fs.ReadLink(s.FS, name)
		if err != nil {
			return nil, err
		}
		fullDest := path.Join(path.Dir(name), dest)
		destInfo, err := fs.Lstat(s.FS, fullDest)
		if err != nil {
			return nil, err
		}
		state.Dest = dest
		state.DestType = destInfo.Mode().Type()
	}
	return state, nil
}

func DirState() *State {
	return &State{Type: fs.ModeDir}
}

func FileState() *State {
	return &State{Type: 0}
}

func LinkState(dest string, destType fs.FileMode) *State {
	return &State{Type: fs.ModeSymlink, Dest: dest, DestType: destType}
}

// String formats s as a string.
func (s *State) String() string {
	if s == nil {
		return "<nil>"
	}
	if s.Type&fs.ModeSymlink != 0 {
		return fmt.Sprintf("%s to %s (%q)", DescribeType(s.Type), DescribeType(s.DestType), s.Dest)
	}
	return DescribeType(s.Type)
}

// DescribeType m's type in English.
func DescribeType(m fs.FileMode) string {
	switch {
	case m.IsRegular():
		return "a regular file"
	case m.IsDir():
		return "a directory"
	case m&fs.ModeSymlink != 0:
		return "a symlink"
	default:
		return fmt.Sprintf("unknown file type %s", m.String())
	}
}
