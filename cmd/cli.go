package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	Commands = []*Command{
		&Link,
		&Unlink,
	}
	CommandsByName = map[string]*Command{}
	CmdDuffel      = Command{
		Name:    "duffel",
		ArgList: "<command> [arguments]",
		Summary: "Maintain dotfiles",
	}
)

func init() {
	for _, c := range Commands {
		CommandsByName[c.Name] = c
	}
	b := &strings.Builder{}
	fmt.Fprintln(b, "COMMANDS")
	fmt.Fprintln(b)
	for _, c := range Commands {
		fmt.Fprintf(b, "  %-8s %s\n", c.Name, c.Summary)
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, `Run 'duffel help <command>' for more information`)
	CmdDuffel.Description = b.String()
}

func PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "Maintain dotfiles")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  duffel <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "COMMANDS")
	fmt.Fprintln(w)
	for _, c := range Commands {
		fmt.Fprintf(w, "  %-8s %s\n", c.Name, c.Summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'duffel help <command>' for more information about a command.")
}

func Execute(args []string) {
	if len(args) < 2 {
		CmdDuffel.PrintHelp(os.Stderr)
		os.Exit(2)
	}

	cmdName := args[1]
	if cmdName == "help" {
		showHelp(args[2:])
	}

	cmd, ok := CommandsByName[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "duffel %s: no such command\n", cmdName)
		fmt.Fprintln(os.Stderr, "Run 'duffel help' for usage.")
		os.Exit(2)
	}

	flags := cmd.Flags
	if flags == nil {
		flags = flag.NewFlagSet("duffel "+cmd.Name, flag.ExitOnError)
	}
	flags.Usage = cmd.Usage
	flags.Parse(args[1:])

	err := cmd.Run(cmd, flags.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func showHelp(topics []string) {
	switch len(topics) {
	case 0:
		CmdDuffel.PrintHelp(os.Stdout)
		os.Exit(0)
	case 1:
		showCommandHelp(topics[0])
	default:
		badHelpRequest(strings.Join(topics, " "))
	}
}

func showCommandHelp(name string) {
	c, ok := CommandsByName[name]
	if !ok {
		badHelpRequest(name)
	}

	c.PrintHelp(os.Stdout)
	os.Exit(0)
}

func badHelpRequest(topic string) {
	fmt.Fprintf(os.Stderr, "duffel help %s: unknown help topic\n", topic)
	fmt.Fprintln(os.Stderr, "Run 'duffel help'.")
	os.Exit(2)
}
