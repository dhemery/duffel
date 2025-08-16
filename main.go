package main

import (
	"fmt"
	"os"

	"github.com/dhemery/duffel/internal/cmd"
	"github.com/dhemery/duffel/internal/file"
)

func main() {
	r, err := os.OpenRoot("/")
	if err != nil {
		fatal(err)
	}
	fsys := file.RootFS(r)

	if err = cmd.Execute(os.Args[1:], fsys, os.Stdout, os.Stderr); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)

}
