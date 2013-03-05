// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package peg

import (
	"container/list"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/template"
)

var Verbose bool

type Type uint8

const (
	TypeUnknown Type = iota
	TypeRule
	TypeVariable
	TypeName
	TypeDot
	TypeCharacter
	TypeString
	TypeClass
	TypePredicate
	TypeCommit
	TypeBegin
	TypeEnd
	TypeAction
	TypeAlternate
	TypeUnorderedAlternate
	TypeSequence
	TypePeekFor
	TypePeekNot
	TypeQuery
	TypeStar
	TypePlus
	TypeNil
	TypeLast
)

func (t Type) GetType() Type {
	return t
}

type Node interface {
	fmt.Stringer
	GetType() Type
}

/* Used to represent TypeRule*/
type Rule interface {
	Node
	GetId() int
	GetExpression() Node
	SetExpression(e Node)
}

type rule struct {
	name       string
	id         int
	expression Node
	hasActions bool
	variables  map[string]*variable
}

func (r *rule) GetType() Type {
	return TypeRule
}

func (r *rule) GetId() int {
	return r.id
}

func (r *rule) GetExpression() Node {
	if r.expression == nil {
		return nilNode
	}
	return r.expression
}

func (r *rule) SetExpression(e Node) {
	r.expression = e
}

func (r *rule) String() string {
	return r.name
}

func (r *rule) GoString() string {
	b := []byte(r.String())
	for i := 0; i < len(b); i++ {
		if b[i] == '-' {
			b[i] = '_'
		}
	}
	return string(b)
}

type variable struct {
	name   string
	offset int
}

/* Used to represent TypeName */
type Name interface {
	Node
}

type name struct {
	Type
	string string
	varp   *variable
}

func (t *name) String() string {
	return t.string
}

/* Used to represent TypeDot, TypeCharacter, TypeString, TypeClass, TypePredicate, and TypeNil. */
type Token interface {
	Node
	GetClass() *characterClass
}

type token struct {
	Type
	string string
	class  *characterClass
}

func (t *token) GetClass() *characterClass {
	return t.class
}

func (t *token) String() string {
	return t.string
}

var nilNode = &token{Type: TypeNil, string: "<nil>"}

/* Used to represent TypeAction. */
type Action interface {
	Node
	GetId() int
	GetRule() string
}

type action struct {
	text string
	id   int
	rule *rule
}

func (a *action) GetType() Type {
	return TypeAction
}

func (a *action) String() string {
	return a.text
}

func (a *action) Code() (s string) {
	vmap := a.rule.variables
	ind := "\t\t\t"
	off := 0
	for _, v := range vmap {
		off--
		if v.offset == 0 {
			v.offset = off
		}
		s += fmt.Sprintf(ind+"%s := yyval[yyp%d]\n", v.name, v.offset)
	}
	s += fmt.Sprintf(ind+"%v\n", a)
	for _, v := range vmap {
		s += fmt.Sprintf(ind+"yyval[yyp%d] = %s\n", v.offset, v.name)
	}
	return
}

func (a *action) GetId() int {
	return a.id
}

func (a *action) GetRule() string {
	return a.rule.String()
}

/* Used to represent a TypeAlternate, TypeSequence, TypePeekFor, TypePeekNot, TypeQuery, TypeStar, or TypePlus */

type List interface {
	Node
	SetType(t Type)

	Init() *list.List
	Front() *list.Element
	PushBack(value interface{}) *list.Element
	Len() int
}

type nodeList struct {
	Type
	list.List
}

func (l *nodeList) SetType(t Type) {
	l.Type = t
}

func (l *nodeList) String() string {
	i := l.List.Front()
	s := "(" + i.Value.(fmt.Stringer).String()
	for i = i.Next(); i != nil; i = i.Next() {
		s += " / " + i.Value.(fmt.Stringer).String()
	}
	return s + ")"
}

/* Used to represent character classes. */
type characterClass [32]uint8

func (c *characterClass) copy() (class *characterClass) {
	class = new(characterClass)
	copy(class[0:], c[0:])
	return
}
func (c *characterClass) add(character uint8)      { c[character>>3] |= (1 << (character & 7)) }
func (c *characterClass) has(character uint8) bool { return c[character>>3]&(1<<(character&7)) != 0 }
func (c *characterClass) complement() {
	for i := range *c {
		c[i] = ^c[i]
	}
}
func (c *characterClass) union(class *characterClass) {
	for index, value := range *class {
		c[index] |= value
	}
}
func (c *characterClass) intersection(class *characterClass) {
	for index, value := range *class {
		c[index] &= value
	}
}
func (c *characterClass) len() (length int) {
	for character := 0; character < 256; character++ {
		if c.has(uint8(character)) {
			length++
		}
	}
	return
}
func (c *characterClass) String() (class string) {
	escape := func(c uint8) string {
		s := ""
		switch uint8(c) {
		case '\a':
			s = `\a` /* bel */
		case '\b':
			s = `\b` /* bs */
		case '\f':
			s = `\f` /* ff */
		case '\n':
			s = `\n` /* nl */
		case '\r':
			s = `\r` /* cr */
		case '\t':
			s = `\t` /* ht */
		case '\v':
			s = `\v` /* vt */
		case '\'':
			s = `\'` /* ' */
		case '"':
			s = `\"` /* " */
		case '[':
			s = `\[` /* [ */
		case ']':
			s = `\]` /* ] */
		case '\\':
			s = `\\` /* \ */
		case '-':
			s = `\-` /* - */
		default:
			switch {
			case c >= 0 && c < 32 || c >= 0x80:
				s = fmt.Sprintf("\\%03o", c)
			default:
				s = fmt.Sprintf("%c", c)
			}
		}
		return s
	}
	class = ""
	l := 0
	for character := 0; character < 256; character++ {
		if c.has(uint8(character)) {
			if l == 0 {
				class += escape(uint8(character))
			}
			l++
		} else {
			if l == 2 {
				class += escape(uint8(character - 1))
			} else if l > 2 {
				class += "-" + escape(uint8(character-1))
			}
			l = 0
		}
	}
	if l >= 2 {
		class += "-" + escape(255)
	}
	return
}

