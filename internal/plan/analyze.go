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

func NewAnalyzer(fsys fs.ReadLinkFS, target string, index *index) *analyzer {
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

func (po *PackageOp) Source() string {
	return po.walkRoot.Source
}

func (po *PackageOp) Package() string {
	return po.walkRoot.Package
}

func (po *PackageOp) Path() string {
	return po.walkRoot.String()
}

type Index interface {
	State(string, *slog.Logger) (file.State, error)
	SetState(string, file.State, *slog.Logger)
}

func (po *PackageOp) VisitFunc(target string, index Index, itemFunc ItemFunc, logger *slog.Logger) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		a := &analysis{dir: po.walkRoot, target: target, Index: index}
		return AnalyzeEntry(name, entry, err, a, itemFunc, logger)
	}
}

type analysis struct {
	Index
	dir    SourcePath
	target string
}

func (a *analysis) WalkDir() string {
	return a.dir.String()
}

type Analysis interface {
	WalkDir() string
	TargetPath(name string) TargetPath
	SourcePath(name string) SourcePath
	State(name string, l *slog.Logger) (file.State, error)
	SetState(name string, state file.State, l *slog.Logger)
}

func (a *analysis) TargetPath(name string) TargetPath {
	return NewTargetPath(a.target, a.SourcePath(name).Item)
}

func (a *analysis) SourcePath(name string) SourcePath {
	return a.dir.WithItemFrom(name)
}

func AnalyzeEntry(name string, entry fs.DirEntry, err error, a Analysis, itemFunc ItemFunc, l *slog.Logger) error {
	if err != nil {
		return err
	}
	if name == a.WalkDir() {
		// Skip the dir being walked, but walk its contents.
		return nil
	}

	sourcePath := a.SourcePath(name)
	targetPath := a.TargetPath(name)

	sourceType, err := file.TypeOf(entry.Type())
	if err != nil {
		return fmt.Errorf("%q: %w", sourcePath, err)
	}
	sourceItem := SourceItem{sourcePath, sourceType}

	tpAttr := slog.Any("path", targetPath)
	goalAttr := slog.Any("goal", GoalInstall)
	indexLogger := l.With(goalAttr, "source", sourceItem, slog.Group("target", tpAttr))
	indexLogger.Info("start analyzing")

	targetState, err := a.State(targetPath.String(), indexLogger)
	if err != nil {
		return err
	}

	targetItem := TargetItem{targetPath, targetState}

	itemFuncLogger := indexLogger.With("target", targetItem)
	newState, err := itemFunc(sourceItem, targetItem, itemFuncLogger)

	if err == nil || err == fs.SkipDir {
		a.SetState(targetPath.String(), newState, indexLogger)
	}

	return err
}
