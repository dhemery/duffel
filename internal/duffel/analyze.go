package duffel

import (
	"io/fs"
	"path"
	"path/filepath"
)

type ItemVisitor interface {
	Visit(pkg, item string, d fs.DirEntry) error
	Analyze(pkg, item string, d fs.DirEntry) error
}

type PkgAnalyst struct {
	FS          fs.FS
	Pkg         string
	SourcePkg   string
	ItemVisitor ItemVisitor
}

func NewPkgAnalyst(fsys fs.FS, source, pkg string, iv ItemVisitor) PkgAnalyst {
	return PkgAnalyst{
		FS:          fsys,
		Pkg:         pkg,
		SourcePkg:   path.Join(source, pkg),
		ItemVisitor: iv,
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
			return pa.ItemVisitor.Analyze(pa.Pkg, item, d)
		})
}

func (pa PkgAnalyst) VisitPath(path string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if path == pa.SourcePkg {
		// Source pkg is not an item.
		return nil
	}
	item, _ := filepath.Rel(pa.SourcePkg, path)
	return pa.ItemVisitor.Visit(pa.Pkg, item, entry)
}
