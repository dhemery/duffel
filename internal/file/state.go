package file

import (
	"errors"
	"io/fs"
	"path"
)

// A State describes the state of an existing or planned file.
type State struct {
	Type     fs.FileMode
	Dest     string
	DestType fs.FileMode
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
