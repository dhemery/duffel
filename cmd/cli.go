package cmd

import (
	"flag"
	"fmt"
	"os"
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
	flags.Usage = cmd.PrintHelp
	flags.Parse(args[2:])

	err := cmd.Run(cmd, flags.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
