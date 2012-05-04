This is a modified version of Go package [peg][], written by
[pointlander](https://github.com/pointlander), for the purpose
of supporting LEG grammars, a variant of PEG grammars as defined
in [peg(1)][]. See README.orig for the original README.  
If you do not intend to use a LEG grammar, please take a look
at the [original package][peg] instead.

The subdirectory *cmd/leg* contains source files for the LEG
parser. Using this parser, the [peg-markdown][] package,
which contains a LEG definition, has been ported to Go.

To download and install, run

	go get github.com/knieriem/peg

Run `make` or `make prepare` to bootstrap the peg parser,
and to create the leg parser and the example parsers. There
should be a binary `peg` in *./cmd/peg* now.

To delete the generated source files and binaries that are
not part of the project, run `make clean`.

The desk calculator example from [peg(1)][] can be built by
typing `go build` in directory *./cmd/legcalc*.

The parser generators now take on option -O to turn on various
optimizations, with a single argument consisting either of a
number of colon-separated flags, or the string "all".
For the possible values of these flags, see [util.go](util.go).


### Summary of other modifications:

*	AddPackage, AddPeg and AddState methods have been
	replaced by a new method AddDefine, which stores
	different values into a map. This way additional strings
	(like yystype) can easily be specified.

*	As [markdown_parser.leg][] makes heavy use of yytext,
	I replaced the action arguments `buffer string,
	begin, end int` by `yytext string`, which equals to
	`buffer[begin:end]`.

*	Headers `%{ ... %}` and a Trailer `%% ...`, which are
	used in LEG Grammers, are supported by new methods
	AddHeader, and AddTrailer. (Both probably could
	be replaced by AddDefine ...)
	
*	Parse() has got an integer argument `ruleId', to
	allow rules different from rule 0 to be applied, as
	it is needed by peg-markdown. The output file now also
	contains a const block containing names and IDs of all
	rules. Defined but unused rules are not deleted anymore
	(the warning has been preserved), because they might
	be called directly.

*	Added support for semantic values as described in
	[peg(1)][]. Results of sub-rules can be referred
	to from within actions, whereas `$$` can be used to
	store the current rule's return value. At the moment this
	only works without the `-inline` option.

*	Added *ResetBuffer* closure to parser. The user can set
	a new buffer to be processed, the remaining part of the
	old buffer is returned. This way a parser can be reused
	without calling *Init* again. See [./leg/calc.leg](./leg/calc.leg)
	for an example.


[peg]: https://github.com/pointlander/peg
[peg(1)]: http://piumarta.com/software/peg/peg.1.html
[peg-markdown]: https://github.com/jgm/peg-markdown
[markdown_parser.leg]: https://github.com/jgm/peg-markdown/blob/master/markdown_parser.leg#L57

--  
Michael Teichgräber