type classEntry struct {
	Index int
	Class *characterClass
}

/* A tree data structure into which a PEG can be parsed. */
type Tree struct {
	rules      map[string]*rule
	rulesCount map[string]uint
	ruleId     int
	varp       *variable
	Headers    []string
	trailers   []string
	list.List
	Actions         []*action
	Classes         map[string]classEntry
	defines         map[string]string
	switchExcl      map[string]bool
	stack           [1024]Node
	top             int
	inline, _switch bool
}

func New(inline, _switch bool) *Tree {
	return &Tree{rules: make(map[string]*rule),
		rulesCount: make(map[string]uint),
		Classes:    make(map[string]classEntry),
		defines: map[string]string{
			"package":   "",
			"Peg":       "yyParser",
			"userstate": "",
			"yystype":   "yyStype",
			"noexport":  "",
		},
		inline:  inline,
		_switch: _switch}
}

func (t *Tree) push(n Node) {
	t.top++
	t.stack[t.top] = n
}

func (t *Tree) pop() Node {
	n := t.stack[t.top]
	t.top--
	return n
}

func (t *Tree) currentRule() *rule {
	return t.stack[1].(*rule)
}

func (t *Tree) AddRule(name string) {
	t.push(&rule{name: name, id: t.ruleId})
	t.ruleId++
}

func (t *Tree) AddExpression() {
	expression := t.pop()
	rule := t.pop().(Rule)
	rule.SetExpression(expression)
	t.PushBack(rule)
}

func (t *Tree) AddHeader(text string) {
	t.Headers = append(t.Headers, text)
}

func (t *Tree) AddTrailer(text string) {
	t.trailers = append(t.trailers, text)
}

func (t *Tree) AddVariable(text string) {
	var v *variable

	r := t.currentRule()
	if r.variables == nil {
		r.variables = make(map[string]*variable)
	}
	if v = r.variables[text]; v == nil {
		v = &variable{name: text}
	}
	r.variables[text] = v
	t.varp = v
}

func (t *Tree) AddName(text string) {
	t.rules[text] = &rule{}
	t.push(&name{Type: TypeName, string: text, varp: t.varp})
	t.varp = nil
}

var dot *token = &token{Type: TypeDot, string: "."}

func (t *Tree) AddDot() { t.push(dot) }
func (t *Tree) AddString(text string) {
	length := len(text)
s:
	switch {
	case length == 1:
	case length > 1:
		if text[0] == '\\' {
			switch length {
			case 2:
				break s
			case 4:
				if text[1] >= '0' && text[1] <= '9' {
					break s
				}
			}
		}
		fallthrough
	default:
		t.push(&token{Type: TypeString, string: text})
		return
	}
	t.push(&token{Type: TypeCharacter, string: text})
}
func (t *Tree) AddClass(text string) {
	t.push(&token{Type: TypeClass, string: text})
	if _, ok := t.Classes[text]; !ok {
		c := new(characterClass)
		t.Classes[text] = classEntry{len(t.Classes), c}
		inverse := false
		if text[0] == '^' {
			inverse = true
			text = text[1:]
		}
		var last uint8
		hasLast := false
		for i := 0; i < (len(text) - 1); i++ {
			switch {
			case (text[i] == '-') && hasLast:
				i++
				for j := last; j <= text[i]; j++ {
					c.add(j)
				}
				hasLast = false
			case (text[i] == '\\'):
				i++
				last, hasLast = text[i], true
				switch last {
				case 'a':
					last = '\a' /* bel */
				case 'b':
					last = '\b' /* bs */
				case 'f':
					last = '\f' /* ff */
				case 'n':
					last = '\n' /* nl */
				case 'r':
					last = '\r' /* cr */
				case 't':
					last = '\t' /* ht */
				case 'v':
					last = '\v' /* vt */
				}
				c.add(last)
			default:
				last, hasLast = text[i], true
				c.add(last)
			}
		}
		c.add(text[len(text)-1])
		if inverse {
			c.complement()
		}
	}
}
func (t *Tree) AddPredicate(text string) {
	t.push(&token{Type: TypePredicate, string: strings.TrimSpace(text)})
}

var commit *token = &token{Type: TypeCommit, string: "commit"}

func (t *Tree) AddCommit() { t.push(commit) }

var begin *token = &token{Type: TypeBegin, string: "<"}

func (t *Tree) AddBegin() { t.push(begin) }

var end *token = &token{Type: TypeEnd, string: ">"}

func (t *Tree) AddEnd() { t.push(end) }
func (t *Tree) AddNil() { t.push(nilNode) }
func (t *Tree) AddAction(text string) {
	b := []byte(text)
	for i := 0; i < len(b)-1; i++ {
		if b[i] == '$' && b[i+1] == '$' {
			b[i], b[i+1] = 'y', 'y'
		}
	}
	a := &action{text: string(b), id: len(t.Actions), rule: t.currentRule()}
	t.currentRule().hasActions = true
	t.Actions = append(t.Actions, a)
	t.push(a)
}
func (t *Tree) Define(name, text string) {
	if _, ok := t.defines[name]; ok {
		t.defines[name] = text
	}
}
func (t *Tree) SwitchExclude(rule string) {
	if t.switchExcl == nil {
		t.switchExcl = make(map[string]bool, 16)
	}
	t.switchExcl[rule] = true
}

