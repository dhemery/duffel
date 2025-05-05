package main

import (
	"flag"
	"fmt"
	"os"

	"dhemery.com/duffel/cmd"
)

func init() {
	cmd.Commands = []*cmd.Command{
		cmd.CmdLink,
		cmd.CmdUnlink,
		cmd.CmdHelp,
	}
}

func main() {
	flag.Usage = cmd.Usage
	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(2)
	}

	cmdName := args[0]
	cmd, ok := cmd.FindCommand(cmdName)
	if !ok {
		fmt.Fprintln(os.Stderr, "no such command:", cmdName)
		flag.Usage()
		os.Exit(2)
	}

	err := cmd.Run(cmd, args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
