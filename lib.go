package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// type of token
const (
  isString = 1
  isRegularExpression = 2
  isBoolean = 3
  isArray = 4
  isNumber = 5
  isReserved = 6
  isNull = 7
)

type (
  tokens []token
  tokenizer struct{
    Length, Index, Column, Line int
    Stream []rune
  }
  token struct{
    Array []token
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
  i := []rune(v)
  if c, failure := ready(i); failure != nil {
    return nil, failure
  } else if t, failure := c.tokenize(); failure != nil {
    return nil, failure
  } else if v, failure := result(t); failure != nil {
    return nil, failure
  } else {
    switch v.Type {
      case isNumber: return v.Number, nil
      case isBoolean: return v.Boolean, nil
      case isString: return v.String, nil
      default: {
        return nil, nil
      }
    }
  }
}

func result(c []token) (token, error) {
  var (
    v token
    keyword = make(tokens, 0)
    operand = make(tokens, 0)
    reduce = func(CanItContinue func(token) bool) error {
      var (
        v = true
        failure error
        k token
      )
      for v && failure == nil {
        k = keyword.pop()
        if v = k.Type != 0 && CanItContinue(k); v {
          failure = operand.reduce(k)
        }
      }
      return nil
    }
  )
  for i := range c {
    v = c[i]
    // Operand (string, regular expression etc...) append to cache always
    if v.Type != isReserved {
      operand = append(
        operand,
        v,
      )
    } else {
      switch v.String {
        case "(": keyword = append(keyword, v)
        case ")": {
          if failure := reduce(func(t token) bool { return t.String != "(" }); failure != nil {
            return token{}, failure
          }
        }
        // Operator is here
        default: {
          last := len(keyword) - 1
          if last > -1 && precedence[v.String] < precedence[keyword[last].String] {
            failure := reduce(func(t token) bool {
              return t.String != "(" && precedence[t.String] > precedence[v.String]
            })
            if failure != nil {
              return token{}, failure
            }
          }
          keyword = append(
            keyword,
            v,
          )
        }
      }
    }
  }
  if len(keyword) > 0 {
    if failure := reduce(func(t token) bool { return t.Type > 0 }); failure != nil {
      return token{}, failure
    }
  }
  return operand[0], nil
}

func isEquals(l token, r token) bool {
  switch l.Type {
    case isNumber: return r.Type == isNumber && l.Number == r.Number
    case isBoolean: return r.Type == isBoolean && l.Boolean == r.Boolean
    case isString: return r.Type == isString && l.String == r.String
    default: {
      return false
    }
  }
}

func isWhitespace(v rune) bool {
  return v == '\x20' || v == '\x0a' || v == '\x0d' || v == '\x09'
}

func isDigit(v rune) bool {
  return v >= '0' && v <= '9'
}

// Remove all comments from source code and construct as context
func ready(v []rune) (tokenizer, error) {
  column := 0
  line := 1
  if y := len(v)-1; y >= 0 {
    chunks, i := make([]rune, 0), 0
    for i <= y {
      // Increase line number & reset column number
      if '\x0a' == v[i] {
        column = 0
        line++
      }
      column++
      // First character of line or block comment are backslash both (regular expression also)
      if v[i] == '/' && i < y {
        // Here is comment
        if next := v[i + 1]; next == '/' || next == '*' {
          var (
            characterInComment rune
            fine = false
          )
          // Skip backslash or asterisk
          // because they are a part of openner of line/block comment
          i++
          // Move to next character
          for i < y && !fine {
            characterInComment, i = v[i + 1], i + 1
            if next != '*' {
              fine = characterInComment == '\n'
            } else {
              fine = characterInComment == '*' && v[i + 1] == '/'
              if fine {
                i++
              }
            }
          }
          // No more content
          if i == y && next != '*' /* LINE COMMENT */ {
            goto success
          }
          if !fine {
            goto failed
          }
          i++
        }
      }
      chunks = append(chunks, v[i])
      i++
    }
    success: {
      return tokenizer{
        Index: -1,
        Length: len(chunks),
        Stream: chunks,
        Line: 1,
      }, nil
    }
  }
  failed: {
    return tokenizer{}, fmt.Errorf(
      "You have an error in your expression, " +
      "check the manual that corresponds to your library version for the right syntax to use, " +
      "near (column: %d, line: %d)",
      column,
      line,
    )
  }
}

func (c *tokenizer) tokenize() (done []token, failure error) {
  var v token
  for {
    v, failure = c.value()
    if v.Type == 0 || failure != nil { break }
    done = append(
      done, 
      v,
    )
  }
  return
}

func (c *tokenizer) value() (token, error) {
  n := c.move()
  if n < 0 {
    return token{}, nil
  }
  digit := isDigit(n)
  // Start with division sign & follow a digit (its negative number)
  if !digit && n == '-' {
    i := c.Index + 1
    if i < c.Length && isDigit(c.Stream[i]) {
      digit = true
    }
  }
  if digit {
    return c.float64()
  }
  switch n {
    case 'i': return c.keyword("in")
    case 'a': return c.keyword("and")
    case 'm': return c.keyword("matches")
    case 'o': return c.keyword("or")
    case 't': return c.keyword("true")
    case 'f': return c.keyword("false")
    case 'n': return c.keyword("null")
    case '(',')','+','-','*': return token{ String: string(n), Type: isReserved }, nil
    case '"': return c.string()
    case '>','<','!','=': return c.comparator()
    case '[': return c.array()
    case '/': {
      division := false
      if c.Index > 0 {
        // Look previous character to find number (0-9)
        // recognize as division sign 
        // if successfully
        for x := c.Index - 1; x >= 0; x-- {
          if token := c.Stream[x]; !isWhitespace(token) {
            division = isDigit(token)
            break
          }
        }
      }
      if division {
        return token{ Type: isReserved, String: "/" }, nil
      } else {
        return c.regularExpression()
      }
    }
  }
  return token{}, fmt.Errorf(
    "You have an error in your expression, " +
    "check the manual that corresponds to your library version for the right syntax to use, " +
    "near (column: %d, line: %d)",
    c.Column,
    c.Line,
  )
}

func (c *tokenizer) float64() (token, error) {
  i := c.Index + 1
  v := '\x00'
  for ; i < c.Length; i++ {
    v = c.Stream[i]
    if v != '\x2e' && v != '\x2b' && v != '\x65' && !isDigit(v) {
      break
    }
  }
  if v, error := strconv.ParseFloat(string(c.Stream[c.Index : i]), 32); error == nil {
    c.Index = i - 1
    return token{
      Type: isNumber,
      Number: v,
    }, nil
  }
  return token{}, fmt.Errorf(
    "You have an error in your expression, " +
    "check the manual that corresponds to your library version for the right syntax to use, " +
    "near (column: %d, line: %d)",
    c.Column,
    c.Line,
  )
}

func (c *tokenizer) regularExpression() (token, error) {
  if n := c.Index + 1; n < c.Length {
    Start := n
    for n < c.Length && (c.Stream[n] != '/' || c.Stream[n - 1] == '\\') {
      n++
    }
    if '/' == c.Stream[n] {
      Ungreedy := false
      CaseInsensitive := false
      Multi := false
      // Match flags and rewrite to re2 syntax style
      c.Index = n
      for i := c.Index + 1; i < c.Length && (c.Stream[i] == 'g' || c.Stream[i] == 'i' || c.Stream[i] == 'm'); i++ {
        switch c.Stream[i] {
          case 'u': Ungreedy = true; c.Index++
          case 'i': CaseInsensitive = true; c.Index++
          case 'm': Multi = true; c.Index++
        }
      }
      ji := ""
      if CaseInsensitive || Multi || Ungreedy {
        ji += "(?"
        if Ungreedy { ji += "U" }
        if CaseInsensitive { ji += "i" }
        if Multi { ji += "m" }
        ji += ")"
      }
      return token{
        String: ji + string(c.Stream[ Start : n ]),
        Type: isRegularExpression,
      }, nil
    }
  }
  return token{}, fmt.Errorf(
    "You have an error in your expression, " +
    "check the manual that corresponds to your library version for the right syntax to use, " +
    "near (column: %d, line: %d)",
    c.Column,
    c.Line,
  )
}

func (c *tokenizer) string() (token, error) {
  if i := c.Index + 1; i < c.Length {
    Start := i
    for i < c.Length && (c.Stream[i] != '"' || c.Stream[i - 1] == '\\') {
      i++
    }
    if '"' == c.Stream[i] {
      c.Index = i // Point to string terminator
      return token{
        String: string(c.Stream[Start : i]),
        Type: isString,
      }, nil
    }
  }
  return token{}, fmt.Errorf(
    "You have an error in your expression, " +
    "check the manual that corresponds to your library version for the right syntax to use, " +
    "near (column: %d, line: %d)",
    c.Column,
    c.Line,
  )
}

func (c *tokenizer) comparator() (token, error) {
  shadow := c.Index
  // Determime: >=, <=, !=, ==
  if y := c.Index + 1; y < c.Length && c.Stream[y] == '=' {
    c.Index++
  }
  k := string(c.Stream[shadow : c.Index + 1])
  // legality check
  switch k {
    case ">",">=","<","<=","!=","==": return token{ Type: isReserved, String: k }, nil
    default: {
      // exclamation or equal sign isn't valid comparator
      return token{}, fmt.Errorf(
        "You have an error in your expression, " +
        "check the manual that corresponds to your library version for the right syntax to use, " +
        "near (column: %d, line: %d)",
        c.Column,
        c.Line,
      )
    }
  }
}

func (c *tokenizer) array() (token, error) {
  var (
    elements = make([]token, 0)
    failure error
    v token
  )
  for {
    if v, failure = c.value(); failure != nil || v.Type == 0 { break }
    elements = append(elements, v)
    if c.move() != ',' {
      break
    }
  }
  if c.Stream[c.Index] == ']' {
    return token{ Type: isArray, Array: elements }, nil
  }
  return token{}, fmt.Errorf(
    "You have an error in your expression," +
    "check the manual that corresponds to your library version for the right syntax to use," +
    "near (column: %d, line: %d)",
    c.Column,
    c.Line,
  )
}

func (c *tokenizer) keyword(k string) (token, error) {
  max := len(k) + c.Index
  if max <= c.Length && string(c.Stream[c.Index : max]) == k {
    c.Index = max - 1
    switch k {
      case "true": return token{ Type: isBoolean, Boolean: true }, nil
      case "false": return token{ Type: isBoolean, Boolean: false }, nil
      case "null": return token{ Type: isNull }, nil
    }
    return token{
      Type: isReserved,
      String: k,
    }, nil
  }
  return token{}, fmt.Errorf(
    "You have an error in your expression, " +
    "check the manual that corresponds to your library version for the right syntax to use, " +
    "near (column: %d, line: %d)",
    c.Column,
    c.Line,
  )
}

func (c *tokenizer) move() rune {
  c.Index++
  for c.Index < c.Length {
    if c.Stream[c.Index] == '\x0a' {
      c.Column = 0
      c.Line++
    }
    c.Column++
    if token := c.Stream[c.Index]; !isWhitespace(token) {
      return token
    } else {
      c.Index++
    }
  }
  return -1
}

func (c *tokens) reduce(o token) error {
  switch o.String {
    case "and","or": return c.logicConnect(o)
    case "+","-","*","/",">=","<=",">","<": return c.handleNumber(o)
    case "!=","==","in": return c.equals(o)
    case "matches": return c.isMatch()
    default: {
      return errors.New("")
    }
  }
}

// Operation for number (include arithmetic & comparision)
func (c *tokens) handleNumber(o token) error {
  r := c.pop()
  l := c.pop()
  if l.Type == isNumber && r.Type == isNumber {
    switch o.String {
      case ">" : *c = append(*c, token{ Type: isBoolean, Boolean: l.Number > r.Number })
      case "<" : *c = append(*c, token{ Type: isBoolean, Boolean: l.Number < r.Number })
      case ">=": *c = append(*c, token{ Type: isBoolean, Boolean: l.Number >= r.Number })
      case "<=": *c = append(*c, token{ Type: isBoolean, Boolean: l.Number <= r.Number })
      case "+" : *c = append(*c, token{ Type: isNumber, Number: l.Number + r.Number })
      case "-" : *c = append(*c, token{ Type: isNumber, Number: l.Number - r.Number })
      case "*" : *c = append(*c, token{ Type: isNumber, Number: l.Number * r.Number })
      case "/" : *c = append(*c, token{ Type: isNumber, Number: l.Number / r.Number })
      default: {
        goto FAILED
      }
    }
    return nil
  }
  // Failed to type assertion or not supported
  FAILED: {
    return errors.New("")
  }
}

func (c *tokens) isMatch() error {
  r := c.pop()
  l := c.pop()
  if l.Type > 0 && r.Type > 0 {
    var t token
    // Swap between lvalue & rvalue to ensure lvalue is scalar, rvalue is regular expression
    if l.Type == isRegularExpression {
      t = l
      l = r
      r = t
    }
    // Re2 syntax
    if s, f := regexp.MatchString(r.String, l.String); f == nil {
      *c = append(*c, token{
        Type: isBoolean,
        Boolean: s,
      })
      return nil
    }
  }
  return errors.New("")
}

func (c *tokens) logicConnect(o token) error {
  r := c.pop()
  l := c.pop()
  if l.Type == isBoolean && r.Type == isBoolean {
    switch o.String {
      case "and": *c = append(*c, token{ Type: isBoolean, Boolean: l.Boolean && r.Boolean })
      case "or" : *c = append(*c, token{ Type: isBoolean, Boolean: l.Boolean || r.Boolean })
      default: {
        goto FAILED
      }
    }
    return nil
  }
  FAILED: {
    return errors.New("")
  }
}

func (c *tokens) equals(o token) error {
  r := c.pop()
  l := c.pop()
  if l.Type > 0 && r.Type > 0 {
    var e bool
    switch o.String {
      case "!=": e = isEquals(l, r) == false
      case "==": e = isEquals(l, r)
      default: {
        // operator <in> requires rvalue is array
        if r.Type != isArray {
          goto FAILED
        }
        for i := range r.Array {
          e = isEquals(l, r.Array[i])
          if e {
            break
          }
        }		
      }
    }
    *c = append(*c, token{ 
      Type: isBoolean,
      Boolean: e,
    })
    return nil
  }
  FAILED: {
    return errors.New("")	
  }
}

func (c *tokens) pop() (o token) {
  y := len(*c) - 1
  if y >= 0 {
    o, *c = (*c)[y], (*c)[:y]
  }
  return
}