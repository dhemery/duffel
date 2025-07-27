package exec

import (
	"encoding/json"
	"io"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/analyze"
	"github.com/dhemery/duffel/internal/file"
)

type Request struct {
	FS     fs.FS
	Source string
	Target string
	Pkgs   []string
}

func Execute(r *Request, dryRun bool, w io.Writer) error {
	stater := file.NewStater(r.FS)
	index := analyze.NewIndex(stater)

	pkgFinder := analyze.NewPkgFinder(r.FS)
	analyst := analyze.NewAnalyst(r.FS, r.Target, index)
	merger := analyze.NewMerger(pkgFinder, analyst)
	install := analyze.NewInstallOp(r.Source, r.Target, merger)

	var pkgOps []analyze.PkgOp
	for _, pkg := range r.Pkgs {
		sourcePkg := path.Join(r.Source, pkg)
		pkgOp := analyze.NewPkgOp(sourcePkg, install)
		pkgOps = append(pkgOps, pkgOp)
	}

	specs, err := analyst.Analyze(pkgOps...)
	if err != nil {
		return err
	}

	p := New(r.Target, specs)
	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(p)
	}

	return p.Execute(r.FS)
}
