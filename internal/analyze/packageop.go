package analyze

import (
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// ItemOp identifies a operation to apply to the items in a package.
type ItemOp int

const (
	OpInstall = ItemOp(1 << iota) // Installs items to the target tree.
	OpRemove                      // Currently unimplemented and ignored.
)

// NewPackageOp returns a new package op that applies itemOp to itens in the package.
func NewPackageOp(source, pkg string, itemOp ItemOp) *PackageOp {
	return &PackageOp{
		walkRoot: NewSourcePath(source, pkg, ""),
		itemOp:   itemOp,
	}
}

// NewMergeOp returns a new package op that applies itemOp to items in itemPath.
func NewMergeOp(itemPath SourcePath, itemOp ItemOp) *PackageOp {
	return &PackageOp{
		walkRoot: itemPath,
		itemOp:   itemOp,
	}
}

type ItemFunc func(sp SourcePath, entry fs.DirEntry, tp TargetPath, state *file.State) (*file.State, error)

// PackageOp walks a directory and applies an item operation to the visited files.
type PackageOp struct {
	walkRoot SourcePath
	itemOp   ItemOp
}

type Index interface {
	State(string) (*file.State, error)
	SetState(string, *file.State)
}

func (po *PackageOp) VisitFunc(target string, index Index, itemFunc ItemFunc, logger *slog.Logger) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == po.walkRoot.String() {
			// Skip the dir being walked.
			return nil
		}

		sourceItem := po.walkRoot.WithItemFrom(name)
		targetItem := NewTargetPath(target, sourceItem.Item)
		oldState, err := index.State(targetItem.String())
		if err != nil {
			return err
		}

		newState, err := itemFunc(sourceItem, entry, targetItem, oldState)

		if err == nil || err == fs.SkipDir {
			index.SetState(targetItem.String(), newState)
		}

		return err
	}
}
