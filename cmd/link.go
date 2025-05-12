package cmd

import (
	"flag"
)

const linkDescription = `
DESCRIPTION

duffel link creates links in the target directory that point to
corresponding items in the named packages.

The default target directory is the parent of the current
working directory. To specify a different target directory, use
the -target option.

duffel looks for the named packages in the source directory. The
default source directory is the current working directory. To
specify a different source directory, use the -source option.

duffel link evaluates all planned actions before performing any.
If any planned action is invalid, duffel link prints an error
message and exits without performing any actions. Use the -plan
option to preview the plan.
`

var (
	Link = Command{
		Name:        "link",
		Run: runLink,
		UsageLine:   "duffel link [options] package...",
		Summary:     "Create links to package items",
		Description: linkDescription,
		Flags:       flag.NewFlagSet("", flag.ExitOnError),
	}

	onlyPlan  *bool
	sourceDir *string
	targetDir *string
	verbose   *bool
)

func init() {
	onlyPlan = Link.Flags.Bool("plan", true, "Print planned actions without executing them.")
	sourceDir = Link.Flags.String("source", ".", "Set source directory to `dir`.")
	targetDir = Link.Flags.String("target", "..", "Set target directory to `dir`.")
	verbose = Link.Flags.Bool("verbose", false, "Print each action before executing it.")
}

func runLink(c *Command, args []string) error {
	return nil
}
