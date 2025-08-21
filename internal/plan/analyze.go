// Package plan creates a plan to change the target tree
// to realize a series of package operations.
package plan

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// A ItemGoal for a [PackageGoal] to accomplish for each item in the package.
type ItemGoal string

const (
	GoalInstall ItemGoal = "install" // Install the package into the target tree.
	GoalMerge   ItemGoal = "merge"   // Merge the foreign package into the target tree.
)

// InstallPackage creates a [PackageGoal] to install a package.
func InstallPackage(source, pkg string) PackageGoal {
	return PackageGoal{
		root: NewSourcePath(source, pkg, ""),
		goal: GoalInstall,
	}
}

// MergePackage creates a [PackageGoal] to merge a previously installed package
// with a package currently being installed.
func MergePackage(itemPath SourcePath) PackageGoal {
	return PackageGoal{
		root: itemPath,
		goal: GoalMerge,
	}
}

// PackageGoal identifies a goal for the items in a package.
type PackageGoal struct {
	root SourcePath
	goal ItemGoal
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

func (pg PackageGoal) Goal() ItemGoal {
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
	fsys    fs.FS
	target  string
	index   *index
	install *installer
}

func (a *analyzer) Analyze(goal PackageGoal, l *slog.Logger) error {
	entryAnalyzer := EntryAnalyzer{
		WalkRoot:     goal.root,
		Target:       a.target,
		Index:        a.index,
		ItemAnalyzer: a.install,
		Logger:       l,
	}
	return fs.WalkDir(a.fsys, goal.Path(), entryAnalyzer.AnalyzeEntry)
}

// ItemAnalyzer identifies the goal states for target items.
type ItemAnalyzer interface {
	// The [Goal] to achieve.
	Goal() ItemGoal

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

type EntryAnalyzer struct {
	WalkRoot     SourcePath
	Target       string
	Index        Index
	ItemAnalyzer ItemAnalyzer
	Logger       *slog.Logger
}

func (ea EntryAnalyzer) AnalyzeEntry(name string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if name == ea.WalkRoot.String() {
		// Skip the dir being walked, but walk its contents.
		return nil
	}

	sourcePath := ea.WalkRoot.WithItemFrom(name)
	sourceType, err := file.TypeOf(entry.Type())
	if err != nil {
		return fmt.Errorf("%q: %w", sourcePath, err)
	}
	sourceItem := SourceItem{sourcePath, sourceType}

	targetPath := NewTargetPath(ea.Target, sourcePath.Item)
	tpAttr := slog.Any("path", targetPath)
	goalAttr := slog.Any("goal", ea.ItemAnalyzer.Goal())
	indexLogger := ea.Logger.With(goalAttr, "source", sourceItem, slog.Group("target", tpAttr))
	indexLogger.Info("start analyzing")

	targetState, err := ea.Index.State(targetPath.String(), indexLogger)
	if err != nil {
		return err
	}

	targetItem := TargetItem{targetPath, targetState}

	itemFuncLogger := indexLogger.With("target", targetItem)
	newState, err := ea.ItemAnalyzer.AnalyzeItem(sourceItem, targetItem, itemFuncLogger)

	if err == nil || err == fs.SkipDir {
		ea.Index.SetState(targetPath.String(), newState, indexLogger)
	}

	return err
}
