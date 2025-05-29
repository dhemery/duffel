package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dhemery/duffel/internal/duffel"
	"github.com/dhemery/duffel/internal/files"
)

func main() {
	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	dryRun := flags.Bool("n", false, "Print planned actions without executing them.")
	source := flags.String("source", ".", "The source `dir`")
	target := flags.String("target", "..", "The target `dir`")

	flags.Parse(os.Args[1:])
	root := "/"

	fsys := files.DirFS(root)

	absTarget, err := filepath.Abs(*target)
	if err != nil {
		fatal(fmt.Errorf("making target absolute: %w", err))
	}
	targetPath, _ := filepath.Rel(root, absTarget)

	absSource, err := filepath.Abs(*source)
	if err != nil {
		fatal(fmt.Errorf("making source absolute: %w", err))
	}
	sourcePath, _ := filepath.Rel(root, absSource)

	req := &duffel.Request{
		Stdout: os.Stdout,
		FS:     fsys,
		Source: sourcePath,
		Target: targetPath,
		Pkgs:   flags.Args(),
		DryRun: *dryRun,
	}

	err = duffel.Execute(req)
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}
