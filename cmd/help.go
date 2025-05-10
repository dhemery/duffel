package cmd

import (
	"fmt"
	"os"
)

var Help = Command{
	Name:        "help",
	Run:         runHelp,
	ArgList:     "<command>",
	Summary:     "Show help for a command",
	Description: "Show help for a command",
}

func runHelp(c *Command, args []string) error {
	if len(args) == 0 {
		Usage(os.Stdout)
		os.Exit(0)
	}

	if len(args) != 1 {
		return fmt.Errorf("too many commands: %s", args)
	}

	cmdName := args[0]
	cmd, ok := CommandsByName[cmdName]
	if !ok {
		return fmt.Errorf("no such command: %s", cmdName)
	}

	cmd.Usage()

	return nil
}