func (t *Tree) addList(listType Type) {
	a := t.pop()
	b := t.pop()
	var l List
	if b.GetType() == listType {
		l = b.(List)
	} else {
		l = &nodeList{Type: listType}
		l.PushBack(b)
	}
	l.PushBack(a)
	t.push(l)
}
func (t *Tree) AddAlternate() { t.addList(TypeAlternate) }
func (t *Tree) AddSequence()  { t.addList(TypeSequence) }

func (t *Tree) addFix(fixType Type) {
	n := &nodeList{Type: fixType}
	n.PushBack(t.pop())
	t.push(n)
}
func (t *Tree) AddPeekFor() { t.addFix(TypePeekFor) }
func (t *Tree) AddPeekNot() { t.addFix(TypePeekNot) }
func (t *Tree) AddQuery()   { t.addFix(TypeQuery) }
func (t *Tree) AddStar()    { t.addFix(TypeStar) }
func (t *Tree) AddPlus()    { t.addFix(TypePlus) }

func join(tasks []func()) {
	length := len(tasks)
	done := make(chan int, length)
	for _, task := range tasks {
		go func(task func()) { task(); done <- 1 }(task)
	}
	for d := <-done; d < length; d += <-done {
	}
}

var anyChar = func() (c *characterClass) {
	c = new(characterClass)
	return
}()

