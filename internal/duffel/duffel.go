package duffel

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type FS interface {
	Join(path string) string
	ReadDir(path string) ([]fs.DirEntry, error)
	Lstat(path string) (fs.FileInfo, error)
	MkdirAll(path string, perm fs.FileMode) error
	Symlink(old, new string) error
}

type Request struct {
	FS     FS
	Source string
	Target string
	Pkgs   []string
	DryRun bool
	Stdout io.Writer
	Stderr io.Writer
}

func Install(r Request) error {
	sourceLinkDest, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}

	for _, pkg := range r.Pkgs {
		pkgDir := filepath.Join(r.Source, pkg)
		pkgLinkDest := filepath.Join(sourceLinkDest, pkg)
		entries, err := r.FS.ReadDir(pkgDir)
		if err != nil {
			return fmt.Errorf("reading entries of %q: %w", pkgDir, err)
		}
		for _, e := range entries {
			linkPath := filepath.Join(r.Target, e.Name())
			linkDest := filepath.Join(pkgLinkDest, e.Name())
			if r.DryRun {
				fmt.Fprintln(os.Stdout, r.FS.Join(linkPath), "-->", linkDest)
				continue
			}
			err := r.FS.MkdirAll(filepath.Dir(linkPath), 0o755)
			if err != nil {
				return fmt.Errorf("making link parent: %w", err)
			}
			err = r.FS.Symlink(linkDest, linkPath)
			if err != nil {
				return fmt.Errorf("making link: %w", err)
			}
		}

	}
	return nil
}
