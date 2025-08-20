package main

import (
	"fmt"
	"os"

	"github.com/dhemery/duffel/internal/cmd"
	"github.com/dhemery/duffel/internal/file"
)

func main() {
	root, err := os.OpenRoot("/")
	if err != nil {
		fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	fsys := file.RootFS(root)

	cmd.Execute(os.Args[1:], fsys, cwd, os.Stdout, os.Stderr)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
