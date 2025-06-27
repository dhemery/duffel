package plan

import (
	"io/fs"
	"path"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

// Install is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type Install struct {
	TargetToSource string // The relative path from the target dir to the source dir.
}

// Apply describes the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// TargetState describes the state of the target file
// after earlier tasks.
func (i Install) Apply(pkg, item string, entry fs.DirEntry, targetState *file.State) (*file.State, error) {
	pkgItem := path.Join(pkg, item)
	itemAsDest := i.toLinkDest(pkgItem)

	if targetState == nil {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return &file.State{Mode: fs.ModeSymlink, Dest: itemAsDest}, err
	}

	if targetState.Mode.IsRegular() {
		return nil, &ErrInvalidTarget{
			Op: "install", Pkg: pkg, Item: item,
			Type: targetState.Mode.Type(),
			Err:  ErrIsFile,
		}
	}

	if targetState.Mode.IsDir() {
		// If target and pkg item are both dirs, install the pkg item's contents
		if entry.IsDir() {
			return targetState, nil
		}
		return nil, &ErrConflict{
			Op: "install", Pkg: pkg, Item: item,
			SourceType: entry.Type(), TargetState: targetState,
		}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, &ErrInvalidTarget{
			Op: "install", Pkg: pkg, Item: item,
			Type: targetState.Mode.Type(),
			Err:  ErrUnknownType,
		}
	}

	if targetState.Dest == itemAsDest {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return targetState, err
	}

	if !entry.IsDir() {
		return nil, &ErrConflict{
			Op: "install", Pkg: pkg, Item: item,
			SourceType: entry.Type(), TargetState: targetState,
		}
	}

	return nil, &ErrInvalidTarget{
		Op: "install", Pkg: pkg, Item: item,
		Type: targetState.Mode.Type(),
		Err:  ErrDestNotPkgItem,
	}
}

func (i Install) toLinkDest(pkgItem string) string {
	depth := strings.Count(pkgItem, "/") - 1
	prefix := strings.Repeat("../", depth)
	return path.Join(prefix, i.TargetToSource, pkgItem)
}
