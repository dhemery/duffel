package plan

import (
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

// Install is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type Install struct {
	TargetToSource string // The relative path from the target dir to the source dir.
}

// Apply returns a State describing the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// InState describes the desired state of the target file
// as determined by prior analysis.
func (i Install) Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error) {
	targetToItem := path.Join(i.TargetToSource, pkg, item)

	if inState == nil {
		// No conflicting target state. Link to this pkg item.
		return &file.State{Mode: fs.ModeSymlink, Dest: targetToItem}, nil
	}

	if inState.Dest == targetToItem {
		// Target already links to this pkg item.
		return inState, nil
	}

	return nil, &ErrConflict{}
}
