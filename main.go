package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dhemery/duffel/internal/exec"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
)

func main() {
	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	dryRunOpt := flags.Bool("n", false, "Print planned actions without executing them.")
	sourceOpt := flags.String("source", ".", "The source `dir`")
	targetOpt := flags.String("target", "..", "The target `dir`")
	logOpt := flags.String("log", "warn", "Log level (error|warn|info|debug|trace|none)")

	flags.Parse(os.Args[1:])

	if err := log.SetByName(*logOpt, os.Stderr); err != nil {
		fatal(err)
	}

	root := "/"
	fsys := file.DirFS(root)

	absTarget, err := filepath.Abs(*targetOpt)
	if err != nil {
		fatal(fmt.Errorf("target: %w", err))
	}
	target, _ := filepath.Rel(root, absTarget)

	absSource, err := filepath.Abs(*sourceOpt)
	if err != nil {
		fatal(fmt.Errorf("source: %w", err))
	}
	source, _ := filepath.Rel(root, absSource)

	req := &exec.Request{
		FS:     fsys,
		Source: source,
		Target: target,
		Pkgs:   flags.Args(),
	}

	err = exec.Execute(req, *dryRunOpt, os.Stdout)
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	log.Error(err.Error())
	os.Exit(1)
}
