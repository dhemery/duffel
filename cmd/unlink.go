package cmd

const unlinkDescription = `
duffel unlink removes links in the target directory that point to
corresponding items within the named packages.
`

var (
	CmdUnlink = &Command{
		Name:            "unlink",
		Run:             runUnlink,
		ArgList:         "pkg...",
		Summary:         "Remove links to packages",
		FullDescription: unlinkDescription,
		Flags:           linkFlags,
	}
)

func runUnlink(cmd *Command, args []string) error {
	return nil
}
