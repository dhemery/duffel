package analyze

import (
	"io/fs"
	"log/slog"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

func NewPkgOp(pkgDir string, itemOp ItemOp, logger *slog.Logger) *PkgOp {
	return &PkgOp{
		walkDir: pkgDir,
		pkgDir:  pkgDir,
		itemOp:  itemOp,
		log:     logger,
	}
}

func NewMergePkgOp(pkgDir, mergeItem string, itemOp ItemOp, logger *slog.Logger) *PkgOp {
	return &PkgOp{
		pkgDir:  pkgDir,
		walkDir: path.Join(pkgDir, mergeItem),
		itemOp:  itemOp,
		log:     logger,
	}
}

type ItemFunc func(name string, entry fs.DirEntry, state *file.State) (*file.State, error)

type PkgOp struct {
	pkgDir  string
	walkDir string
	itemOp  ItemOp
	log     *slog.Logger
}

func (po *PkgOp) VisitFunc(target string, index Index, itemFunc ItemFunc) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		po.log.Info("analyze", "name", name, "entry", entry, "err", err)
		if err != nil {
			return err
		}
		if name == po.walkDir {
			// Skip the dir being walked.
			return nil
		}

		item := name[len(po.pkgDir)+1:]
		targetItem := path.Join(target, item)
		oldState, err := index.State(targetItem)
		if err != nil {
			return err
		}

		newState, err := itemFunc(name, entry, oldState)

		if err == nil || err == fs.SkipDir {
			index.SetState(targetItem, newState)
		}

		return err
	}
}
