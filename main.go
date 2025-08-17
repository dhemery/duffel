package main

import (
	"os"

	"github.com/dhemery/duffel/internal/cmd"
	"github.com/dhemery/duffel/internal/file"
)

func main() {
	cmd.Execute(os.Args[1:], makeFS, os.Stdout, os.Stderr)
}

// MakeFS returns a [cmd.FS] rooted at root.
func makeFS(root string) (cmd.FS, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	return file.RootFS(r), nil
}
