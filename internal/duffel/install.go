package duffel

import (
	"fmt"
	"io"
	"path/filepath"
)

func Install(r *Request) error {
	plan := Plan{}
	sourceLinkDest, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}

	for _, pkg := range r.Pkgs {
		pkgDir := filepath.Join(r.Source, pkg)
		pkgLinkDest := filepath.Join(sourceLinkDest, pkg)
		entries, err := r.FS.ReadDir(pkgDir)
		if err != nil {
			return fmt.Errorf("reading package %s: %w", pkg, err)
		}
		for _, e := range entries {
			linkPath := filepath.Join(r.Target, e.Name())
			linkDest := filepath.Join(pkgLinkDest, e.Name())
			plan = append(plan, MakeLink{Path: linkPath, Dest: linkDest})
		}

	}
	if r.DryRun {
		_, err := plan.WriteTo(r.Stdout)
		return err
	}
	return plan.Execute(r.FS)
}

type Action interface {
	io.WriterTo
	Execute(fsys FS) error
}
type Plan []Action

func (p Plan) Execute(fsys FS) error {
	for _, action := range p {
		if err := action.Execute(fsys); err != nil {
			return err
		}
	}
	return nil
}

func (p Plan) WriteTo(w io.Writer) (int64, error) {
	var totalN int64
	for _, action := range p {
		n, err := action.WriteTo(w)
		totalN += int64(n)
		if err != nil {
			return totalN, err
		}
	}
	return totalN, nil
}

type MakeLink struct {
	Path string
	Dest string
}

func (a MakeLink) WriteTo(w io.Writer) (int64, error) {
	s := fmt.Sprint(a.Path, " --> ", a.Dest)
	n, err := w.Write([]byte(s))
	return int64(n), err
}

func (a MakeLink) Execute(fsys FS) error {
	return fsys.Symlink(a.Dest, a.Path)
}
