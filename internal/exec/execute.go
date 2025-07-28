package exec

import (
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
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

func Execute(r *Request, dryRun bool, w io.Writer, logger *slog.Logger) error {
	stater := file.NewStater(r.FS)
	index := analyze.NewIndex(stater, logger)

	analyst := analyze.NewAnalyst(r.FS, r.Source, r.Target, index, logger)

	var pkgOps []*analyze.PkgOp
	for _, pkg := range r.Pkgs {
		sourcePkg := path.Join(r.Source, pkg)
		pkgOp := analyze.NewPkgOp(sourcePkg, analyze.OpInstall, logger)
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
