package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	"github.com/dhemery/duffel/internal/plan"
)

// Compile compiles a [Command] that perform the operations requested by args.
func Compile(opts Options, args []string, fsys FS, cwd string, wout, werr io.Writer) (Command, error) {
	target := relativeToRoot(opts.Target, cwd)
	source := relativeToRoot(opts.Source, cwd)

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
		return fmt.Errorf("%s %s (%s): not a directory",
			desc, name, typ)
	}

	return nil
}

// validateSource checks that source is a directory
// that contains a duffel file.
func validateSource(fsys fs.ReadLinkFS, source string) error {
	if err := validateDir(fsys, "source", source); err != nil {
		return err
	}
	return nil
}

// validatePackage checks that op's package
// is a directory and is a child of its source directory.
func validatePackage(fsys fs.ReadLinkFS, op *plan.PackageOp) error {
	pname := op.Package()
	if path.IsAbs(pname) {
		return fmt.Errorf("package %s: is absolute", pname)
	}

	ppath := op.Path()
	pparent := path.Dir(ppath)
	source := op.Source()
	if pparent != source {
		return fmt.Errorf("package %s (%s): not a child of source %s", pname, ppath, source)
	}

	return validateDir(fsys, "package", op.Path())
}

// Abs returns the path to filename relative to the root of the file system.
// If filename is relative, it is joined onto cwd before being made relative to the root.
// Cwd is assumed to be relative to the root.
func relativeToRoot(filename, cwd string) string {
	if path.IsAbs(filename) {
		return filename[1:]
	}
	return path.Join(cwd, filename)
}
