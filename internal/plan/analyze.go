// Package plan creates a plan to change the target tree
// to realize a series of package operations.
package plan

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

// InstallPackage creates a [DirGoal] to install the items in a package.
func InstallPackage(source, pkg string) DirGoal {
	return DirGoal{
		dir:  newSourcePath(source, pkg, ""),
		goal: goalInstall,
	}
}

// DirGoal identifies a goal for the items in a directory.
type DirGoal struct {
	dir  SourcePath // The directory that contains the items.
	goal itemGoal   // The goal to achieve for the items.
}

// mergeDir creates a [DirGoal] to merge a previously installed directory
// into the directory being installed.
func mergeDir(dir SourcePath) DirGoal {
	return DirGoal{
		dir:  dir,
		goal: goalMerge,
	}
}

// An itemGoal is a goal for a [DirGoal] to accomplish for each item in the directory.
type itemGoal string

const (
	// Install the package into the target tree.
	goalInstall itemGoal = "install"

	// Merge a previously installed directory into the directory being installed.
	goalMerge itemGoal = "merge"
)

func newAnalyzer(fsys fs.ReadLinkFS, target string, index *specIndex) *analyzer {
	analyst := &analyzer{
		fsys:   fsys,
		target: target,
		index:  index,
	}
	itemizer := itemizer{fsys}
	merger := newMerger(itemizer, analyst)
	analyst.install = &installer{merger}
	return analyst
}

type analyzer struct {
	fsys    fs.FS
	target  string
	index   *specIndex
	install *installer
}

func (a *analyzer) analyze(goal DirGoal, l *slog.Logger) error {
	entryAnalyzer := entryAnalyzer{
		root:         goal.dir,
		target:       a.target,
		itemAnalyzer: a.install,
		index:        a.index,
		logger:       l.With(slog.Any("goal", goal.goal)),
	}
	return fs.WalkDir(a.fsys, goal.dir.String(), entryAnalyzer.analyze)
}

// itemAnalyzer identifies the goal states for target items.
type itemAnalyzer interface {
	// analyze analyzes the source and target to identify the goal state for the target item.
	// If the target was planned by previous operations,
	// the target item describes the previously planned goal state.
	// Otherwise it describes the state of the file in the target tree.
	analyze(SourceItem, TargetItem, *slog.Logger) (file.State, error)
}

// An index maintains the planned states of items in the target tree.
type index interface {
	state(TargetPath, *slog.Logger) (file.State, error)
	setState(TargetPath, file.State, *slog.Logger)
}

type entryAnalyzer struct {
	root         SourcePath   // The root dir that contains the items to analyze.
	target       string       // The root of the target tree in which to achieve the goal states.
	itemAnalyzer itemAnalyzer // Analyzes each item to identify the goal state.
	index        index        // The known or planned states of target items.
	logger       *slog.Logger
}

func (ea entryAnalyzer) analyze(name string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if name == ea.root.String() {
		// Skip the root dir being walked, but walk its contents.
		return nil
	}

	sourcePath := ea.root.withItemFrom(name)
	sourceType, err := file.TypeOf(entry.Type())
	if err != nil {
		return fmt.Errorf("%q: %w", sourcePath, err)
	}

	sourceItem := SourceItem{sourcePath, sourceType}
	indexLogger := ea.logger.With(slog.Any("source", sourceItem))

	targetPath := newTargetPath(ea.target, sourcePath.Item)

	targetState, err := ea.index.state(targetPath, indexLogger)
	if err != nil {
		return err
	}

	targetItem := TargetItem{targetPath, targetState}

	newState, err := ea.itemAnalyzer.analyze(sourceItem, targetItem, ea.logger)

	if err == nil || err == fs.SkipDir {
		ea.index.setState(targetPath, newState, indexLogger)
	}

	return err
}
