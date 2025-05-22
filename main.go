package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dhemery/duffel/internal/files"
)

func main() {
	run(os.Args[1:])
}

type duffelFS interface {
	ReadDir(path string) ([]fs.DirEntry, error)
	Lstat(path string) (fs.FileInfo, error)
	MkdirAll(path string, perm fs.FileMode) error
	Symlink(old, new string) error
}

type request struct {
	fsys   duffelFS
	source string
	target string
	pkgs   []string
}

func run(args []string) error {
	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	source := flags.String("source", ".", "The source `dir`")
	target := flags.String("target", "..", "The target `dir`")

	flags.Parse(args)
	root := "/"

	fsys := files.DirFS(root)

	absTarget, err := filepath.Abs(*target)
	if err != nil {
		return fmt.Errorf("making target absolute: %w", err)
	}
	targetPath, _ := filepath.Rel(root, absTarget)

	absSource, err := filepath.Abs(*source)
	if err != nil {
		return fmt.Errorf("making source absolute: %w", err)
	}
	sourcePath, _ := filepath.Rel(root, absSource)

	req := request{
		fsys:   fsys,
		source: sourcePath,
		target: targetPath,
		pkgs:   flags.Args(),
	}
	return install(req)
}

func install(r request) error {
	sourceLinkDest, err := filepath.Rel(r.target, r.source)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}

	for _, pkg := range r.pkgs {
		pkgDir := filepath.Join(r.source, pkg)
		pkgLinkDest := filepath.Join(sourceLinkDest, pkg)
		entries, err := r.fsys.ReadDir(pkgDir)
		if err != nil {
			return fmt.Errorf("reading entries of %q: %w", pkgDir, err)
		}
		for _, e := range entries {
			linkPath := filepath.Join(r.target, e.Name())
			linkDest := filepath.Join(pkgLinkDest, e.Name())
			err := r.fsys.MkdirAll(filepath.Dir(linkPath), 0o755)
			if err != nil {
				return fmt.Errorf("making link parent: %w", err)
			}
			err = r.fsys.Symlink(linkDest, linkPath)
			if err != nil {
				return fmt.Errorf("making link: %w", err)
			}
		}

	}
	return nil
}
