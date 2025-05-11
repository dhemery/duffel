package cmd

const unlinkDescription = `
duffel unlink removes links in the target directory that point to
corresponding items within the named packages.
`

var Unlink = Command{
	Name:        "unlink",
	ArgList:     "pkg...",
	Summary:     "Remove links to packages",
	Description: unlinkDescription,
	Flags:       linkFlags,
}

func init() {
	Unlink.Run = runUnlink
}

func runUnlink(c *Command, args []string) error {
	return nil
}
