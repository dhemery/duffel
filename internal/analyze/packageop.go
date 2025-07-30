package analyze

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// ItemOp identifies a operation to apply to the items in a package.
type ItemOp int

func (op ItemOp) String() string {
	switch op {
	case OpInstall:
		return "install"
	case OpRemove:
		return "remove"
	}
	return fmt.Sprintf("unknown goal: %d", op)
}

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

type ItemFunc func(SourcePath, fs.DirEntry, TargetPath, *file.State, *slog.Logger) (*file.State, error)

// PackageOp walks a directory and applies an item operation to the visited files.
type PackageOp struct {
	walkRoot SourcePath
	itemOp   ItemOp
}

type Index interface {
	State(string, *slog.Logger) (*file.State, error)
	SetState(string, *file.State, *slog.Logger)
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

		sourcePath := po.walkRoot.WithItemFrom(name)
		targetPath := NewTargetPath(target, sourcePath.Item)

		goalAttr := slog.String("goal", po.itemOp.String())
		sGroup := slog.Group("source", "path", sourcePath, "entry", entry)
		tpAttr := slog.Any("path", targetPath)
		indexLogger := logger.With(goalAttr, sGroup, slog.Group("target", tpAttr))
		indexLogger.Info("analyzing")

		targetState, err := index.State(targetPath.String(), indexLogger)
		if err != nil {
			return err
		}

		tGroup := slog.Group("target", tpAttr, "state", targetState)
		itemOpLogger := indexLogger.With(tGroup)
		newState, err := itemFunc(sourcePath, entry, targetPath, targetState, itemOpLogger)

		if err == nil || err == fs.SkipDir {
			index.SetState(targetPath.String(), newState, indexLogger)
		}

		return err
	}
}