func (t *Tree) Compile(out io.Writer, optiFlags string) {
	counts := [TypeLast]uint{}
	nvar := 0

	O := parseOptiFlags(optiFlags)

	for element := t.Front(); element != nil; element = element.Next() {
		node := element.Value.(Node)
		switch node.GetType() {
		case TypeRule:
			rule := node.(*rule)
			t.rules[rule.String()] = rule
			nvar += len(rule.variables)
		}
	}
	for name, r := range t.rules {
		if r.name == "" {
			r := &rule{name: name, id: t.ruleId}
			t.ruleId++
			t.rules[name] = r
			t.PushBack(r)
		}
	}

	join([]func(){
		func() {
			var countTypes func(node Node)
			countTypes = func(node Node) {
				t := node.GetType()
				counts[t]++
				switch t {
				case TypeRule:
					countTypes(node.(Rule).GetExpression())
				case TypeAlternate, TypeUnorderedAlternate, TypeSequence:
					for element := node.(List).Front(); element != nil; element = element.Next() {
						countTypes(element.Value.(Node))
					}
				case TypePeekFor, TypePeekNot, TypeQuery, TypeStar, TypePlus:
					countTypes(node.(List).Front().Value.(Node))
				}
			}
			for _, rule := range t.rules {
				countTypes(rule)
			}
		},
		func() {
			var countRules func(node Node)
			ruleReached := make([]bool, len(t.rules))
			countRules = func(node Node) {
				switch node.GetType() {
				case TypeRule:
					rule := node.(Rule)
					name, id := rule.String(), rule.GetId()
					if count, ok := t.rulesCount[name]; ok {
						t.rulesCount[name] = count + 1
					} else {
						t.rulesCount[name] = 1
					}
					if ruleReached[id] {
						return
					}
					ruleReached[id] = true
					countRules(rule.GetExpression())
				case TypeName:
					countRules(t.rules[node.String()])
				case TypeAlternate, TypeUnorderedAlternate, TypeSequence:
					for element := node.(List).Front(); element != nil; element = element.Next() {
						countRules(element.Value.(Node))
					}
				case TypePeekFor, TypePeekNot, TypeQuery, TypeStar, TypePlus:
					countRules(node.(List).Front().Value.(Node))
				}
			}
			for element := t.Front(); element != nil; element = element.Next() {
				node := element.Value.(Node)
				if node.GetType() == TypeRule {
					countRules(node.(*rule))
					break
				}
			}
		},
		func() {
			var checkRecursion func(node Node) bool
			ruleReached := make([]bool, len(t.rules))
			checkRecursion = func(node Node) bool {
				switch node.GetType() {
				case TypeRule:
					rule := node.(Rule)
					id := rule.GetId()
					if ruleReached[id] {
						fmt.Fprintf(os.Stderr, "possible infinite left recursion in rule '%v'\n", node)
						return false
					}
					ruleReached[id] = true
					consumes := checkRecursion(rule.GetExpression())
					ruleReached[id] = false
					return consumes
				case TypeAlternate:
					for element := node.(List).Front(); element != nil; element = element.Next() {
						if !checkRecursion(element.Value.(Node)) {
							return false
						}
					}
					return true
				case TypeSequence:
					for element := node.(List).Front(); element != nil; element = element.Next() {
						if checkRecursion(element.Value.(Node)) {
							return true
						}
					}
				case TypeName:
					return checkRecursion(t.rules[node.String()])
				case TypePlus:
					return checkRecursion(node.(List).Front().Value.(Node))
				case TypeCharacter, TypeString:
					return len(node.String()) > 0
				case TypeDot, TypeClass:
					return true
				}
				return false
			}
			for _, rule := range t.rules {
				checkRecursion(rule)
			}
		}})

	var inlineLeafes func(node Node) Node
	inlineLeafes = func(node Node) (ret Node) {
		ret = node
		switch node.GetType() {
		case TypeRule:
			rule := node.(Rule)
			switch x := rule.GetExpression(); x.GetType() {
			case TypeCharacter, TypeDot, TypeClass, TypeString:
				ret = x
			case TypePlus, TypeStar, TypeQuery, TypePeekNot, TypePeekFor:
				switch x.(List).Front().Value.(Node).GetType() {
				case TypeCharacter, TypeDot, TypeClass, TypeString:
					ret = x
				}
			}
		case TypeName:
			r := t.rules[node.String()]
			x := inlineLeafes(r)
			if r != x {
				stats.inlineLeafs++
				ret = x
			}
		case TypeSequence, TypeAlternate:
			for el := node.(List).Front(); el != nil; el = el.Next() {
				el.Value = inlineLeafes(el.Value.(Node))
			}
		case TypePlus, TypeStar, TypeQuery, TypePeekNot, TypePeekFor:
			v := &node.(List).Front().Value
			*v = inlineLeafes((*v).(Node))
		}
		return
	}
	if O.inlineLeafs {
		for _, rule := range t.rules {
			inlineLeafes(rule.GetExpression())
		}
	}

	if t._switch {
		var optimizeAlternates func(node Node) (consumes, eof, peek bool, class *characterClass)
		cache := make([]struct {
			reached, consumes, eof, peek bool
			class                        *characterClass
		}, len(t.rules))
		optimizeAlternates = func(node Node) (consumes, eof, peek bool, class *characterClass) {
			switch node.GetType() {
			case TypeRule:
				rule := node.(Rule)
				if t.switchExcl != nil && t.switchExcl[rule.String()] {
					return
				}
				cache := &cache[rule.GetId()]
				if cache.reached {
					consumes, eof, peek, class = cache.consumes, cache.eof, cache.peek, cache.class
					if class == nil {
						class = anyChar
					}
					return
				}
				cache.reached = true
				consumes, eof, peek, class = optimizeAlternates(rule.GetExpression())
				cache.consumes, cache.eof, cache.peek, cache.class = consumes, eof, peek, class
			case TypeName:
				consumes, eof, peek, class = optimizeAlternates(t.rules[node.String()])
			case TypeDot:
				consumes, class = true, new(characterClass)
				for index, _ := range *class {
					class[index] = 0xff
				}
			case TypeString, TypeCharacter:
				if node.String() == "" {
					consumes, class = true, anyChar
					return
				}
				consumes, class = true, new(characterClass)
				b := node.String()[0]
				if b == '\\' {
					b = node.String()[1]
					switch b {
					case 'a':
						b = '\a' /* bel */
					case 'b':
						b = '\b' /* bs */
					case 'f':
						b = '\f' /* ff */
					case 'n':
						b = '\n' /* nl */
					case 'r':
						b = '\r' /* cr */
					case 't':
						b = '\t' /* ht */
					case 'v':
						b = '\v' /* vt */
					default:
						if s := node.String(); len(s) == 4 {
							b = (s[1]-'0')*64 + (s[2]-'0')*8 + s[3] - '0'
						}
					}
				}
				class.add(b)
			case TypeClass:
				consumes, class = true, t.Classes[node.String()].Class
			case TypeAlternate:
				consumes, peek, class = true, true, new(characterClass)
				alternate := node.(List)
				mconsumes, meof, mpeek, properties, c :=
					consumes, eof, peek, make([]struct {
						intersects bool
						class      *characterClass
					}, alternate.Len()), 0
				empty := false
				for element := alternate.Front(); element != nil; element = element.Next() {
					mconsumes, meof, mpeek, properties[c].class = optimizeAlternates(element.Value.(Node))
					consumes, eof, peek = consumes && mconsumes, eof || meof, peek && mpeek
					if properties[c].class != nil {
						class.union(properties[c].class)
						if properties[c].class.len() == 0 {
							empty = true
						}
					}
					c++
				}
				if eof {
					break
				}
				intersections := 0
			compare:
				for ai, a := range properties[0 : len(properties)-1] {
					for _, b := range properties[ai+1:] {
						for i, v := range *a.class {
							if (b.class[i] & v) != 0 {
								intersections++
								properties[ai].intersects = true
								continue compare
							}
						}
					}
				}
				if empty {
					class = new(characterClass)
					consumes = false
					break
				}
				if intersections < len(properties) && len(properties) >= 2 {
					c, unordered, ordered, max :=
						0, &nodeList{Type: TypeUnorderedAlternate}, &nodeList{Type: TypeAlternate}, 0
					for element := alternate.Front(); element != nil; element = element.Next() {
						if properties[c].intersects {
							ordered.PushBack(element.Value)
						} else {
							class := &token{Type: TypeClass, string: properties[c].class.String(), class: properties[c].class}

							sequence, predicate, length :=
								&nodeList{Type: TypeSequence}, &nodeList{Type: TypePeekFor}, properties[c].class.len()
							predicate.PushBack(class)
							sequence.PushBack(predicate)
							sequence.PushBack(element.Value)

							if element.Value.(Node).GetType() == TypeString && element.Value.(Node).String() == "" {
								unordered.PushBack(sequence)
							} else if element.Value.(Node).GetType() == TypeNil {
								unordered.PushBack(sequence)
							} else if length > max {
								unordered.PushBack(sequence)
								max = length
							} else {
								unordered.PushFront(sequence)
							}
						}
						c++
					}
					alternate.Init()
					if ordered.Len() == 0 {
						alternate.SetType(TypeUnorderedAlternate)
						for element := unordered.Front(); element != nil; element = element.Next() {
							alternate.PushBack(element.Value)
						}
					} else {
						for element := ordered.Front(); element != nil; element = element.Next() {
							alternate.PushBack(element.Value)
						}
						if unordered.Len() == 1 {
							alternate.PushBack(unordered.Front().Value.(List).Front().Next().Value)
						} else {
							alternate.PushBack(unordered)
						}
					}
				}
			case TypeSequence:
				sequence := node.(List)
				meof, classes, c, element :=
					eof, make([]struct {
						peek  bool
						class *characterClass
					}, sequence.Len()), 0, sequence.Front()
				for ; !consumes && element != nil; element, c = element.Next(), c+1 {
					consumes, meof, classes[c].peek, classes[c].class = optimizeAlternates(element.Value.(Node))
					eof, peek = eof || meof, peek || classes[c].peek
				}
				eof, peek, class = !consumes && eof, !consumes && peek, new(characterClass)
				for c--; c >= 0; c-- {
					if classes[c].class != nil {
						if classes[c].peek {
							class.intersection(classes[c].class)
						} else {
							class.union(classes[c].class)
						}
					}
				}
				for ; element != nil; element = element.Next() {
					optimizeAlternates(element.Value.(Node))
				}
			case TypePeekNot:
				peek = true
				// might be buggy
				_, eof, _, _ = optimizeAlternates(node.(List).Front().Value.(Node))
				class = new(characterClass)
				eof = !eof
				class = class.copy()
				class.complement()
			case TypePeekFor:
				peek = true
				fallthrough
			case TypeQuery, TypeStar:
				_, eof, _, class = optimizeAlternates(node.(List).Front().Value.(Node))
			case TypePlus:
				consumes, eof, peek, class = optimizeAlternates(node.(List).Front().Value.(Node))
			case TypeAction, TypeNil:
				class = new(characterClass)
			}
			return
		}
		for element := t.Front(); element != nil; element = element.Next() {
			node := element.Value.(Node)
			if node.GetType() == TypeRule {
				optimizeAlternates(node.(*rule))
				break
			}
		}
	}

	w := newWriter(out)
	w.elimRestore = O.elimRestore
	print := func(format string, a ...interface{}) {
		if !w.dryRun {
			fmt.Fprintf(w, format, a...)
		}
	}

	var printRule func(node Node)
	var compile func(expression Node, ko *label) (chgFlags, chgFlags)
	printRule = func(node Node) {
		switch node.GetType() {
		case TypeRule:
			print("%v <- ", node)
			expression := node.(Rule).GetExpression()
			if expression != nilNode {
				printRule(expression)
			}
		case TypeDot:
			print(".")
		case TypeName:
			print("%v", node)
		case TypeCharacter,
			TypeString:
			print("'%v'", node)
		case TypeClass:
			print("[%v]", node)
		case TypePredicate:
			print("&{%v}", node)
		case TypeAction:
			print("{%v}", node)
		case TypeCommit:
			print("commit")
		case TypeBegin:
			print("<")
		case TypeEnd:
			print(">")
		case TypeAlternate:
			print("(")
			list := node.(List)
			element := list.Front()
			printRule(element.Value.(Node))
			for element = element.Next(); element != nil; element = element.Next() {
				print(" / ")
				printRule(element.Value.(Node))
			}
			print(")")
		case TypeUnorderedAlternate:
			print("(")
			element := node.(List).Front()
			printRule(element.Value.(Node))
			for element = element.Next(); element != nil; element = element.Next() {
				print(" | ")
				printRule(element.Value.(Node))
			}
			print(")")
		case TypeSequence:
			print("(")
			element := node.(List).Front()
			printRule(element.Value.(Node))
			for element = element.Next(); element != nil; element = element.Next() {
				print(" ")
				printRule(element.Value.(Node))
			}
			print(")")
		case TypePeekFor:
			print("&")
			printRule(node.(List).Front().Value.(Node))
		case TypePeekNot:
			print("!")
			printRule(node.(List).Front().Value.(Node))
		case TypeQuery:
			printRule(node.(List).Front().Value.(Node))
			print("?")
		case TypeStar:
			printRule(node.(List).Front().Value.(Node))
			print("*")
		case TypePlus:
			printRule(node.(List).Front().Value.(Node))
			print("+")
		default:
			fmt.Fprintf(os.Stderr, "illegal node type: %v\n", node.GetType())
		}
	}
	compileExpression := func(rule *rule, ko *label) (cko, cok chgFlags) {
		nvar := len(rule.variables)
		if nvar > 0 {
			w.lnPrint("doarg(yyPush, %d)", nvar)
		}
		cko, cok = compile(rule.GetExpression(), ko)
		if nvar > 0 {
			w.lnPrint("doarg(yyPop, %d)", nvar)
			cko.thPos = true
			cok.thPos = true
		}
		return
	}
	canCompilePeek := func(node Node, jumpIfTrue bool, label *label) bool {
		if !O.peek {
			return false
		}
		switch node.GetType() {
		case TypeDot:
			label.cJump(jumpIfTrue, "(position < len(p.Buffer))")
			stats.Peek.Dot++
		case TypeCharacter:
			label.cJump(jumpIfTrue, "peekChar('%v')", node)
			stats.Peek.Char++
		case TypeClass:
			label.cJump(jumpIfTrue, "peekClass(%d)", t.Classes[node.String()].Index)
			stats.Peek.Class++
		case TypePredicate:
			label.cJump(jumpIfTrue, "(%v)", node)
		default:
			return false
		}
		return true
	}
	compile = func(node Node, ko *label) (chgko, chgok chgFlags) {
		updateFlags := func(cko, cok chgFlags) (chgFlags, chgFlags) {
			chgko, chgok = updateChgFlags(chgko, chgok, cko, cok)
			return chgko, chgok
		}
		switch node.GetType() {
		case TypeRule:
			fmt.Fprintf(os.Stderr, "internal error #1 (%v)\n", node)
		case TypeDot:
			ko.cJump(false, "matchDot()")
			stats.Match.Dot++
			chgok.pos = true
		case TypeName:
			varp := node.(*name).varp
			name := node.String()
			rule := t.rules[name]
			if t.inline && t.rulesCount[name] == 1 {
				chgko, chgok = compileExpression(rule, ko)
			} else {
				ko.cJump(false, "p.rules[rule%s]()", rule.GoString())
				if len(rule.variables) != 0 || rule.hasActions {
					chgok.thPos = true
				}
				chgok.pos = true // safe guess
			}
			if varp != nil {
				w.lnPrint("doarg(yySet, %d)", varp.offset)
				chgok.thPos = true
			}
		case TypeCharacter:
			ko.cJump(false, "matchChar('%v')", node)
			stats.Match.Char++
			chgok.pos = true
		case TypeString:
			if s := node.String(); s != "" {
				ko.cJump(false, "matchString(\"%s\")", s)
				stats.Match.String++
				chgok.pos = true
			}
		case TypeClass:
			ko.cJump(false, "matchClass(%d)", t.Classes[node.String()].Index)
			chgok.pos = true
		case TypePredicate:
			ko.cJump(false, "(%v)", node)
		case TypeAction:
			w.lnPrint("do(%d)", node.(Action).GetId())
			chgok.thPos = true
		case TypeCommit:
			ko.cJump(false, "(commit(thunkPosition0))")
			chgko.thPos = true
		case TypeBegin:
			if t.Actions != nil {
				w.lnPrint("begin = position")
			}
		case TypeEnd:
			if t.Actions != nil {
				w.lnPrint("end = position")
			}
		case TypeAlternate:
			list := node.(List)
			ok := w.newLabel()
			element := list.Front()
			if ok.unsafe() {
				w.begin()
				ok.save()
			}
			var next *label
			for element.Next() != nil {
				next = w.newLabel()
				cko, _ := updateFlags(compile(element.Value.(Node), next))
				ok.jump()
				if next.used {
					ok.lrestore(next, cko.pos, cko.thPos)
				}
				element = element.Next()
			}
			if next == nil || next.used {
				updateFlags(compile(element.Value.(Node), ko))
			}
			if ok.unsafe() {
				w.end()
			}
			if ok.used {
				ok.label()
			}
		case TypeUnorderedAlternate:
			list := node.(List)
			done, ok := ko, w.newLabel()
			w.begin()
			done.cJump(true, "position == len(p.Buffer)")
			w.lnPrint("switch p.Buffer[position] {")
			element := list.Front()
			for ; element != nil; element = element.Next() {
				sequence := element.Value.(List).Front()
				class := sequence.Value.(List).Front().Value.(Node).(Token).GetClass()
				node := sequence.Next().Value.(Node)

				if element.Next() == nil {
					if class.len() > 2 {
						w.lnPrint("default:")
						w.indent++
						updateFlags(compile(node, done))
						w.indent--
						break
					}
				}

				w.lnPrint("case")
				comma := false
				for d := 0; d < 256; d++ {
					if class.has(uint8(d)) {
						if comma {
							print(",")
						}
						s := ""
						switch uint8(d) {
						case '\a':
							s = `\a` /* bel */
						case '\b':
							s = `\b` /* bs */
						case '\f':
							s = `\f` /* ff */
						case '\n':
							s = `\n` /* nl */
						case '\r':
							s = `\r` /* cr */
						case '\t':
							s = `\t` /* ht */
						case '\v':
							s = `\v` /* vt */
						case '\\':
							s = `\\` /* \ */
						case '\'':
							s = `\'` /* ' */
						default:
							switch {
							case d >= 0 && d < 32 || d >= 0x80:
								s = fmt.Sprintf("\\%03o", d)
							default:
								s = fmt.Sprintf("%c", d)
							}
						}
						print(" '%s'", s)
						comma = true
					}
				}
				print(":")
				w.indent++
				if O.unorderedFirstItem {
					updateFlags(compileOptFirst(w, node, done, compile))
				} else {
					updateFlags(compile(node, done))
				}
				w.lnPrint("break")
				w.indent--
				if element.Next() == nil {
					w.lnPrint("default:")
					w.indent++
					done.jump()
					w.indent--
				}
			}
			w.lnPrint("}")
			w.end()
			if ok.used {
				ok.label()
			}
		case TypeSequence:
			var cs []string
			var peek Type
			var element0 = node.(List).Front()

			if O.seqPeekNot {
				for el := element0; el != nil; el = el.Next() {
					sub := el.Value.(Node)
					switch typ := sub.GetType(); typ {
					case TypePeekNot:
						switch child := sub.(List).Front().Value.(Node); child.GetType() {
						case TypeCharacter:
							cs = append(cs, "'"+child.String()+"'")
							continue
						}
					case TypeDot:
						if len(cs) > 0 {
							peek = typ
							element0 = el.Next()
						}
					default:
						if len(cs) > 1 {
							peek = typ
							element0 = el
						}
					}
					break
				}
			}

			if peek != 0 {
				stats.seqIfNot++
				ko.cJump(true, "position == len(p.Buffer)")
				w.lnPrint("switch p.Buffer[position] {")

				w.lnPrint("case %s:", strings.Join(cs, ", "))
				w.indent++
				ko.jump()
				w.indent--
				w.lnPrint("default:")
				w.indent++
				if peek == TypeDot {
					w.lnPrint("position++")
					chgok.pos = true
				}
			}
			for element := element0; element != nil; element = element.Next() {
				cko, cok := compile(element.Value.(Node), ko)
				if element.Next() == nil {
					if chgok.pos {
						cko.pos = true
					}
					if chgok.thPos {
						cko.thPos = true
					}
				}
				updateFlags(cko, cok)
			}
			if peek != 0 {
				w.indent--
				w.lnPrint("}")
			}
		case TypePeekFor:
			sub := node.(List).Front().Value.(Node)
			if canCompilePeek(sub, false, ko) {
				return
			}
			l := w.newLabel()
			l.saveBlock()
			cko, cok := compile(sub, ko)
			l.lrestore(nil, cok.pos, cok.thPos)
			chgko = cko
		case TypePeekNot:
			sub := node.(List).Front().Value.(Node)
			if canCompilePeek(sub, true, ko) {
				return
			}
			ok := w.newLabel()
			ok.saveBlock()
			cko, cok := compile(sub, ok)
			ko.jump()
			if ok.used {
				ok.restore(cko.pos, cko.thPos)
			}
			chgko = cok
		case TypeQuery:
			sub := node.(List).Front().Value.(Node)
			switch sub.GetType() {
			case TypeCharacter:
				w.lnPrint("matchChar('%v')", sub)
				chgok.pos = true
				return
			case TypeDot:
				w.lnPrint("matchDot()")
				chgok.pos = true
				return
			}
			qko := w.newLabel()
			qok := w.newLabel()
			qko.saveBlock()
			cko, cok := compile(sub, qko)
			if qko.unsafe() {
				qok.jump()
			}
			if qko.used {
				qko.restore(cko.pos, cko.thPos)
			}
			if qko.unsafe() {
				qok.label()
			}
			chgok = cok
		case TypeStar:
			again := w.newLabel()
			out := w.newLabel()
			again.label()
			out.saveBlock()
			cko, cok := compile(node.(List).Front().Value.(Node), out)
			again.jump()
			out.restore(cko.pos, cko.thPos)
			chgok = cok
		case TypePlus:
			again := w.newLabel()
			out := w.newLabel()
			updateFlags(compile(node.(List).Front().Value.(Node), ko))
			again.label()
			out.saveBlock()
			cko, _ := compile(node.(List).Front().Value.(Node), out)
			again.jump()
			if out.used {
				out.restore(cko.pos, cko.thPos)
			}
		case TypeNil:
		default:
			fmt.Fprintf(os.Stderr, "illegal node type: %v\n", node.GetType())
		}
		return
	}

	// dry compilation
	// figure out which items need to restore position resp. thunkPosition,
	// storing into w.saveFlags
	w.setDry(true)
	for element := t.Front(); element != nil; element = element.Next() {
		node := element.Value.(Node)
		if node.GetType() != TypeRule {
			continue
		}
		rule := node.(*rule)
		expression := rule.GetExpression()
		if expression == nilNode {
			continue
		}
		ko := w.newLabel()
		ko.sid = 0
		if count, ok := t.rulesCount[rule.String()]; !ok {
		} else if t.inline && count == 1 && ko.id != 0 {
			continue
		}
		ko.save()
		cko, _ := compileExpression(rule, ko)
		if ko.used {
			ko.restore(cko.pos, cko.thPos)
		}
	}
	w.setDry(false)
	if Verbose {
		log.Printf("%+v\n", stats)
	}
	tpl := template.New("parser")
	tpl.Funcs(template.FuncMap{
		"len": itemLength,
		"def": func(key string) string { return t.defines[key] },
		"id": func(identifier string) string {
			if t.defines["noexport"] != "" {
				return identifier
			}
			return strings.Title(identifier)
		},
		"stats":    func() *statValues { return &stats },
		"nvar":     func() int { return nvar },
		"numRules": func() int { return len(t.rules) },
		"sortedRules": func() (r []*rule) {
			for el := t.Front(); el != nil; el = el.Next() {
				node := el.Value.(Node)
				if node.GetType() != TypeRule {
					continue
				}
				r = append(r, node.(*rule))
			}
			return
		},
		"hasCommit": func() bool { return counts[TypeCommit] > 0 },
		"actionBits": func() (bits int) {
			for n := len(t.Actions); n != 0; n >>= 1 {
				bits++
			}
			switch {
			case bits < 8:
				bits = 8
			case bits < 16:
				bits = 16
			case bits < 32:
				bits = 32
			case bits < 64:
				bits = 64
			}
			return
		},
	})
	if _, err := tpl.Parse(parserTemplate); err != nil {
		log.Fatal(err)
	}
	if err := tpl.Execute(w, t); err != nil {
		log.Fatal(err)
	}

	/* now for the real compile pass */
	for element := t.Front(); element != nil; element = element.Next() {
		node := element.Value.(Node)
		if node.GetType() != TypeRule {
			continue
		}
		rule := node.(*rule)
		expression := rule.GetExpression()
		if expression == nilNode {
			fmt.Fprintf(os.Stderr, "rule '%v' used but not defined\n", rule)
			w.lnPrint("nil,")
			continue
		}
		ko := w.newLabel()
		ko.sid = 0
		w.lnPrint("/* %v ", rule.GetId())
		printRule(rule)
		print(" */")
		if count, ok := t.rulesCount[rule.String()]; !ok {
			fmt.Fprintf(os.Stderr, "rule '%v' defined but not used\n", rule)
		} else if t.inline && count == 1 && ko.id != 0 {
			w.lnPrint("nil,")
			continue
		}
		w.lnPrint("func() bool {")
		w.indent++
		ko.save()
		cko, _ := compileExpression(rule, ko)
		w.lnPrint("return true")
		if ko.used {
			ko.restore(cko.pos, cko.thPos)
			w.lnPrint("return false")
		}
		w.indent--
		w.lnPrint("},")
	}
	print("\n\t}")
	print("\n}\n")

	for _, s := range t.trailers {
		print("%s", s)
	}
}

