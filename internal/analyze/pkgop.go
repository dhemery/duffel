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
	OpInstall = ItemOp(1 << iota)
	OpRemove  // Currently unimplemented and ignored.
)

func NewPkgOp(source, pkg string, itemOp ItemOp) *PkgOp {
	return &PkgOp{
		root:   SourceItem{Source: source, Package: pkg},
		itemOp: itemOp,
	}
}

func NewMergeOp(mergeRoot SourceItem, itemOp ItemOp) *PkgOp {
	return &PkgOp{
		root:   mergeRoot,
		itemOp: itemOp,
	}
}

type ItemFunc func(si SourceItem, entry fs.DirEntry, target string, state *file.State) (*file.State, error)

type PkgOp struct {
	root   SourceItem
	itemOp ItemOp
}

func (po *PkgOp) VisitFunc(target string, index Index, itemFunc ItemFunc, logger *slog.Logger) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		logger.Info("analyze", "name", name, "entry", entry, "err", err)
		if err != nil {
			return err
		}
		if name == po.root.String() {
			// Skip the dir being walked.
			return nil
		}

		item := po.root.WithItemFrom(name)
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
