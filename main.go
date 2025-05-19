package main

import (
	"os"
)

func run(_ []string) {
	os.Symlink("source/pkg/pkgItem", "../pkgItem")
}
