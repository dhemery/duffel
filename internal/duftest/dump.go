package duftest

import (
	"flag"
	"fmt"
	"strings"
	"testing"
)

var DumpOpts string

func init() {
	flag.StringVar(&DumpOpts, "dump", "lf", "print extra test details")
}

func Dump(t *testing.T, name string, val fmt.Stringer) {
	t.Helper()
	tag := strings.ToLower(name[:1])
	forceTag := strings.ToUpper(tag)
	forceDump := strings.Contains(DumpOpts, forceTag)
	dump := forceDump || strings.Contains(DumpOpts, tag)
	if !dump {
		return
	}
	if forceDump && !t.Failed() {
		t.Error("Failure forced by -dump", forceTag)
	}
	t.Logf("%s:\n%s\n", name, val)
}
