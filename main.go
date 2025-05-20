package main

import (
	"os"
)

func main() {
}

func run(_ []string) {
	os.Symlink("source/pkg/pkgItem", "../pkgItem")
}
