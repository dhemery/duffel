package cmd

import (
	"fmt"
)

var Help = Command{
	Name:        "help",
	Run:         runHelp,
	ArgList:     "<command>",
	Summary:     "Show help for a command",
	Description: "Show help for a command",
}

func runHelp(args []string) error {
	if len(args) == 0 {
		Usage()
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("too many commands: %s", args)
	}

	cmdName := args[0]
	cmd, ok := CommandsByName[cmdName]
	if !ok {
		return fmt.Errorf("no such command: %s", cmdName)
	}

	cmd.Help()
	cmd.ExtraHelp()

	return nil
}
