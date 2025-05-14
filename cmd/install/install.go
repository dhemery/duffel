package install

import (
	"flag"
	"log"
	"path/filepath"

	"dhemery.com/duffel/cmd"
	"dhemery.com/duffel/plan"
)

const installDescription = `
DESCRIPTION

'duffel install' installs the named packages by creating links
in the target directory that point to items in the packages.

The default target directory is the parent of the current
working directory. To specify a different target directory, use
the -target option.

duffel looks for the named packages in the source directory. The
default source directory is the current working directory. To
specify a different source directory, use the -source option.

duffel install evaluates all planned actions before performing
any. If any planned action is invalid, duffel prints an error
message and exits without performing any actions. Use the -n
option to print the planned actions without executing them.
`

var (
	Cmd = cmd.Command{
		Name:        "install",
		Run:         runInstall,
		UsageLine:   "duffel install [options] package...",
		Summary:     "Install packages",
		Description: installDescription,
		Flags:       flag.NewFlagSet("", flag.ExitOnError),
	}

	duffelDir string
	targetDir string
	dryRun    bool
)

func init() {
	Cmd.Flags.StringVar(&duffelDir, "source", ".", "Find packages in `dir`.")
	Cmd.Flags.StringVar(&targetDir, "target", "..", "Install packages into `dir`.")
	Cmd.Flags.BoolVar(&dryRun, "n", false, "Print planned actions but do not execute them.")
}

func runInstall(packages []string) {
	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		log.Fatal(err)
	}

	duffelDir, err = filepath.Abs(duffelDir)
	if err != nil {
		log.Fatal(err)
	}
	duffelDir, err = filepath.Rel(targetDir, duffelDir)
	if err != nil {
		log.Fatal(err)
	}
	targetFS := plan.NewDirFS(targetDir)
	duffelFS := plan.NewDirFS(duffelDir)

	plan.Plan(targetFS, duffelFS, nil, packages)
}
