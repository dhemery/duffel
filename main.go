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

	fmt.Println("Dry run:", *dryRun)
	req := duffel.Request{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		FS:     fsys,
		Source: sourcePath,
		Target: targetPath,
		Pkgs:   flags.Args(),
		DryRun: *dryRun,
	}

	err = duffel.Install(req)
	if err != nil {
		fatal(fmt.Errorf("installing: %w", err))
	}
}

func fatal(err error) {
	fmt.Fprint(os.Stderr, err)
	os.Exit(1)
}
