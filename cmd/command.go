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
	name := c.Name
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

func (c *Command) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, c.Summary)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w)
	fmt.Fprintln(w, " ", c.usageLine())

	if c.Flags != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "OPTIONS")
		fmt.Fprintln(w)
		c.Flags.SetOutput(w)
		c.Flags.PrintDefaults()
	}
	if c.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, strings.TrimSpace(c.Description))
	}
}
