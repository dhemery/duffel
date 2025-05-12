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
	Run         func(args []string)
	UsageLine   string
	Summary     string
	Description string
	Flags       *flag.FlagSet
}

func (c *Command) Usage() {
	fmt.Fprintln(os.Stderr, "usage:", c.UsageLine)
	fmt.Fprintf(os.Stderr, "Run 'duffel help %s' for details.\n", c.Name)
}

func (c *Command) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, c.Summary)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w)
	fmt.Fprintln(w, " ", c.UsageLine)

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
