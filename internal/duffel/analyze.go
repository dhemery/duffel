package duffel

import (
	"io/fs"
	"path"
	"path/filepath"
)

type ItemAnalyst interface {
	Analyze(pkg, item string, d fs.DirEntry) error
}

type PkgAnalyst struct {
	FS          fs.FS
	Pkg         string
	SourcePkg   string
	ItemAnalyst ItemAnalyst
}

func NewPkgAnalyst(fsys fs.FS, source, pkg string, ia ItemAnalyst) PkgAnalyst {
	return PkgAnalyst{
		FS:          fsys,
		Pkg:         pkg,
		SourcePkg:   path.Join(source, pkg),
		ItemAnalyst: ia,
	}
}

func (pa PkgAnalyst) Analyze() error {
	return fs.WalkDir(pa.FS, pa.SourcePkg,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Don't visit SourcePkg. It's a pkg, not an item.
			if path == pa.SourcePkg {
				return nil
			}
			item, _ := filepath.Rel(pa.SourcePkg, path)
			return pa.ItemAnalyst.Analyze(pa.Pkg, item, d)
		})
}
