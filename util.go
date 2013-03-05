package peg

import (
	"strings"
)

/*
Possible optimizations:

	1	Thanks to the `switch' optimization, for the
		first item of `case' sections it is already known what the
		first character is. Otherwise the case branch would not have
		been entered. This patch makes use of this information and
		avoids testing for the same conditions again.

	l	Inline leaf rules, if they contain only one element of Dot, Char,
		Class or Predicate type, or such an element embedded in a
		expression out of + * ? ! &.

	(p)	When doing a peek for Dot, Char, Class, and Predicate,
		don't modify position so that it doesn't have to be restored.

	r	If possible, try to avoid saving and restoring of
		positions. For each rule two pairs of flags will be tracked,
		which tell whether the reading position, or action thunk
		position might advance in case the rule matches, and in case
		it does not.

	s	if a sequence starts with one or more `!Char'
		(PeekNot for Character), insert a switch expression

Flags that are shown within braces are less effective now than they used
to be, probably because of improvements of the Go compilers.
*/
const (
	AllOptimizations = "1:l:p:r:s"
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
		flags = AllOptimizations
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
