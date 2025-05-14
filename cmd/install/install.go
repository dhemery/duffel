package install

import (
	"flag"
	"log"
	"path/filepath"

	"dhemery.com/duffel/cmd"
	"dhemery.com/duffel/cmd/preview"
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

	sourceOpt string
	targetOpt string
	dryRunOpt bool
)

func init() {
	Cmd.Flags.StringVar(&sourceOpt, "source", ".", "Find packages in `dir`.")
	Cmd.Flags.StringVar(&targetOpt, "target", "..", "Install packages into `dir`.")
	Cmd.Flags.BoolVar(&dryRunOpt, "n", false, "Print planned actions but do not execute them.")
}

func runInstall(packages []string) {
	absTarget, err := filepath.Abs(targetOpt)
	if err != nil {
		log.Fatal(err)
	}

	absSource, err := filepath.Abs(sourceOpt)
	if err != nil {
		log.Fatal(err)
	}
	linkPrefix, err := filepath.Rel(absTarget, absSource)
	if err != nil {
		log.Fatal(err)
	}

	targetFS := preview.DirFS(targetOpt).(plan.FS)
	duffelFS := preview.DirFS(sourceOpt).(plan.FS)

	advisor := plan.NewInstallAdvisor(duffelFS, targetFS, linkPrefix)

	plan.Build(duffelFS, advisor, packages)
}
