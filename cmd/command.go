package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	Commands = []*Command{}
	CommandsByName = map[string]*Command{}
)

func init() {
	addCommand(&Link)
	addCommand(&Unlink)
	addCommand(&Help)
}

type Command struct {
	Name            string
	Run             func(cmd *Command, args []string) error
	ArgList         string
	Summary         string
	FullDescription string
	Flags           *flag.FlagSet
}

func addCommand(c *Command) {
	Commands = append(Commands, c)
	CommandsByName[c.Name] = c
}

func Execute() {
	if len(os.Args) < 2 {
		Usage()
		os.Exit(2)
	}

	cmdName := os.Args[1]
	cmd, ok := CommandsByName[cmdName]
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
	fmt.Fprintln(os.Stderr, strings.TrimSpace(cmd.FullDescription))
}

