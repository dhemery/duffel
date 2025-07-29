package exec

import (
	"encoding/json/v2"
	"io"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/analyze"
)

type Request struct {
	FS     fs.FS
	Source string
	Target string
	Pkgs   []string
}

func Execute(r *Request, dryRun bool, w io.Writer, logger *slog.Logger) error {
	var pkgOps []*analyze.PackageOp
	for _, pkg := range r.Pkgs {
		pkgOp := analyze.NewPackageOp(r.Source, pkg, analyze.OpInstall)
		pkgOps = append(pkgOps, pkgOp)
	}

	specs, err := analyze.Analyze(r.FS, r.Target, pkgOps, logger)
	if err != nil {
		return err
	}

	p := New(r.Target, specs)
	if dryRun {
		return json.MarshalWrite(w, p)
	}

	return p.Execute(r.FS)
}
