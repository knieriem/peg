package peg

import (
	"strings"
)

const (
	allOptimizations = "1:l:p:r:s"
)

type optiFlags struct {
	peek               bool
	elimRestore        bool
	inlineLeafs        bool
	seqPeekNot         bool
	unorderedFirstItem bool
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
		case '1':
			o.unorderedFirstItem = true
		case 'p':
			o.peek = true
		case 'r':
			o.elimRestore = true
		case 'l':
			o.inlineLeafs = true
		case 's':
			o.seqPeekNot = true
		}
	}
	return
}
