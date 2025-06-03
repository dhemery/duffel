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
	status := v.image.Status(item)
	if status.WillExist() {
		return &ErrConflict{}
	}

	dest := path.Join(v.targetToSource, pkg, item)
	state := State{Dest: dest}
	v.image.Create(item, state)

	return nil
}
