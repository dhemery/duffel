// Package plan creates a plan to change the target tree
// to realize a series of package operations.
package plan

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// A Goal for a [PackageGoal] to accomplish.
type Goal string

const (
	GoalInstall Goal = "install" // Install the package into the target tree.
	GoalMerge   Goal = "merge"   // Merge the foreign package into the target tree.
)

// InstallGoal creates a [PackageGoal] to install a package.
func InstallGoal(source, pkg string) PackageGoal {
	return PackageGoal{
		root: NewSourcePath(source, pkg, ""),
		goal: GoalInstall,
	}
}

// MergeGoal creates a [PackageGoal] to merge a previously installed package with a package currently being installed.
func MergeGoal(itemPath SourcePath) PackageGoal {
	return PackageGoal{
		root: itemPath,
		goal: GoalMerge,
	}
}

// PackageGoal identifies a goal for the items in a package.
type PackageGoal struct {
	root SourcePath
	goal Goal
}

func (pg PackageGoal) Source() string {
	return pg.root.Source
}

func (pg PackageGoal) Package() string {
	return pg.root.Package
}

func (pg PackageGoal) Path() string {
	return pg.root.String()
}

func (pg PackageGoal) Goal() Goal {
	return pg.goal
}

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
	fsys   fs.FS
	target string
	*index
	install *installer
}

func (a *analyzer) Target() string {
	return a.target
}

func (a *analyzer) Analyze(goal PackageGoal, l *slog.Logger) error {
	return fs.WalkDir(a.fsys, goal.Path(), func(name string, entry fs.DirEntry, err error) error {
		return AnalyzeEntry(name, entry, err, goal.root, a, a.install, l)
	})
}

// ItemAnalyzer identifies the goal states for target items.
type ItemAnalyzer interface {
	// The [Goal] to achieve.
	Goal() Goal

	// AnalyzeItem analyzes the source and target to identify the goal state for the target item.
	// If the target was planned by previous operations,
	// the target item describes the previously planned goal state.
	// Otherwise it describes the state of the file in the target tree.
	AnalyzeItem(SourceItem, TargetItem, *slog.Logger) (file.State, error)
}

type Index interface {
	State(string, *slog.Logger) (file.State, error)
	SetState(string, file.State, *slog.Logger)
}

type Analysis interface {
	Target() string
	State(name string, l *slog.Logger) (file.State, error)
	SetState(name string, state file.State, l *slog.Logger)
}

func AnalyzeEntry(name string, entry fs.DirEntry, err error,
	walkRoot SourcePath, analysis Analysis, ia ItemAnalyzer, l *slog.Logger) error {
	if err != nil {
		return err
	}
	if name == walkRoot.String() {
		// Skip the dir being walked, but walk its contents.
		return nil
	}

	sourcePath := walkRoot.WithItemFrom(name)
	sourceType, err := file.TypeOf(entry.Type())
	if err != nil {
		return fmt.Errorf("%q: %w", sourcePath, err)
	}
	sourceItem := SourceItem{sourcePath, sourceType}

	targetPath := NewTargetPath(analysis.Target(), sourcePath.Item)
	tpAttr := slog.Any("path", targetPath)
	goalAttr := slog.Any("goal", ia.Goal())
	indexLogger := l.With(goalAttr, "source", sourceItem, slog.Group("target", tpAttr))
	indexLogger.Info("start analyzing")

	targetState, err := analysis.State(targetPath.String(), indexLogger)
	if err != nil {
		return err
	}

	targetItem := TargetItem{targetPath, targetState}

	itemFuncLogger := indexLogger.With("target", targetItem)
	newState, err := ia.AnalyzeItem(sourceItem, targetItem, itemFuncLogger)

	if err == nil || err == fs.SkipDir {
		analysis.SetState(targetPath.String(), newState, indexLogger)
	}

	return err
}
