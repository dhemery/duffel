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

// Compile compiles a [Command] that perform the operations requested by args.
func Compile(opts Options, args []string, fsys FS, cwd string, wout, werr io.Writer) (Command, error) {
	target := fullValidPath(cwd, opts.Target)
	source := fullValidPath(cwd, opts.Source)

	terr := validateDir(fsys, "target", target)
	serr := validateSource(fsys, source)
	if err := errors.Join(serr, terr); err != nil {
		return Command{}, err
	}

	var errs []error
	var ops []*plan.PackageOp
	for _, pkg := range args {
		op := plan.NewInstallOp(source, pkg)
		ops = append(ops, op)
		errs = append(errs, validatePackage(fsys, op))
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
		Planner:  plan.NewPlanner(fsys, target, ops, logger),
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

// validatePackage checks that op's package
// is a directory and is a child of its source directory.
func validatePackage(fsys fs.ReadLinkFS, op *plan.PackageOp) error {
	pname := op.Package()
	ppath := op.Path()
	pparent := path.Dir(ppath)
	source := op.Source()
	if pparent != source {
		return fmt.Errorf("package %s (%s): %w: not a child of source (%s)",
			pname, ppath, fs.ErrInvalid, source)
	}

	return validateDir(fsys, "package", op.Path())
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
