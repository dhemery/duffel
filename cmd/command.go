package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Command struct {
	Name        string
	Run         func(cmd *Command, args []string) error
	ArgList     string
	Summary     string
	Description string
	Flags       *flag.FlagSet
}

func (cmd *Command) PrintHelp() {
	fmt.Fprintln(os.Stderr, "USAGE")
	fmt.Fprintln(os.Stderr)
	if cmd.Flags != nil {
		fmt.Fprintln(os.Stderr, "  duffel", cmd.Name, "[options]", cmd.ArgList)
	} else {
		fmt.Fprintln(os.Stderr, "  duffel", cmd.Name, cmd.ArgList)
	}
	fmt.Fprintln(os.Stderr)

	if cmd.Flags != nil {
		fmt.Fprintln(os.Stderr, "OPTIONS")
		fmt.Fprintln(os.Stderr)
		cmd.Flags.PrintDefaults()
	}
}

func (cmd *Command) PrintExtraHelp() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "DESCRIPTION")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, strings.TrimSpace(cmd.Description))
}

func (cmd *Command) Usage() {
	cmd.PrintHelp()
	cmd.PrintExtraHelp()
}
