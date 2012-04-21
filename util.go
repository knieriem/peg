package peg

import (
	"strings"
)

const (
	allOptimizations = "l:p:r"
)

type optiFlags struct {
	peek        bool
	elimRestore bool
	inlineLeafs bool
}

func parseOptiFlags(flags string) (o *optiFlags) {
	o = new(optiFlags)
	if flags == "all" {
		flags = allOptimizations
	}
	for _, f := range strings.Split(flags, ":") {
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case 'p':
			o.peek = true
		case 'r':
			o.elimRestore = true
		case 'l':
			o.inlineLeafs = true
		}
	}
	return
}
