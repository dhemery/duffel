package main

import (
	"fmt"
	"os"

	"github.com/dhemery/duffel/internal/cmd"
)

func main() {
	r, err := os.OpenRoot("/")
	if err != nil {
		fatal(err)
	}
	if err = cmd.Execute(r); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)

}
