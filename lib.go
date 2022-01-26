package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// placeholder/reserved is invisible for user
// use it internally
const (
	placeholder = iota + 1
	reserved
	String
	Boolean
	Number
	List
)

type (
	stack []Token
	tokenizer struct {
		Column, Line, Length, Index int
		Input string
	}
	Token struct{
		List []Token
		Boolean bool
		Number float64
		String string
		Type int
	}
)

var precedence = map[string]int {
	"+": 3,
	"-": 3,
	"*": 4,
	"/": 4,
	"and": 1,
	"matches": 1,
	"or": 1,
	"in": 2,
	">": 2,
	">=": 2,
	"==": 2,
	"!=": 2,
	"<=": 2,
	"<": 2,
}

func Interpret(v string) (interface{}, error) {
	c := &tokenizer{
		Index: -1,
		Length: len(v),
		Input: v,
		Line: 1,
	}
	if v, failure := c.tokenize(abort); failure != nil {
		return nil, failure
	} else {
		return result(v)
	}
}

func InterpretWith(v string, patch func(string) (Token, error)) (interface{}, error) {
	c := &tokenizer{
		Index: -1,
		Length: len(v),
		Input: v,
		Line: 1,
	}
	if v, failure := c.tokenize(abort); failure != nil {
		return nil, failure
	} else {
		for i, x := range v {
			if x.Type == placeholder {
				if x, failure := patch(x.String); failure != nil {
					return nil, failure
				} else {
					v[i] = x
				}
			}
		}
		return result(v)
	}
}

func (s *stack) reduce(o Token) (e error) {
	c, rv := last(*s)
	c, lv := last(c)
	*s = c
	require := func(v int, myth interface{}) {
		if v != lv.Type || v != rv.Type {
			e = fmt.Errorf("Data type mismatch in criteria expression (operator: %s).", o.String)
		} else {
			switch v := myth.(type) {
				case float64: *s = append(*s, Token{ Type: Number, Number: v })
				case bool: *s = append(*s, Token{ Type: Boolean, Boolean: v })
				case string: *s = append(*s, Token{ Type: String, String: v })
				case func(): v()
			}
		}
	}
	switch o.String {
		case ">": require(Number, lv.Number > rv.Number)
		case ">=": require(Number, lv.Number >= rv.Number)
		case "<=": require(Number, lv.Number <= rv.Number)
		case "<": require(Number, lv.Number < rv.Number)
		case "!=": *s = append(*s, Token{ Type: Boolean, Boolean: equalsTo(lv, rv) == false })
		case "==": *s = append(*s, Token{ Type: Boolean, Boolean: equalsTo(lv, rv) })
		case "+": require(Number, lv.Number + rv.Number)
		case "-": require(Number, lv.Number - rv.Number)
		case "*": require(Number, lv.Number * rv.Number)
		case "/": require(Number, lv.Number / rv.Number)
		case "and": require(Boolean, lv.Boolean && rv.Boolean)
		case "or": require(Boolean, lv.Boolean || rv.Boolean)
		case "matches": {
			require(String, func() {
				if v, e := regexp.MatchString(rv.String, lv.String); e == nil {
					*s = append(*s, Token{
						Type: Boolean,
						Boolean: v,
					}) 
				}
			})
		}
		// Is given operand exists in list?
		case "in": {
			if rv.Type != List {
				e = fmt.Errorf("Data type mismatch in criteria expression (operator: %s).", o.String)
			} else {
				var z bool
				for _, x := range rv.List {
					if equalsTo(x, lv) {
						z = true
						break
					}
				}
				*s = append(*s, Token{ Type: Boolean, Boolean: z })
			}
		}
	}
	return
}

func (t *tokenizer) tokenize(stop func(*tokenizer) bool) (list []Token, failure error) {
	for {
		c := t.next()
		if c > 0 {
			var v Token
			switch c {
				case '(',')','+','-','*','/': v = Token{ String: string(c), Type: reserved }
				case '"': v, failure = t.string()
				case '>','<','!','=': v, failure = t.comparator()
				case '[': v, failure = t.array()
				default: {
					f64 := digit(c)
					if f64 == false && c == '-' {
						i := t.Index + 1
						if i < t.Length && digit(t.Input[i]) {
							f64 = true
						}
					}
					// Start with division sign & follow a digit (its negative number)
					// otherwise parse as reserved
					if f64 == false { 
						v, failure = t.reserved(), nil
					} else {
						v, failure = t.float64()
					}
				}
			}
			if failure == nil {
				list = append(list, v)
			}
		}
		// break when Index >= Length or doesn't delimit by some character
		// exception occurred yet
		if failure != nil || stop(t) {
			break
		}
	}
	return
}

func (t *tokenizer) next() byte {
	for t.Index++; t.Index < t.Length; t.Index++ {
		if v := t.Input[t.Index]; v == '\n' {
			t.Column = 1
			t.Line++
		} else {
			t.Column += 1
			if v != '\x20' && v != '\x0a' && v != '\x0d' && v != '\x09' {
				return v
			}
		}
	}
	return 0
}

