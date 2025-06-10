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
	gaps           TargetGap
}

func (i Install) Analyze(pkg, item string, _ fs.DirEntry) error {
	itemGap, ok := i.gaps[item]
	if !ok {
		targetItem := path.Join(i.target, item)
		info, err := fs.Stat(i.fsys, targetItem)
		switch {
		case err == nil:
			itemGap = NewFileGap(info.Mode(), "")
		case !errors.Is(err, fs.ErrNotExist):
			// TODO: Record the error in the gap
			return err
		}
		i.gaps[item] = itemGap
	}

	if itemGap.Current != nil || itemGap.Desired != nil {
		return &ErrConflict{}
	}

	dest := path.Join(i.targetToSource, pkg, item)
	itemGap.Desired = &FileState{Mode: fs.ModeSymlink, Dest: dest}

	i.gaps[item] = itemGap
	return nil
}
