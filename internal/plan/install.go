package plan

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

type InstallMerger interface {
	Merge(name string, l *slog.Logger) error
}

// installer describes the installed state
// of the target item file that corresponds
// to each given source item file.
type installer struct {
	merger InstallMerger
}

func (i installer) Goal() ItemGoal {
	return GoalInstall
}

// Analyze returns the state of the target item file
// that would result from installing the source item file.
func (i installer) Analyze(s SourceItem, t TargetItem, l *slog.Logger) (file.State, error) {
	var state file.State
	targetPath := t.Path
	targetState := t.State
	sourceType := s.Type
	itemAsDest := targetPath.PathTo(s.Path.String())

	if targetState.Type.IsNoFile() {
		// There is no target file, so we're free to create a link to the source item.
		var err error
		if sourceType.IsDir() {
			// Linking to the dir installs the dir and its contents.
			// There's no need to walk its contents.
			err = fs.SkipDir
		}
		return file.LinkState(itemAsDest, sourceType), err
	}

	// At this point, we know that the tasks planned earlier (if any)
	// either create or preserve a file at the target path.

	targetType := targetState.Type
	if targetType.IsRegular() {
		// Cannot modify an existing regular target file.
		return state, &ConflictError{Source: s, Target: t}
	}

	if targetType.IsDir() {
		if sourceType.IsDir() {
			// The target and source item are both dirs.
			// Return the target state unchanged,
			// and a nil error to walk the pkg item's contents.
			return targetState, nil
		}

		// The target item is a dir, but the source item is not.
		// Cannot merge the target dir with a non-dir source item.
		return state, &ConflictError{Source: s, Target: t}
	}

	if !targetType.IsLink() {
		// Target item is not file, dir, or link.
		return state, &ConflictError{Source: s, Target: t}
	}

	// At this point, we know that the target is a symlink.

	targetDest := targetState.Dest

	if targetDest.Path == itemAsDest {
		// The target symlink already points to the source item.
		// There's nothing more to do.
		var err error
		if sourceType.IsDir() {
			// We're done with this item. Do not walk its contents.
			err = fs.SkipDir
		}
		return targetState, err
	}

	if targetDest.Type.IsNoFile() {
		// The target links to nothing, so replace it with a link to the source item.
		var err error
		if sourceType.IsDir() {
			// Linking to the dir installs the dir and its contents.
			// There's no need to walk its contents.
			err = fs.SkipDir
		}
		return file.LinkState(itemAsDest, sourceType), err
	}

	if !targetDest.Type.IsDir() {
		// The target item's link destination is not a dir. Cannot merge.
		return state, &ConflictError{Source: s, Target: t}
	}

	if !sourceType.IsDir() {
		// Tne entry is not a dir. Cannot merge.
		return state, &ConflictError{Source: s, Target: t}
	}

	// The package item is a dir and the target is a link to a dir.
	// Try to merge the target item.
	mergeDir := targetPath.Resolve(targetState.Dest.Path)
	l.Debug("merging", slog.Any("source", s), slog.Any("target", t), slog.String("merge_dir", mergeDir))
	err := i.merger.Merge(mergeDir, l)
	if err != nil {
		return state, err
	}

	// No conflicts merging the target destination dir.
	// Now change the target to a dir, and walk the source item
	// to install its contents into the dir.
	return file.DirState(), nil
}

type ConflictError struct {
	Source SourceItem
	Target TargetItem
}

func (ce *ConflictError) Error() string {
	return fmt.Sprintf("install conflict: source item %q is %s, target item %q is %s",
		ce.Source.Path, ce.Source.Type, ce.Target.Path, ce.Target.State)
}
