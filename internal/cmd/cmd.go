// Package cmd constructs and executes a plan
// to satisfy the user's request.
package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/dhemery/duffel/internal/file"
)

// FS is an [fs.FS] that implements all of the methods used by duffel.
type FS interface {
	fs.ReadLinkFS
	file.ActionFS
}

// Execute performs the duffel operations requested by args.
func Execute(args []string, fsys FS, cwd string, wout, werr io.Writer) {
	opts, args, err := parseArgs(args, werr)
	if err != nil {
		fatalUsage(werr, err)
	}

	cmd, err := newCommand(opts, args, fsys, cwd[1:], wout, werr)
	if err != nil {
		fatalUsage(werr, err)
	}

	if err := cmd.execute(); err != nil {
		fatal(werr, err)
	}
}

func fatal(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(1)
}

func fatalUsage(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(2)
}
