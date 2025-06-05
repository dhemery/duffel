package duffel

import (
	"io/fs"
	"path"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

type Install struct {
	source         string
	target         string
	targetToSource string
	tree           TargetTree
}

func (v Install) Analyze(pkg, item string, _ fs.DirEntry) error {
	status, _ := v.tree.Status(item)
	// TODO: If not ok, stat the file
	if status.Desired != nil {
		return &ErrConflict{}
	}

	dest := path.Join(v.targetToSource, pkg, item)
	state := &State{Dest: dest}
	v.tree.Create(item, state)

	return nil
}
