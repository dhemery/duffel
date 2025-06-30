package file

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path"
)

// A State describes the state of an existing or planned file.
type State struct {
	Mode     fs.FileMode
	Dest     string
	DestMode fs.FileMode
}

// MarshalJSON returns the JSON representation of s.
// It represents the Mode field as a descriptive string
// by calling [fs.FileMode.String] on the Mode.
func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Mode string `json:"mode"`
		Dest string `json:"dest,omitzero"`
	}{
		Mode: s.Mode.String(),
		Dest: s.Dest,
	})
}

type StateLoader struct {
	FS fs.FS
}

func (s StateLoader) Load(name string) (*State, error) {
	info, err := fs.Lstat(s.FS, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state := &State{Mode: info.Mode()}
	if info.Mode()&fs.ModeSymlink != 0 {
		dest, err := fs.ReadLink(s.FS, name)
		if err != nil {
			return nil, err
		}
		destFull := path.Join(path.Dir(name), dest)
		destInfo, err := fs.Lstat(s.FS, destFull)
		if err != nil {
			return nil, err
		}
		state.Dest = dest
		state.DestMode = destInfo.Mode()
	}
	return state, nil
}

type DirStater struct {
	FS  fs.FS
	Dir string
}

func (s DirStater) State(name string) (*State, error) {
	fullname := path.Join(s.Dir, name)
	info, err := fs.Lstat(s.FS, fullname)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state := &State{Mode: info.Mode()}
	if info.Mode()&fs.ModeSymlink != 0 {
		dest, err := fs.ReadLink(s.FS, fullname)
		if err != nil {
			return nil, err
		}
		destFull := path.Join(path.Dir(fullname), dest)
		destInfo, err := fs.Lstat(s.FS, destFull)
		if err != nil {
			return nil, err
		}
		state.Dest = dest
		state.DestMode = destInfo.Mode()
	}
	return state, nil
}
