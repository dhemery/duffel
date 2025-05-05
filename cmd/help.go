package cmd

import (
	"fmt"
	"os"
)

var CmdHelp = &Command{
	Name:            "help",
	Run:             runHelp,
	ArgList:         "command",
	Summary:         "Show help for a command",
	FullDescription: "Show help for a command",
	Flags:           nil,
}

func Usage() {
	fmt.Fprintln(os.Stderr, "Maintain dotfiles")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "USAGE")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  duffel <command> [arguments]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "COMMANDS")
	fmt.Fprintln(os.Stderr)
	for _, cmd := range Commands {
		fmt.Fprintf(os.Stderr, "  %-8s %s\n", cmd.Name, cmd.Summary)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, `Use "duffel help <command>" for more information`)
}

func runHelp(cmd *Command, args []string) error {
	if len(args) == 0 {
		Usage()
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("too many commands: %s", args)
	}

	targetName := args[0]
	targetCommand, ok := FindCommand(targetName)
	if !ok {
		return fmt.Errorf("no such command: %s", targetName)
	}

	targetCommand.Usage()
	fmt.Fprintln(os.Stderr)
	if targetCommand.Flags != nil {
		fmt.Fprintln(os.Stderr, "OPTIONS")
		fmt.Fprintln(os.Stderr)
		targetCommand.Flags.PrintDefaults()
	}
	targetCommand.Help()

	return nil
}
