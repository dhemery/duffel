package cmd

import "flag"

const uninstallDescription = `
DESCRIPTION

'duffel uninstall' removes items in the target directory that
correspond to items within the named packages.
`

var Uninstall = Command{
	Name:        "uninstall",
	Run:         runUninstall,
	UsageLine:   "duffel uninstall [options] package...",
	Summary:     "Uninstall packages",
	Description: uninstallDescription,
	Flags:       flag.NewFlagSet("", flag.ExitOnError),
}

func init() {
	Uninstall.Flags.StringVar(&Config.DuffelDir, "source", ".", "Find packages in `dir`.")
	Uninstall.Flags.StringVar(&Config.TargetDir, "target", "..", "Uninstall packages from `dir`.")
	Uninstall.Flags.BoolVar(&Config.DryRun, "n", false, "Print planned actions but do not execute them.")
}

func runUninstall(args []string) {
}
