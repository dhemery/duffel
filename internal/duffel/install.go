package duffel

import (
	"fmt"
	"path/filepath"
)

func Install(r *Request) error {
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
			if r.DryRun {
				fmt.Fprintln(r.Stdout, linkPath, "-->", linkDest)
				continue
			}
			err = r.FS.Symlink(linkDest, linkPath)
			if err != nil {
				return fmt.Errorf("making link: %w", err)
			}
		}

	}
	return nil
}
