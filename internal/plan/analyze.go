// Package plan creates a plan to change the target tree
// to realize a series of package operations.
package plan

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// A Goal for a [PackageOp] to accomplish.
type Goal string

const (
	GoalInstall Goal = "install" // Install the package into the target tree.
	GoalMerge   Goal = "merge"   // Merge the foreign package into the target tree.
)

func NewAnalyzer(fsys fs.FS, target string, index *index) *analyzer {
	analyst := &analyzer{
		fsys:   fsys,
		target: target,
		index:  index,
	}
	itemizer := NewItemizer(fsys)
	merger := NewMerger(itemizer, analyst)
	analyst.install = &installer{merger}
	return analyst
}

type analyzer struct {
	fsys    fs.FS
	target  string
	index   *index
	install *installer
}

func (a *analyzer) Analyze(op *PackageOp, l *slog.Logger) error {
	return fs.WalkDir(a.fsys, op.walkRoot.String(), op.VisitFunc(a.target, a.index, a.install.Analyze, l))
}

// NewInstallOp creates a [PackageOp] to install a package.
func NewInstallOp(source, pkg string) *PackageOp {
	return &PackageOp{
		walkRoot: NewSourcePath(source, pkg, ""),
		goal:     GoalInstall,
	}
}

// NewMergeOp creates a [PackageOp] to merge a previously installed package with a package currently being installed.
func NewMergeOp(itemPath SourcePath) *PackageOp {
	return &PackageOp{
		walkRoot: itemPath,
		goal:     GoalMerge,
	}
}

// ItemFunc analyzes the source and target to identify the planned state for the target item.
// If the target was planned by previous operations, the target item describes the previously planned state.
// Otherwise it describes the state of the file in the target tree.
type ItemFunc func(SourceItem, TargetItem, *slog.Logger) (file.State, error)

// PackageOp plans how to achieve a goal for a package.
type PackageOp struct {
	walkRoot SourcePath
	goal     Goal
}

type Index interface {
	State(string, *slog.Logger) (file.State, error)
	SetState(string, file.State, *slog.Logger)
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

		sourceType, err := file.TypeOf(entry.Type())
		if err != nil {
			return fmt.Errorf("%q: %w", sourcePath, err)
		}
		sourceItem := SourceItem{sourcePath, sourceType}

		tpAttr := slog.Any("path", targetPath)
		goalAttr := slog.Any("goal", po.goal)
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
