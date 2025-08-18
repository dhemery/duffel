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
func Compile(args []string, fsys FS, wout, werr io.Writer) (Command, error) {
	opts, args, err := ParseArgs(args)
	if err != nil {
		return Command{}, err
	}

	target, terr := relativeToRoot("target", opts.target)
	source, serr := relativeToRoot("source", opts.source)
	if err := errors.Join(serr, terr); err != nil {
		return Command{}, err
	}

	terr = validateDir(fsys, "target", target)
	serr = validateSource(fsys, source)
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

	logger := log.Logger(werr, &opts.logLevel)

	var planFunc PlanFunc
	if opts.dryRun {
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

// relativeToRoot converts filename to a name relative to [Root].
// If filename is relative, it is joined to the currrent working directory
// before being made relative to [Root].
func relativeToRoot(desc, filename string) (string, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return "", fmt.Errorf("%s: %w", desc, err)
	}

	rel, err := filepath.Rel(Root, abs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", desc, err)
	}
	return rel, err
}
