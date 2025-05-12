package main

import (
	"fmt"
	"os"
	"strings"

	"dhemery.com/duffel/cmd"
)

var (
	Commands = []*cmd.Command{
		&cmd.Link,
		&cmd.Unlink,
	}
	CmdDuffel = cmd.Command{
		Name:        "duffel",
		UsageLine:   "duffel <command> [arguments]",
		Summary:     "Maintain dotfiles",
		Description: duffelDescription(),
	}
)

func main() {
	if len(os.Args) < 2 {
		CmdDuffel.PrintHelp(os.Stderr)
		os.Exit(2)
	}

	cmdName := os.Args[1]
	if cmdName == "help" {
		showHelp(os.Args[2:])
	}

	c := lookup(cmdName)
	if c == nil {
		fmt.Fprintf(os.Stderr, "duffel %s: no such command\n", cmdName)
		fmt.Fprintln(os.Stderr, "Run 'duffel help' for usage.")
		os.Exit(2)
	}

	flags := c.Flags
	flags.Usage = c.Usage
	flags.Parse(os.Args[2:])

	c.Run(flags.Args())
}

func lookup(name string) *cmd.Command {
	for _, c := range Commands {
		if c.Name == name {
			return c
		}
	}
	return nil
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
	c := lookup(name)
	if c == nil {
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

func duffelDescription() string {
	b := &strings.Builder{}
	fmt.Fprintln(b, "COMMANDS")
	fmt.Fprintln(b)
	for _, c := range Commands {
		fmt.Fprintf(b, "  %-8s %s\n", c.Name, c.Summary)
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, `Run 'duffel help <command>' for more information`)
	return b.String()
}