func compileOptFirst(w *writer, node Node, ko *label, compile func(Node, *label) (chgFlags, chgFlags)) (chgko, chgok chgFlags) {
	updateFlags := func(cko, cok chgFlags) (chgFlags, chgFlags) {
		chgko, chgok = updateChgFlags(chgko, chgok, cko, cok)
		return chgko, chgok
	}
	switch node.GetType() {
	case TypeCharacter:
		w.lnPrint("position++ // matchChar")
		chgok.pos = true
		stats.optFirst.char++
	case TypeDot:
		chgok.pos = true
		stats.optFirst.dot++
	case TypeClass:
		w.lnPrint("position++ // matchClass")
		chgok.pos = true
		stats.optFirst.class++
	case TypeString:
		if s := node.String(); len(s) == 2 {
			w.lnPrint("position++ // matchString(`%s`)", s)
			ko.cJump(false, "matchChar('%c')", s[1])
			chgok.pos = true
			stats.Match.Char++
			stats.optFirst.str++
		} else if s != "" {
			w.lnPrint("position++")
			ko.cJump(false, "matchString(\"%s\")", s[1:])
			chgok.pos = true
			stats.Match.String++
			stats.optFirst.str++
		}
	case TypeSequence:
		front := node.(List).Front()
		for element := front; element != nil; element = element.Next() {
			if element == front {
				updateFlags(compileOptFirst(w, element.Value.(Node), ko, compile))
			} else {
				updateFlags(compile(element.Value.(Node), ko))
			}
		}
		if node.(List).Len() > 1 {
			if chgok.pos {
				chgko.pos = true
			}
			if chgok.thPos {
				chgko.thPos = true
			}
		}
	default:
		chgko, chgok = compile(node, ko)
	}
	return
}

