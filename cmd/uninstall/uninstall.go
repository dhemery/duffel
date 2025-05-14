package uninstall

import (
	"flag"

	"dhemery.com/duffel/cmd"
)

const uninstallDescription = `
DESCRIPTION

'duffel uninstall' removes items in the target directory that
correspond to items within the named packages.
`

var (
	Cmd = cmd.Command{
		Name:        "uninstall",
		Run:         runUninstall,
		UsageLine:   "duffel uninstall [options] package...",
		Summary:     "Uninstall packages",
		Description: uninstallDescription,
		Flags:       flag.NewFlagSet("", flag.ExitOnError),
	}

	duffelDir = Cmd.Flags.String("source", ".", "Find packages in `dir`.")
	targetDir = Cmd.Flags.String("target", "..", "Uninstall packages from `dir`.")
	dryRun    = Cmd.Flags.Bool("n", false, "Print planned actions but do not execute them.")
)

func runUninstall(args []string) {
}
