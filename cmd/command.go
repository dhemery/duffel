package cmd

import (
	"flag"
	"fmt"
	"os"
)

var (
	Commands = []*Command{}
)

const ()

type Command struct {
	Name            string
	Run             func(cmd *Command, args []string) error
	ArgList         string
	Summary         string
	FullDescription string
	Flags           *flag.FlagSet
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
