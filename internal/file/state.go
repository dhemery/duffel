package file

import (
	"errors"
	"io/fs"
	"log/slog"
	"path"
)

// A State describes the state of an existing or planned file.
type State struct {
	Type     fs.FileMode `json:"type"`
	Dest     string      `json:"dest"`
	DestType fs.FileMode `json:"desttype"`
}

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

func NewStater(fsys fs.FS) stater {
	return stater{fsys}
}

// A stater describes the states of files in a file system.
type stater struct {
	FS fs.FS
}

// State returns the state of the named file.
func (s stater) State(name string) (*State, error) {
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
