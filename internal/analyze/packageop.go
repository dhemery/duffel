package analyze

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// Goal identifies the goal for a PackageOp to plan for the items in a package.
type Goal int

func (op Goal) String() string {
	switch op {
	case GoalInstall:
		return "install"
	case GoalMerge:
		return "merge"
	}
	return fmt.Sprintf("unknown goal: %d", op)
}

const (
	GoalInstall Goal = 1 // Install a package into the target tree.
	GoalMerge   Goal = 2 // Merge a foreign package into the target tree.
)

// NewPackageOp creates a [PackageOp] to achieve the goal for the package.
func NewPackageOp(source, pkg string, goal Goal) *PackageOp {
	return &PackageOp{
		walkRoot: NewSourcePath(source, pkg, ""),
		goal:     goal,
	}
}

// NewMergeOp creates a [PackageOp] to merge a package.
func NewMergeOp(itemPath SourcePath) *PackageOp {
	return &PackageOp{
		walkRoot: itemPath,
		goal:     GoalMerge,
	}
}

type ItemFunc func(SourceItem, TargetItem, *slog.Logger) (*file.State, error)

// PackageOp plans how to achieve a goal for a package.
type PackageOp struct {
	walkRoot SourcePath
	goal     Goal
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

		goalAttr := slog.String("goal", po.goal.String())
		sourceItem := SourceItem{sourcePath, entry}
		tpAttr := slog.Any("path", targetPath)
		indexLogger := logger.With(goalAttr, "source", sourceItem, slog.Group("target", tpAttr))
		indexLogger.Info("start analyzing")

		targetState, err := index.State(targetPath.String(), indexLogger)
		if err != nil {
			return err
		}

		targetItem := TargetItem{targetPath, targetState}

		itemFuncLogger := indexLogger.With("target", targetItem)
		newState, err := itemFunc(sourceItem, targetItem, itemFuncLogger)

		if err == nil || err == fs.SkipDir {
			index.SetState(targetPath.String(), newState, indexLogger)
		}

		return err
	}
}
