package analyze

import (
	"io/fs"
	"log/slog"
	"path"

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
		walkRoot: SourceItem{Source: source, Package: pkg},
		itemOp:   itemOp,
	}
}

// NewMergeOp returns a new package op that applies itemOp to items in mergeRoot.
func NewMergeOp(mergeRoot SourceItem, itemOp ItemOp) *PackageOp {
	return &PackageOp{
		walkRoot: mergeRoot,
		itemOp:   itemOp,
	}
}

type ItemFunc func(si SourceItem, entry fs.DirEntry, target string, state *file.State) (*file.State, error)

type PackageOp struct {
	walkRoot SourceItem
	itemOp   ItemOp
}

type Index interface {
	State(string) (*file.State, error)
	SetState(string, *file.State)
}

func (po *PackageOp) VisitFunc(target string, index Index, itemFunc ItemFunc, logger *slog.Logger) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		logger.Info("analyze", "name", name, "entry", entry, "err", err)
		if err != nil {
			return err
		}
		if name == po.walkRoot.String() {
			// Skip the dir being walked.
			return nil
		}

		item := po.walkRoot.WithItemFrom(name)
		targetItem := path.Join(target, item.Item)
		oldState, err := index.State(targetItem)
		if err != nil {
			return err
		}

		newState, err := itemFunc(item, entry, target, oldState)

		if err == nil || err == fs.SkipDir {
			index.SetState(targetItem, newState)
		}

		return err
	}
}
