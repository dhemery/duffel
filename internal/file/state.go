package file

import (
	"encoding/json"
	"io/fs"
)

// A State describes the state of an existing or planned file.
type State struct {
	Mode fs.FileMode
	Dest string
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
