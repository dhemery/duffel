package duffel

import (
	"errors"
	"io/fs"
	"path"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

type Install struct {
	fsys           fs.FS
	target         string
	targetToSource string
	tree           TargetTree
}

func (i Install) Analyze(pkg, item string, _ fs.DirEntry) error {
	status, ok := i.tree[item]
	if !ok {
		targetItem := path.Join(i.target, item)
		info, err := fs.Stat(i.fsys, targetItem)
		switch {
		case err == nil:
			status = NewStatus(info.Mode(), "")
		case !errors.Is(err, fs.ErrNotExist):
			// TODO: Record the error in the status
			return err
		}
		i.tree[item] = status
	}

	if status.Current != nil || status.Desired != nil {
		return &ErrConflict{}
	}

	dest := path.Join(i.targetToSource, pkg, item)
	status.Desired = &State{Mode: fs.ModeSymlink, Dest: dest}

	i.tree[item] = status
	return nil
}
