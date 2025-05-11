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
		&Help,
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
	fmt.Fprintln(w, `Run 'duffel help <command>' for more information`)
}

func Execute(args []string) {
	if len(args) < 1 {
		CmdDuffel.PrintHelp(os.Stderr)
		os.Exit(2)
	}

	cmdName := args[0]
	cmd, ok := CommandsByName[cmdName]
	if !ok {
		fmt.Fprintln(os.Stderr, "no such command:", cmdName)
		PrintHelp(os.Stderr)
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
