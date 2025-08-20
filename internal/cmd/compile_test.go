package cmd_test

import (
	"testing"

	"github.com/dhemery/duffel/internal/cmd"
	"github.com/dhemery/duffel/internal/errfs"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		desc    string
		files   map[string]*errfs.File
		opts    cmd.Options
		args    []string
		wantErr error
	}{{
		desc: "absolute target",
	}}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Log(test.desc)
		})
	}
}
