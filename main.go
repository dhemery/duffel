package main

import (
	"dhemery.com/duffel/cmd"
)

func init() {
	cmd.Commands = []*cmd.Command{
		&cmd.Link,
		&cmd.Unlink,
		&cmd.Help,
	}
}

func main() {
	cmd.Execute()
}


