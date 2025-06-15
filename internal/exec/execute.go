package exec

import (
	"encoding/json"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/dhemery/duffel/internal/item"
	"github.com/dhemery/duffel/internal/plan"
)

type FS interface {
	fs.ReadDirFS
	plan.SymlinkFS
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

	index := item.NewIndex(nil)
	install := plan.Install{
		FS:             r.FS,
		TargetToSource: targetToSource,
	}

	var pkgAnalysts []plan.PkgAnalyst
	for _, pkg := range r.Pkgs {
		pa := plan.NewPkgAnalyst(r.FS, r.Target, r.Source, pkg, index, install)
		pkgAnalysts = append(pkgAnalysts, pa)
	}

	for _, pa := range pkgAnalysts {
		err = pa.Analyze()
		if err != nil {
			break
		}
	}

	plan := plan.New(r.Target, index)

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}
	if err != nil {
		return err
	}

	return plan.Execute(r.FS)
}
