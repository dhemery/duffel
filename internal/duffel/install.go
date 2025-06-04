package duffel

import (
	"io/fs"
	"path"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

type InstallVisitor struct {
	source         string
	target         string
	targetToSource string
	image          Image
}

func (v InstallVisitor) VisitItem(pkg, item string, _ fs.DirEntry) error {
	status, _ := v.image.Status(item)
	// TODO: If not ok, stat the file
	if status.Desired != nil {
		return &ErrConflict{}
	}

	dest := path.Join(v.targetToSource, pkg, item)
	state := &State{Dest: dest}
	v.image.Create(item, state)

	return nil
}
