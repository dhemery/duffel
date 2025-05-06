package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	Commands       = []*Command{}
	CommandsByName = map[string]*Command{}
)

func addCommand(c *Command) {
	Commands = append(Commands, c)
	CommandsByName[c.Name] = c
}

func init() {
	addCommand(&Link)
	addCommand(&Unlink)
	addCommand(&Help)
}

type Command struct {
	Name        string
	Run         func(args []string) error
	ArgList     string
	Summary     string
	Description string
	Flags       *flag.FlagSet
}

func (cmd *Command) Help() {
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

func (cmd *Command) ExtraHelp() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "DESCRIPTION")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, strings.TrimSpace(cmd.Description))
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

func Execute() {
	args := os.Args
	if len(args) < 2 {
		Usage()
		os.Exit(2)
	}

	cmdName := args[1]
	cmd, ok := CommandsByName[cmdName]
	if !ok {
		fmt.Fprintln(os.Stderr, "no such command:", cmdName)
		Usage()
		os.Exit(2)
	}

	flags := cmd.Flags
	if flags == nil {
		flags = flag.NewFlagSet("", flag.ExitOnError)
	}
	flags.Usage = cmd.Help
	flags.Parse(args[2:])

	err := cmd.Run(flags.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
