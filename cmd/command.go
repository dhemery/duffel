package cmd

import (
	"flag"
	"fmt"
	"os"
)

var (
	Commands = []*Command{}
)

type Command struct {
	Name            string
	Run             func(cmd *Command, args []string) error
	ArgList         string
	Summary         string
	FullDescription string
	Flags           *flag.FlagSet
}

func Execute() {
	if len(os.Args) < 2 {
		Usage()
		os.Exit(2)
	}

	cmdName := os.Args[1]
	cmd, ok := FindCommand(cmdName)
	if !ok {
		fmt.Fprintln(os.Stderr, "no such command:", cmdName)
		Usage()
		os.Exit(2)
	}

	err := cmd.Run(cmd, os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func (cmd *Command) Usage() {
	fmt.Fprintln(os.Stderr, "USAGE")
	fmt.Fprintln(os.Stderr)
	if cmd.Flags != nil {
		fmt.Fprintln(os.Stderr, "  duffel", cmd.Name, "[options]", cmd.ArgList)
	} else {
		fmt.Fprintln(os.Stderr, "  duffel", cmd.Name, cmd.ArgList)
	}
}

func (cmd *Command) Help() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "DESCRIPTION")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, cmd.FullDescription)
}

func FindCommand(name string) (*Command, bool) {
	for _, cmd := range Commands {
		if cmd.Name == name {
			return cmd, true
		}
	}
	return nil, false
}
