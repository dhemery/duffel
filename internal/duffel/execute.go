package duffel

import (
	"encoding/json"
	"io"
	"io/fs"
	"path/filepath"
)

type FS interface {
	fs.ReadDirFS
	Symlink(oldname, newname string) error
}

type Request struct {
	FS     FS
	Source string
	Target string
	Pkgs   []string
}

func Execute(r *Request, dryRun bool, w io.Writer) error {
	targetToSource, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return err
	}

	tree := TargetTree{}
	install := Install{
		fsys:           r.FS,
		target:         r.Target,
		targetToSource: targetToSource,
		tree:           tree,
	}

	var pkgAnalysts []PkgAnalyst
	for _, pkg := range r.Pkgs {
		pa := NewPkgAnalyst(r.FS, r.Source, pkg, install)
		pkgAnalysts = append(pkgAnalysts, pa)
	}

	for _, pa := range pkgAnalysts {
		err = pa.Analyze()
		if err != nil {
			break
		}
	}

	plan := NewPlan(r.Target, tree)

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}
	if err != nil {
		return err
	}

	return plan.Execute(r.FS)
}
