package duffel

import (
	"io/fs"
	"path"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

// Install is an [ItemAdvisor] that identifies the desired states
// of the target files that correspond to the given pkg items.
type Install struct {
	FS             fs.FS  // The file system of the source and target files to analyze.
	TargetToSource string // The relative path from the target dir to the source dir.
}

// Visit returns a FileState describing the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// PriorAdvice describes the desired state of the target file
// as determined by prior advisors.
func (i Install) Advise(pkg, item string, entry fs.DirEntry, priorAdvice *FileState) (*FileState, error) {
	if priorAdvice != nil {
		return nil, &ErrConflict{}
	}

	dest := path.Join(i.TargetToSource, pkg, item)
	return &FileState{Mode: fs.ModeSymlink, Dest: dest}, nil
}
