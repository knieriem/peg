This is a modified version of Go package [peg][], written by
[pointlander](https://github.com/pointlander), for the purpose
of supporting LE Grammars, as defined in [peg(1)][]. See
README.orig for the original README.

The new sub directory `leg` contains source files for the
LEG parser. Using this parser, the [peg-markdown][] package,
which contains a LEG definition, could be ported to Go. The
desk calculator example from [peg(1)][] can be built by typing
`make calc` in directory `leg`.

### Summary of modifications:

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

*	Some changes that probably could have been avoided if
	my editor had more display capabilities (like syntax
	highlighting):

	In order to be able to quickly analyze the program and
	its output, I gofmt'ed peg.go, and adjusted its output
	commands so that the generated code nearly looks like
	formatted with gofmt.

	Code sections inside string arguments of print()
	have been prefixed with \`+\` to make it easier to
	distinguish them from normal code.


[peg]: https://github.com/pointlander/peg
[peg(1)]: http://piumarta.com/software/peg/peg.1.html
[peg-markdown]: https://github.com/jgm/peg-markdown
[markdown_parser.leg]: https://github.com/jgm/peg-markdown/blob/master/markdown_parser.leg#L57

--  
Michael Teichgräber
