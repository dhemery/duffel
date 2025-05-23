package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dhemery/duffel/internal/duffel"
	"github.com/dhemery/duffel/internal/files"
)

func main() {
	run(os.Args[1:])
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

	req := duffel.Request{
		FS:     fsys,
		Source: sourcePath,
		Target: targetPath,
		Pkgs:   flags.Args(),
	}
	return duffel.Install(req)
}
