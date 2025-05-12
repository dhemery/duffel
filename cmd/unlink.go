package cmd

const unlinkDescription = `
DESCRIPTION

duffel unlink removes links in the target directory that point to
corresponding items within the named packages.
`

var Unlink = Command{
	Name:        "unlink",
	Run:         runUnlink,
	UsageLine:   "duffel unlink [options] package...",
	Summary:     "Remove links to package items",
	Description: unlinkDescription,
	Flags:       Link.Flags, // Same flags as LinkCmd for now
}

func runUnlink(args []string) {
}
