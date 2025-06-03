package duffel

import (
	"path"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

type InstallVisitor struct {
	target         string
	targetToSource string
}

func (v InstallVisitor) Visit(source, pkg, item string, image Image) error {
	status := image.Status(item)
	if status.WillExist() {
		return &ErrConflict{}
	}

	dest := path.Join(v.targetToSource, pkg, item)
	state := State{Dest: dest}
	image.Create(item, state)

	return nil
}
