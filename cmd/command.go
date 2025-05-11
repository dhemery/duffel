package cmd

import (
	"flag"
	"fmt"
	"io"
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

func (c *Command) usageLine() string {
	opts := ""
	var name = c.Name
	if name != "duffel" {
		name = "duffel " + c.Name
	}
	if c.Flags != nil {
		opts = " [options]"
	}
	return fmt.Sprintf("%s%s %s", name, opts, c.ArgList)
}

func (c *Command) Usage() {
	fmt.Fprintln(os.Stderr, "usage:", c.usageLine())
	fmt.Fprintf(os.Stderr, "Run 'duffel help %s' for details.\n", c.Name)
}

func (cmd *Command) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w)
	fmt.Fprintln(w, " ", cmd.usageLine())

	if cmd.Flags != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "OPTIONS")
		fmt.Fprintln(w)
		cmd.Flags.SetOutput(w)
		cmd.Flags.PrintDefaults()
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "DESCRIPTION")
	fmt.Fprintln(w)
	fmt.Fprintln(w, strings.TrimSpace(cmd.Description))
}
