package plan

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

type PkgFinder interface {
	FindPkg(name string) (string, error)
}

func NewInstallOp(source, target string, pkgFinder PkgFinder, analyzer Analyzer) installOp {
	return installOp{
		target:    target,
		source:    source,
		pkgFinder: pkgFinder,
		analyzer:  analyzer,
	}
}

// installOp is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type installOp struct {
	source    string
	target    string
	pkgFinder PkgFinder
	analyzer  Analyzer
}

// Apply describes the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// TargetState describes the state of the target file
// after earlier tasks.
func (op installOp) Apply(name string, entry fs.DirEntry, targetState *file.State) (*file.State, error) {
	pkgItem := name[len(op.source)+1:]
	pkg, item, _ := strings.Cut(pkgItem, "/")
	targetItem := path.Join(op.target, item)
	itemAsDest, _ := filepath.Rel(path.Dir(targetItem), name)

	if targetState == nil {
		// There is no target item, so we're free to create a link to the pkg item.
		var err error
		if entry.IsDir() {
			// Linking to the dir installs the dir and its contents.
			// There's no need to walk its contents.
			err = fs.SkipDir
		}
		return &file.State{Mode: fs.ModeSymlink, Dest: itemAsDest}, err
	}

	// At this point, we know that the target exists,
	// either on the file system or as planned by a previous operation.

	if targetState.Mode.IsRegular() {
		// Cannot modify an existing regular file.
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	if targetState.Mode.IsDir() {
		if entry.IsDir() {
			// The target and pkg item are both dirs.
			// Return the target state unchanged,
			// and a nil error to walk the pkg item's contents.
			return targetState, nil
		}

		// The target is a dir, but the pkg item is not.
		// Cannot merge the target dir with a non-dir pkg item.
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	// At this point, we know that the existing target is a symlink.

	if targetState.Dest == itemAsDest {
		// The target already links to this pkg item.
		// There's nothing more to do.
		var err error
		if entry.IsDir() {
			// We're done with this item. Do not walk its contents.
			err = fs.SkipDir
		}
		return targetState, err
	}

	if !targetState.DestMode.IsDir() {
		// Target links to a non-dir. Cannot merge with that.
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	if !entry.IsDir() {
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	// The package item is a dir and the target is a link to a dir.
	// If the link target is a foreign package item
	// (i.e. it is in some package in some duffel source),
	// then merge the two by changing the target to a dir,
	// analyzing the content of the foreign package item,
	// and returning a nil error to walk the contents of the current item.

	// First, find out if the target's link dest is a foreign package item.
	foreignItem := path.Join(path.Dir(targetItem), targetState.Dest)
	foreignPkg, err := op.pkgFinder.FindPkg(foreignItem)
	if err != nil {
		return nil, err
	}

	foreignTarget := "FIXME/installOp/foreign/target"

	foreignPkgOp := NewForeignPkgOp(foreignPkg, foreignItem, op)

	// First, analyze the contents of the target's destination,
	// as if the target were already a dir.
	err = op.analyzer.Analyze(foreignPkgOp, foreignTarget)
	if err != nil {
		return nil, err
	}

	// No conflicts installing the target destination's contents.
	// Now change the target to a dir, and walk the current item
	// to install its contents into the dir.
	dirState := &file.State{Mode: fs.ModeDir | 0o755}
	return dirState, nil
}
