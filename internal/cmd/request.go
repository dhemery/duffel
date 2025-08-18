package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/plan"
)

type Request struct {
	target string
	source string
	ops    []*plan.PackageOp
}

func CompileRequest(root string, fsys fs.ReadLinkFS, opts Options, args []string) (Request, error) {
	target, terr := toFSPath("target", root, opts.target)
	source, serr := toFSPath("source", root, opts.source)
	if err := errors.Join(serr, terr); err != nil {
		return Request{}, err
	}

	terr = validateDir(fsys, "target", target)
	serr = validateSource(fsys, source)
	if err := errors.Join(serr, terr); err != nil {
		return Request{}, err
	}

	r := Request{
		target: target,
		source: source,
	}

	var errs []error
	for _, pkg := range args {
		op := plan.NewInstallOp(source, pkg)
		r.ops = append(r.ops, op)
		errs = append(errs, validatePackage(fsys, op))
	}
	return r, errors.Join(errs...)
}

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

func validateSource(fsys fs.ReadLinkFS, source string) error {
	if err := validateDir(fsys, "source", source); err != nil {
		return err
	}
	return nil
}

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

func toFSPath(desc, root, name string) (string, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", desc, err)
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", desc, err)
	}
	return rel, err
}
