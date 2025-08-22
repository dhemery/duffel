// Package plan creates a plan to change the target tree
// to realize a series of package operations.
package plan

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// A ItemGoal for a [DirGoal] to accomplish for each item in the package.
type ItemGoal string

const (
	// Install the package into the target tree.
	GoalInstall ItemGoal = "install"

	// Merge a previously installed directory into the directory being installed.
	GoalMerge ItemGoal = "merge"
)

// InstallPackage creates a [DirGoal] to install the items in a package.
func InstallPackage(source, pkg string) DirGoal {
	return DirGoal{
		Dir:  NewSourcePath(source, pkg, ""),
		Goal: GoalInstall,
	}
}

// MergeDir creates a [DirGoal] to merge a previously installed directory
// into the directory being installed.
func MergeDir(dir SourcePath) DirGoal {
	return DirGoal{
		Dir:  dir,
		Goal: GoalMerge,
	}
}

// DirGoal identifies a goal for the items in a directory.
type DirGoal struct {
	Dir  SourcePath // The directory that contains the items.
	Goal ItemGoal   // The goal to achieve for the items.
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

func (a *analyzer) Analyze(goal DirGoal, l *slog.Logger) error {
	entryAnalyzer := EntryAnalyzer{
		WalkRoot:     goal.Dir,
		Target:       a.target,
		ItemAnalyzer: a.install,
		Index:        a.index,
		Logger:       l.With(slog.Any("goal", goal.Goal)),
	}
	return fs.WalkDir(a.fsys, goal.Dir.String(), entryAnalyzer.Analyze)
}

// ItemAnalyzer identifies the goal states for target items.
type ItemAnalyzer interface {
	// Analyze analyzes the source and target to identify the goal state for the target item.
	// If the target was planned by previous operations,
	// the target item describes the previously planned goal state.
	// Otherwise it describes the state of the file in the target tree.
	Analyze(SourceItem, TargetItem, *slog.Logger) (file.State, error)
}

// An Index maintains the planned states of items in the target tree.
type Index interface {
	State(TargetPath, *slog.Logger) (file.State, error)
	SetState(TargetPath, file.State, *slog.Logger)
}

type EntryAnalyzer struct {
	WalkRoot     SourcePath   // The root dir that contains the items to analyze.
	Target       string       // The root of the target tree in which to achieve the goal states.
	ItemAnalyzer ItemAnalyzer // Analyzes each item to identify the goal state.
	Index        Index        // The known or planned states of target items.
	Logger       *slog.Logger
}

func (ea EntryAnalyzer) Analyze(name string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if name == ea.WalkRoot.String() {
		// Skip the root dir being walked, but walk its contents.
		return nil
	}

	sourcePath := ea.WalkRoot.WithItemFrom(name)
	sourceType, err := file.TypeOf(entry.Type())
	if err != nil {
		return fmt.Errorf("%q: %w", sourcePath, err)
	}

	sourceItem := SourceItem{sourcePath, sourceType}
	indexLogger := ea.Logger.With(slog.Any("source", sourceItem))

	targetPath := NewTargetPath(ea.Target, sourcePath.Item)

	targetState, err := ea.Index.State(targetPath, indexLogger)
	if err != nil {
		return err
	}

	targetItem := TargetItem{targetPath, targetState}

	newState, err := ea.ItemAnalyzer.Analyze(sourceItem, targetItem, ea.Logger)

	if err == nil || err == fs.SkipDir {
		ea.Index.SetState(targetPath, newState, indexLogger)
	}

	return err
}
