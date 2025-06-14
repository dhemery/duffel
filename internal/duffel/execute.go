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

	targetGap := Index{}
	install := Install{
		FS:             r.FS,
		TargetToSource: targetToSource,
	}

	var pkgAnalysts []PkgAnalyst
	for _, pkg := range r.Pkgs {
		pa := NewPkgAnalyst(r.FS, r.Target, r.Source, pkg, targetGap, install)
		pkgAnalysts = append(pkgAnalysts, pa)
	}

	for _, pa := range pkgAnalysts {
		err = pa.Analyze()
		if err != nil {
			break
		}
	}

	plan := NewPlan(r.Target, targetGap)

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}
	if err != nil {
		return err
	}

	return plan.Execute(r.FS)
}
