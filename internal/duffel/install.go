package duffel

import (
	"errors"
	"io/fs"
	"log/slog"
	"path"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

type Install struct {
	fsys           fs.FS
	source         string
	target         string
	targetToSource string
	tree           TargetTree
	logger         *slog.Logger
}

func (i Install) Analyze(pkg, item string, _ fs.DirEntry) error {
	log := i.logger.With(slog.Group("Install.Analyze()", "pkg", pkg, "item", item))
	status, ok := i.tree.Status(item)
	if !ok {
		log.Debug("no prior status")
		targetItem := path.Join(i.target, item)
		info, err := fs.Stat(i.fsys, targetItem)
		switch {
		case err == nil:
			log.Debug("no target file")
			status = NewStatus(info.Mode(), "")
		case !errors.Is(err, fs.ErrNotExist):
			// TODO: Record the error in the status
			return err
		}
		i.tree.Set(item, status)
		log.Debug("added status", "status", status)
	}

	if status.Current != nil || status.Desired != nil {
		log.Debug("conflict", "status", status)
		return &ErrConflict{}
	}

	dest := path.Join(i.targetToSource, pkg, item)
	status.Desired = &State{Mode: fs.ModeSymlink, Dest: dest}
	log.Debug("proposed status", "status", status)

	i.tree.Set(item, status)
	return nil
}