type chgFlags struct {
	pos, thPos bool
}

func updateChgFlags(ko, ok, newko, newok chgFlags) (chgFlags, chgFlags) {
	if newko.pos {
		ko.pos = true
	}
	if newko.thPos {
		ko.thPos = true
	}
	if newok.pos {
		ok.pos = true
	}
	if newok.thPos {
		ok.thPos = true
	}
	return ko, ok
}

type writer struct {
	io.Writer
	indent      int
	nLabels     int
	dryRun      bool
	savedIndent int
	saveFlags   []saveFlags
	elimRestore bool
}

type saveFlags struct {
	pos, thPos bool
}

func newWriter(out io.Writer) *writer {
	return &writer{Writer: out, indent: 2}
}

func (w *writer) begin() {
	w.lnPrint("{")
	w.indent++
}

func (w *writer) end() {
	w.indent--
	w.lnPrint("}")
}

func (w *writer) setDry(on bool) {
	w.dryRun = on
	if on {
		w.savedIndent = w.indent
	} else {
		w.indent = w.savedIndent
		w.nLabels = 0
	}
}

type label struct {
	id, sid int
	*writer
	used           bool
	savedBlockOpen bool
}

func (w *writer) newLabel() *label {
	i := w.nLabels
	w.nLabels++
	if w.dryRun {
		w.saveFlags = append(w.saveFlags, saveFlags{})
	}
	return &label{id: i, sid: i, writer: w}
}

