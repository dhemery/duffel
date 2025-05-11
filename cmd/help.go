package cmd

import (
	"fmt"
	"os"
)

var Help = Command{
	Name:        "help",
	ArgList:     "<command>",
	Summary:     "Show help for a command",
	Description: "Show help for a command",
}

func init() {
	Help.Run = runHelp
}

func runHelp(c *Command, args []string) error {
	if len(args) == 0 {
		PrintHelp(os.Stdout)
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

	cmd.PrintHelp(os.Stdout)

	return nil
}
