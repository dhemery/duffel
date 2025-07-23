package file

import (
	"encoding/json"
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

// MarshalJSON returns the JSON representation of s.
// It represents the Mode field as a descriptive string
// by calling [fs.FileMode.String] on the Mode.
func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Mode string `json:"mode"`
		Dest string `json:"dest,omitzero"`
	}{
		Mode: s.Type.String(),
		Dest: s.Dest,
	})
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
