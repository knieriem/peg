package peg

import (
	"fmt"
	"reflect"
	"strings"
	"os"
)

var parserTemplate = strings.Replace(`\
{{range .Headers}}{{.}}{{end}}\
{{with def "package"}}\
package {{.}}

import (
	"fmt"
	"peg"
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
	ResetBuffer	func(string) string
}

func (p *{{def "Peg"}}) Parse(ruleId int) bool {
	if p.rules[ruleId]() {
		return true
	}
	return false
}

func (p *{{def "Peg"}}) PrintError() {
	line := 1
	character := 0
	for i, c := range p.Buffer[0:] {
		if c == '\n' {
			line++
			character = 0
		} else {
			character++
		}
		if i == p.Min {
			if p.Min != p.Max {
				fmt.Printf("parse error after line %v character %v\n", line, character)
			} else {
				break
			}
		} else if i == p.Max {
			break
		}
	}
	fmt.Printf("parse error: unexpected ")
	if p.Max >= len(p.Buffer) {
		fmt.Printf("end of file found\n")
	} else {
		fmt.Printf("'%c' at line %v character %v\n", p.Buffer[p.Max], line, character)
	}
}

func (p *{{def "Peg"}}) Init() {
	var position int
{{if nvar}}\
	var yyp int
	var yy {{def "yystype"}}
	var yyval = make([]{{def "yystype"}}, 200)
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
				s := make([]{{def "yystype"}}, cap(yyval)+200)
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
		thunks[thunkPosition].action = action
		if arg != 0 {
			thunks[thunkPosition].begin = arg // use begin to store an argument
		} else {
			thunks[thunkPosition].begin = begin
		}
		thunks[thunkPosition].end = end
		thunkPosition++
	}
	do := func(action uint{{$bits}}) {
		doarg(action, 0)
	}
{{	end}}
	p.ResetBuffer = func(s string) (old string) {
		if p.Max < len(p.Buffer) {
			old = p.Buffer[p.Max:]
		}
		p.Buffer = s
		thunkPosition = 0
		position = 0
		p.Min = 0
		p.Max = 0
		return
	}
{{	if hasCommit}}
	commit := func(thunkPosition0 int) bool {
		if thunkPosition0 == 0 {
			for i := 0; i < thunkPosition; i++ {
				b := thunks[i].begin
				e := thunks[i].end
				s := ""
				if b >= 0 && e <= len(p.Buffer) && b <= e {
					s = p.Buffer[b:e]
				}
				magic := b
				actions[thunks[i].action](s, magic)
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
{{if .Peek.Dot}}\
	peekDot := func() bool {
		return position < len(p.Buffer)
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
		if (next <= len(p.Buffer)) && (p.Buffer[position:next] == s) {
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
{{range $bitmap := $.Classes}}		{{"{"}}{{range $i, $b := $bitmap}}{{if $i}}, {{end}}{{$b | printf "%d"}}{{end}}{{"}"}},
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
{{	end}}
{{end}}\
	p.rules = [...]func() bool{
`, "\\\n", "", -1)


// used as template function `len'
func itemLength(item interface{}) (n int, err os.Error) {
	v := reflect.ValueOf(item)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		n = v.Len()
	default:
		err = fmt.Errorf("can't compute the length of an item of type %s", v.Type())
	}
	return
}