func (t *tokenizer) float64() (Token, error) {
	i := t.Index + 1
	for ; i < t.Length; i++ {
		v := t.Input[i]
		if digit(v) == false && v != '.' && v != 'e' {
			break
		}
	}
	if v, e := strconv.ParseFloat(t.Input[t.Index : i], 32); e == nil {
		t.Index = i-1
		return Token{
			Type: Number,
			Number: v,
		}, nil
	}
	return Token{}, fmt.Errorf(
		"You have an error in your expression, " +
		"check the manual that corresponds to your library version for the right syntax to use, " +
		"near (column: %d, line: %d)",
		t.Column,
		t.Line,
	)
}

func (t *tokenizer) comparator() (Token, error) {
	s := t.Index
	// Determime >=, <=, !=, ==
	if i := t.Index + 1; i < t.Length && t.Input[i] == '=' {
		t.Index++
	}
	k := t.Input[s : t.Index + 1]
	// legality check
	switch k {
		case ">",">=","<","<=","!=","==": {
			return Token{
				Type: reserved,
				String: k,
			}, nil
		}
		default: {
			// exclamation or equal sign isn't valid comparator
			return Token{}, fmt.Errorf(
				"You have an error in your expression, " +
				"check the manual that corresponds to your library version for the right syntax to use, " +
				"near (column: %d, line: %d)",
				t.Column,
				t.Line,
			)
		}
	}
}

func (t *tokenizer) string() (v Token, e error) {
	n := t.Index + 1
	for {
		i := strings.IndexByte(t.Input[n:], '"')
		// Can't pair by quote
		if i == -1 {
			break
		}
		n += i
		if t.Input[n - 1] != '\\' {
			break
		}
	}
	if n < t.Length && t.Input[n] == '"' {
		v, t.Index = Token{ Type: String, String: t.Input[t.Index + 1: n] }, n
		return
	}
	return v, fmt.Errorf(
		"You have an error in your expression, " +
		"check the manual that corresponds to your library version for the right syntax to use, " +
		"near (column: %d, line: %d)",
		t.Column,
		t.Line,
	)
}

func (t *tokenizer) reserved() Token {
	n := -1
	recognize := func(n int) Token {
		v := t.Input[t.Index : n]
		t.Index = n - 1
		switch v {
			case "false": return Token{ Type: Boolean, Boolean: false }
			case "in","matches","and","or": return Token{ Type: reserved, String: v }
			case "true": return Token{ Type: Boolean, Boolean: true }
			default: {
				return Token{
					Type: placeholder,
					String: v,
				}
			}
		}
	}
	for v,x := []byte{' ','"','(',')','+','-','*','/','>','=','!','<','['}, t.Index + 1; x < t.Length; x++ {
		n = bytes.IndexByte(v, t.Input[x])
		if n != -1 {
			return recognize(x)
		}
	}
	// recognized as placeholder if we can't index n of delimiter
	return recognize(t.Length)
}

func (t *tokenizer) array() (Token, error) {
	if v, err := t.tokenize(func(t *tokenizer) bool { return t.next() != ',' }); err != nil {
		return Token{}, err
	} else if t.Index < t.Length && t.Input[t.Index] == ']' {
		return Token{ Type: List, List: v}, nil
	} else {
		return Token{}, fmt.Errorf(
			"You have an error in your expression," +
			"check the manual that corresponds to your library version for the right syntax to use," +
			"near (column: %d, line: %d)",
			t.Column,
			t.Line,
		)
	}
}

func abort(t *tokenizer) bool {
	return t.Index >= t.Length
}

func valueOf(i Token) (interface{}, error) {
	switch i.Type {
		case Number: return i.Number, nil
		case Boolean: return i.Boolean, nil
		case String: return i.String, nil
		default: {
			return nil, fmt.Errorf(``)
		}
	}
}

func digit(v byte) bool {
	return v >= '0' && v <= '9'
}

func result(s []Token) (r interface{}, e error) {
	var (
		operator = make([]Token, 0)
		value = make(stack, 0)
		i Token
	)
	for _, v := range s {
		// Operand (string, regular expression etc...) append to cache always
		if v.Type != reserved {
			value = append(value, v)
		} else {
			switch v.String {
				case "(": operator = append(operator, v)
				case ")": {
					for operator, i = last(operator); i.Type > 0 && i.String != "("; operator, i = last(operator) {
						e = value.reduce(i)
						if e != nil {
							return
						}
					}
				}
				default: {
					// calculate lower priority
					for c, i := last(operator); i.Type > 0 && precedence[i.String] > precedence[v.String]; c, i = last(operator) {
						e, operator = value.reduce(i), c
						if e != nil {
							return
						}
					}
					operator = append(
						operator,
						v,
					)
				}
			}
		}
	}
	if len(operator) > 0 {
		for operator, i = last(operator); i.Type > 0; operator, i = last(operator) {
			e = value.reduce(i)
			if e != nil {
				return
			}
		}
	}
	return valueOf(
		value[0],
	)
}

func equalsTo(l Token, r Token) bool {
	switch l.Type {
		case Number: return r.Type == Number && l.Number == r.Number
		case Boolean: return r.Type == Boolean && l.Boolean == r.Boolean
		case String: return r.Type == String && l.String == r.String
		case List: {
			v := r.Type == List && len(l.List) == len(r.List)
			if v {
				for i := range r.List {
					v = equalsTo(l.List[i], r.List[i]) && v
				}
			}
			return v
		}
	}
	return false
}

func last(s stack) (stack, Token) {
	if 0 < len(s) {
		return s[:len(s)-1], s[len(s)-1]
	} else {
		return s, Token{}
	}
}