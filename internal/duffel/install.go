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

// Install is an [ItemAnalyst] that identifies the gap
// the gap between an existing target file (if any)
// and the target file to install to represent an item in a pkg.
type Install struct {
	FS             fs.FS     // The file system of the source and target files to analyze.
	Target         string    // The target directory into which to install the files.
	TargetToSource string    // The relative path from the target dir to the source dir.
	TargetGap      TargetGap // The file gaps for each item that has been analyzed.
}

// Analyze records into i.TargetGap
// the desired state of a file to be installed from a pkg.
// If the file has not yet been analyzed,
// Analyze also records the current state of the file in the target dir.
// Pkg and item identify the file to be installed,
// and entry describes the state of the file in the source tree.
func (i Install) Analyze(pkg, item string, entry fs.DirEntry) error {
	itemGap, ok := i.TargetGap[item]
	if !ok {
		targetItem := path.Join(i.Target, item)
		info, err := fs.Stat(i.FS, targetItem)
		switch {
		case err == nil:
			itemGap = NewFileGap(info.Mode(), "")
		case !errors.Is(err, fs.ErrNotExist):
			// TODO: Record the error in the gap
			return err
		}
		i.TargetGap[item] = itemGap
	}

	if itemGap.Current != nil || itemGap.Desired != nil {
		return &ErrConflict{}
	}

	dest := path.Join(i.TargetToSource, pkg, item)
	itemGap.Desired = &FileState{Mode: fs.ModeSymlink, Dest: dest}

	i.TargetGap[item] = itemGap
	return nil
}

func (i Install) Visit(pkg, item string, entry fs.DirEntry) error {
	return nil
}
