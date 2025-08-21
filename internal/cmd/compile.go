package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	"github.com/dhemery/duffel/internal/plan"
)

// Compile compiles a [Command] satisfy the goals described by args.
func Compile(opts Options, args []string, fsys FS, cwd string, wout, werr io.Writer) (Command, error) {
	target := fullValidPath(cwd, opts.Target)
	source := fullValidPath(cwd, opts.Source)

	terr := validateDir(fsys, "target", target)
	serr := validateSource(fsys, source)
	if err := errors.Join(serr, terr); err != nil {
		return Command{}, err
	}

	var errs []error
	var goals []plan.PackageGoal
	for _, pkg := range args {
		goal := plan.InstallPackage(source, pkg)
		goals = append(goals, goal)
		errs = append(errs, validateGoal(fsys, goal))
	}

	if err := errors.Join(errs...); err != nil {
		return Command{}, err
	}

	logger := log.Logger(werr, &opts.LogLevel)

	var planFunc PlanFunc
	if opts.DryRun {
		planFunc = plan.Print(wout)
	} else {
		planFunc = plan.Execute(fsys, logger)
	}

	return Command{
		Planner:  plan.NewPlanner(fsys, target, goals, logger),
		PlanFunc: planFunc,
	}, nil
}

// validateDir checks that the named file exists and is a directory.
func validateDir(fsys fs.ReadLinkFS, desc, name string) error {
	info, err := fsys.Lstat(name)
	if err != nil {
		return fmt.Errorf("%s: %w", desc, err)
	}

	if !info.IsDir() {
		typ, _ := file.TypeOf(info.Mode().Type())
		return fmt.Errorf("%s %s (%s): %w: not a directory",
			desc, name, typ, fs.ErrInvalid)
	}

	return nil
}

// validateSource checks that source is a directory
// that contains a duffel file.
func validateSource(fsys fs.ReadLinkFS, source string) error {
	if err := validateDir(fsys, "source", source); err != nil {
		return err
	}
	sd, err := file.SourceDir(fsys, source)

	if errors.Is(err, fs.ErrNotExist) || sd != source {
		return fmt.Errorf("source %s: %w: no %s file",
			source, fs.ErrInvalid, file.SourceMarkerFile)
	}

	if err != nil {
		return fmt.Errorf("source %s: %w", source, err)
	}

	return nil
}

// validateGoal checks that goal's package
// is a directory and is a child of its source directory.
func validateGoal(fsys fs.ReadLinkFS, goal plan.PackageGoal) error {
	pname := goal.Package()
	ppath := goal.Path()
	pparent := path.Dir(ppath)
	source := goal.Source()
	if pparent != source {
		return fmt.Errorf("package %s (%s): %w: not a child of source (%s)",
			pname, ppath, fs.ErrInvalid, source)
	}

	return validateDir(fsys, "package", goal.Path())
}

// fullValidPath returns the relative path from / to name.
// If name is relative, it is joined onto cwd,
// which either is absolute or is assumed to be relative to /.
func fullValidPath(cwd, name string) string {
	name = filepath.ToSlash(name)
	if !path.IsAbs(name) {
		name = path.Join(filepath.ToSlash(cwd), name)
	}
	if path.IsAbs(name) {
		name = name[1:]
	}
	return path.Clean(name)
}
