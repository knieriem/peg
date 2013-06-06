package peg

import (
	"fmt"
	"reflect"
	"strings"
)

var parserTemplate = strings.Replace(`\
{{range .Headers}}{{.}}{{end}}\
{{with def "package"}}\
package {{.}}

import (
	"fmt"
	"github.com/knieriem/peg"
)
{{end}}
const (\
{{range sortedRules}}
	rule{{.GoString}}{{if not .GetId}} = iota{{end}}{{end}}
)

type {{def "Peg"}} struct {
	{{def "userstate"}}
	Buffer string
	Min, Max int
	rules [{{numRules}}]func() bool
	commit func(int)bool
	ResetBuffer	func(string) string
}

func (p *{{def "Peg"}}) Parse(ruleId int) (err error) {
	if p.rules[ruleId]() {
		// Make sure thunkPosition is 0 (there may be a yyPop action on the stack).
		p.commit(0)
		return
	}
	return p.parseErr()
}

type {{id "e"}}rrPos struct {
	Line, Pos int
}

func	(e *{{id "e"}}rrPos) String() string {
	return fmt.Sprintf("%d:%d", e.Line, e.Pos)
}

type {{id "u"}}nexpectedCharError struct {
	After, At	{{id "e"}}rrPos
	Char	byte
}

func (e *{{id "u"}}nexpectedCharError) Error() string {
	return fmt.Sprintf("%v: unexpected character '%c'", &e.At, e.Char)
}

type {{id "u"}}nexpectedEOFError struct {
	After {{id "e"}}rrPos
}

func (e *{{id "u"}}nexpectedEOFError) Error() string {
	return fmt.Sprintf("%v: unexpected end of file", &e.After)
}

func (p *{{def "Peg"}}) parseErr() (err error) {
	var pos, after {{id "e"}}rrPos
	pos.Line = 1
	for i, c := range p.Buffer[0:] {
		if c == '\n' {
			pos.Line++
			pos.Pos = 0
		} else {
			pos.Pos++
		}
		if i == p.Min {
			if p.Min != p.Max {
				after = pos
			} else {
				break
			}
		} else if i == p.Max {
			break
		}
	}
	if p.Max >= len(p.Buffer) {
		err = &{{id "u"}}nexpectedEOFError{after}
	} else {
		err = &{{id "u"}}nexpectedCharError{after, pos, p.Buffer[p.Max]}
	}
	return
}

func (p *{{def "Peg"}}) Init() {
	var position int
{{if nvar}}\
	var yyp int
	var yy {{def "yystype"}}
	var yyval = make([]{{def "yystype"}}, 256)
{{end}}\

{{if .Actions}}\
	actions := [...]func(string, int){
{{	range .Actions}}		/* {{.GetId}} {{.GetRule}} */
		func(yytext string, _ int) {
{{.Code}}		},
{{	end}}
{{	if nvar}}\
		/* yyPush */
		func(_ string, count int) {
			yyp += count
			if yyp >= len(yyval) {
				s := make([]{{def "yystype"}}, cap(yyval)+256)
				copy(s, yyval)
				yyval = s
			}
		},
		/* yyPop */
		func(_ string, count int) {
			yyp -= count
		},
		/* yySet */
		func(_ string, count int) {
			yyval[yyp+count] = yy
		},
	}
	const (
		yyPush = {{len .Actions}} + iota
		yyPop
		yySet
	)
{{	else}}\
	}
{{	end}}\
{{	with $bits := actionBits}}
	type thunk struct {
		action uint{{$bits}}
		begin, end int
	}
	var thunkPosition, begin, end int
	thunks := make([]thunk, 32)
	doarg := func(action uint{{$bits}}, arg int) {
		if thunkPosition == len(thunks) {
			newThunks := make([]thunk, 2*len(thunks))
			copy(newThunks, thunks)
			thunks = newThunks
		}
		t := &thunks[thunkPosition]
		thunkPosition++
		t.action = action
		if arg != 0 {
			t.begin = arg // use begin to store an argument
		} else {
			t.begin = begin
		}
		t.end = end
	}
	do := func(action uint{{$bits}}) {
		doarg(action, 0)
	}
{{	end}}
	p.ResetBuffer = func(s string) (old string) {
		if position < len(p.Buffer) {
			old = p.Buffer[position:]
		}
		p.Buffer = s
		thunkPosition = 0
		position = 0
		p.Min = 0
		p.Max = 0
		end = 0
		return
	}
{{	if hasCommit}}
	p.commit = func(thunkPosition0 int) bool {
		if thunkPosition0 == 0 {
			s := ""
			for _, t := range thunks[:thunkPosition] {
				b := t.begin
				if b >= 0 && b <= t.end {
					s = p.Buffer[b:t.end]
				}
				magic := b
				actions[t.action](s, magic)
			}
			p.Min = position
			thunkPosition = 0
			return true
		}
		return false
	}
{{	end}}\
{{end}}\
{{with stats}}\
{{if .Match.Dot}}\
	matchDot := func() bool {
		if position < len(p.Buffer) {
			position++
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}
{{end}}
{{if .Match.Char}}\
	matchChar := func(c byte) bool {
		if (position < len(p.Buffer)) && (p.Buffer[position] == c) {
			position++
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}
{{end}}
{{if .Peek.Char}}\
	peekChar := func(c byte) bool {
		return position < len(p.Buffer) && p.Buffer[position] == c
	}
{{end}}
{{if .Match.String}}\
	matchString := func(s string) bool {
		length := len(s)
		next := position + length
		if (next <= len(p.Buffer)) && p.Buffer[position] == s[0] && (p.Buffer[position:next] == s) {
			position = next
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}
{{end}}
{{	if len $.Classes}}\
	classes := [...][32]uint8{
{{range $.Classes}}	{{.Index}}:	{{"{"}}{{range $i, $b := .Class}}{{if $i}}, {{end}}{{$b | printf "%d"}}{{end}}{{"}"}},
{{end}}\
	}
	matchClass := func(class uint) bool {
		if (position < len(p.Buffer)) &&
			((classes[class][p.Buffer[position]>>3] & (1 << (p.Buffer[position] & 7))) != 0) {
			position++
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}
{{if .Peek.Class}}\
	peekClass := func(class uint) bool {
		if (position < len(p.Buffer)) &&
			((classes[class][p.Buffer[position]>>3] & (1 << (p.Buffer[position] & 7))) != 0) {
			return true
		}
		return false
	}
{{end}}
{{	end}}
{{end}}\
	p.rules = [...]func() bool{
`, "\\\n", "", -1)

// used as template function `len'
func itemLength(item interface{}) (n int, err error) {
	v := reflect.ValueOf(item)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		n = v.Len()
	default:
		err = fmt.Errorf("can't compute the length of an item of type %s", v.Type())
	}
	return
}
