package cmd

import (
	"flag"
)

const uninstallDescription = `
DESCRIPTION

'duffel uninstall' removes items in the target directory that
correspond to items within the named packages.
`

var (
	Uninstall = Command{
		Name:        "uninstall",
		Run:         runUninstall,
		UsageLine:   "duffel uninstall [options] package...",
		Summary:     "Uninstall packages",
		Description: uninstallDescription,
		Flags:       flag.NewFlagSet("", flag.ExitOnError),
	}

	uninstallSourceDir = Uninstall.Flags.String("source", ".", "Find packages in `dir`.")
	uninstallTargetDir = Uninstall.Flags.String("target", "..", "Uninstall packages from `dir`.")
	uninstallDryRun    = Uninstall.Flags.Bool("n", false, "Print planned actions but do not execute them.")
)

func runUninstall(args []string) {
}
