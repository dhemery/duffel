package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	run(os.Args[1:])
}

func run(args []string) error {
	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	source := flags.String("source", ".", "The source `dir`")
	target := flags.String("target", "..", "The target `dir`")

	flags.Parse(args)

	targetDir, err := filepath.Abs(*target)
	if err != nil {
		return fmt.Errorf("making target absolute: %w", err)
	}

	sourceDir, err := filepath.Abs(*source)
	if err != nil {
		return fmt.Errorf("making source absolute: %w", err)
	}

	sourceLinkDest, err := filepath.Rel(targetDir, sourceDir)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}

	pkgs := flags.Args()

	for _, pkg := range pkgs {
		pkgDir := filepath.Join(sourceDir, pkg)
		pkgLinkDest := filepath.Join(sourceLinkDest, pkg)
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			return fmt.Errorf("reading entries of %q: %w", pkgDir, err)
		}
		for _, e := range entries {
			linkPath := filepath.Join(targetDir, e.Name())
			linkDest := filepath.Join(pkgLinkDest, e.Name())
			err := os.MkdirAll(filepath.Dir(linkPath), 0o755)
			if err != nil {
				return fmt.Errorf("making link parent: %w", err)
			}
			err = os.Symlink(linkDest, linkPath)
			if err != nil {
				return fmt.Errorf("making link: %w", err)
			}
		}

	}
	return nil
}