func (w *label) label() {
	w.indent--
	w.lnPrint("l%d:", w.id)
	w.indent++
}

func (w *label) jump() {
	w.lnPrint("goto l%d", w.id)
	w.used = true
}

func (w *label) saveBlock() {
	save := w.saveFlags[w.id]
	if save.pos || save.thPos {
		w.begin()
		w.save()
		w.savedBlockOpen = true
	}
}
func (w *label) save() {
	save := w.saveFlags[w.id]
	switch {
	case save.pos && save.thPos:
		w.lnPrint("position%d, thunkPosition%d := position, thunkPosition", w.sid, w.sid)
	case !save.pos && save.thPos:
		w.lnPrint("thunkPosition%d := thunkPosition", w.sid)
	case save.pos:
		w.lnPrint("position%d := position", w.sid)
	}
}

func (w *label) unsafe() bool {
	save := w.saveFlags[w.id]
	return save.pos || save.thPos
}

func (w *label) restore(savePos, saveThPos bool) {
	w.lrestore(w, savePos, saveThPos)
}
func (w *label) lrestore(label *label, savePos, saveThPos bool) {
	if label != nil {
		if label.used {
			label.label()
		}
	}
	if !w.elimRestore {
		savePos = true
		saveThPos = true
	}
	switch {
	case savePos && saveThPos:
		w.lnPrint("position, thunkPosition = position%d, thunkPosition%d", w.sid, w.sid)
	case !savePos && saveThPos:
		w.lnPrint("thunkPosition = thunkPosition%d", w.sid)
		stats.elimRestore.pos++
	case savePos:
		w.lnPrint("position = position%d", w.sid)
		stats.elimRestore.thunkPos++
	default:
		stats.elimRestore.thunkPos++
		stats.elimRestore.pos++
	}
	if w.dryRun {
		save := &w.saveFlags[w.id]
		if !save.pos {
			save.pos = savePos
		}
		if !save.thPos {
			save.thPos = saveThPos
		}
	}
	if w.savedBlockOpen {
		w.end()
		w.savedBlockOpen = false
	}
}

func (w *label) cJump(jumpIfTrue bool, format string, a ...interface{}) {
	w.used = true
	if w.dryRun {
		return
	}
	if jumpIfTrue {
		format = "if " + format
	} else {
		format = "if !" + format
	}
	w.lnPrint(format, a...)
	fmt.Fprint(w, " {")
	w.lnPrint("\tgoto l%d", w.id)
	w.lnPrint("}")
}

func (w *writer) lnPrint(format string, a ...interface{}) {
	if w.dryRun {
		return
	}
	s := "\n"
	for i := 0; i < w.indent; i++ {
		s += "\t"
	}
	fmt.Fprintf(w, s+format, a...)
}

type statValues struct {
	Peek, Match struct {
		Char, Class, Dot, String int
	}
	elimRestore struct {
		pos, thunkPos int
	}
	optFirst struct {
		char, dot, str, class int
	}
	seqIfNot    int
	inlineLeafs int
}

var stats statValues
