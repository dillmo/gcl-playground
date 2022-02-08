/*
 * GCL Playground - an HTML pretty-printer for GCL
 * Copyright (C) 2022  Dillon Morse
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/* Language spec
 * <EXPR> -> <ASSIGN> <EXPR'>
 *         | skip <EXPR'>
 * <EXPR'> -> ; <EXPR>
 *          | <EMPTY>
 * <ASSIGN> -> <ID> <ASSIGN'> <MATH>
 * <ASSIGN'> -> , <ID> <ASSIGN'> <MATH> ,
 *            | :=
 * <MATH> -> <ID> <MATH'>
 * <MATH'> -> + <MATH>
 *          | <EMPTY>
 */

package main

import (
  "io"
  "fmt"
  "strings"
  "syscall/js"
)

// Token constants
const (
  ID = iota
  PLUS
  GETS
  COMMA
  SEMICOLON
  SKIP
  ERROR
)

type Token struct {
  Type int
  Lexeme string
}

type Lexer struct {
  in io.RuneReader
  // These two fields are used to rewind by one token when
  // necessary
  stream []*Token
  pos int
  // These two fields are used to rewind by one rune when
  // necessary
  lastRune rune
  repeatRune bool
}

func NewLexer(r io.RuneReader) *Lexer {
  return &Lexer{in: r, pos: 0}
}

// Return the next token, error at end of input
func (l *Lexer) Next() (*Token, error) {
  // Return the same token after the client rewinds
  if l.pos < len(l.stream) {
    token := l.stream[l.pos]
    l.pos++
    return token, nil
  }

  r, err := l.nextRune()
  // nextRune errors when it runs out of runes to read
  if err != nil {
    return nil, fmt.Errorf("no more tokens")
  }

  var token *Token
  switch {
  // Plus sign
  case r == '+':
    token, err = &Token{Type: PLUS}, nil
  // Identifiers are single alphabetical runes
  case ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z'):
    token, err = l.lexIDOrKeyword(r), nil
  // Skip whitespace
  case r == ' ' || r == '\n' || r == '\r':
    token, err = l.Next()
  // Lex rest of := command
  case r == ':':
    token, err = l.lexGets()
  // Comma
  case r == ',':
    token, err = &Token{Type: COMMA}, nil
  // Semicolon
  case r == ';':
    token, err = &Token{Type: SEMICOLON}, nil
  // Not a recognized token
  default:
    token, err = &Token{Type: ERROR}, nil
  }
  l.stream = append(l.stream, token)
  l.pos++
  return token, err
}

// Rewind the lexer by one token
func (l *Lexer) Rewind() {
  if l.pos > 0 {
    l.pos--
  }
}

// Rewind the lexer's reader by one rune
func (l *Lexer) rewindRune(lastRune rune) {
  l.lastRune = lastRune
  l.repeatRune = true
}

// Return the next rune in the input sequence
func (l *Lexer) nextRune() (rune, error) {
  // If we recently rewound by one rune, return the last
  // rune
  if l.repeatRune {
    l.repeatRune = false
    return l.lastRune, nil
  } else {
    r, _, err := l.in.ReadRune()
    // ReadRune errors when it runs out of runes to read
    return r, err
  }
}

// Lex the rest of an ID or keyword after the first rune
func (l *Lexer) lexIDOrKeyword(firstChar rune) *Token {
  var builder strings.Builder
  builder.WriteRune(firstChar)
  r, err := l.nextRune()
  // r is always an identifier rune (a-z, A-Z) inside this
  // loop
  for (('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')) && err == nil {
    builder.WriteRune(r)
    r, err = l.nextRune()
  }
  // Either err = nil or we looked ahead to a non-ID rune
  // and need to rewind
  if err == nil {
    l.rewindRune(r)
  }
  str := builder.String()
  // Check if str is a keyword before returning
  switch str {
  case "skip":
    return &Token{Type: SKIP}
  default:
    return &Token{Type: ID, Lexeme: str}
  }
}

// Lex the '=' rune in a ':=' token
func (l *Lexer) lexGets() (*Token, error) {
  r, err := l.nextRune()
  // nextRune errors when it runs out of runes to read
  if err != nil {
    return nil, fmt.Errorf("no more tokens")
  }
  // The next rune should be '='
  if r == '=' {
    return &Token{Type: GETS}, nil
  }
  // This is not a recognized token
  return &Token{Type: ERROR}, nil
}

// Parser is a recursive-descent parser
type Parser struct {
  lexer *Lexer
}

func NewParser(l *Lexer) *Parser {
  return &Parser{lexer: l}
}

// Top-level parse function
func (p *Parser) Parse() (string, error) {
  // Parse as much input as we can
  result, err := p.expr()

  // Tack any errors onto what we parsed
  if err != nil {
    return result, err
  }

  // There should be no tokens left
  nextToken, _ := p.lexer.Next()
  if nextToken != nil {
    return result, fmt.Errorf("unexpected token")
  }

  return result, err
}

// Parse an <EXPR> nonterminal
func (p *Parser) expr() (string, error) {
  // We can determine which type of expression to parse by
  // reading the next token
  nextToken, err := p.lexer.Next()
  // If we hit the end of input, then let the <ASSIGN>
  // parser write the error message
  if err != nil {
    return p.assign()
  }
  
  var result string
  switch nextToken.Type {
  // If the next token is an ID, then this is an assignment
  // command
  case ID:
    // Rewind the lexer for the next parser call
    p.lexer.Rewind()
    result, err = p.assign()
  // If the next token is "skip", then this is a skip
  // command
  case SKIP:
    result, err = "<strong>skip</strong>", nil
  // If the next token is neither, then the input is invalid
  default:
    result, err = "", fmt.Errorf("unexpected token")
  }
  // Tack any errors onto the rest of the expression
  if err != nil {
    return result, err
  }

  // Parse the rest of the expression
  rest, err := p.exprP()
  if err != nil {
    return result + rest, err
  }

  return result + rest, err
}

// Parse an <EXPR'> nonterminal
func (p *Parser) exprP() (string, error) {
  token, err := p.lexer.Next()
  // lexer errors when there is no more input
  if err != nil {
    return "", nil
  }
  // The next token should be a semicolon
  if token.Type != SEMICOLON {
    // Rewind the lexer so the next function gets this token
    p.lexer.Rewind()
    return "", nil
  }
  rest, err := p.expr()
  // Tack on any errors to what we've already parsed
  return fmt.Sprintf("; <br />\n%s", rest), err
}

// Parse an <ASSIGN> nonterminal
func (p *Parser) assign() (string, error) {
  token, err := p.lexer.Next()
  // lexer errors when there is no more input
  // All assign expressions start with an identifier
  if err != nil || token.Type != ID {
    return "", fmt.Errorf("expected a variable")
  }
  // We have an identifier. Now try to parse the middle of
  // the expression.
  middle, err := p.assignP()
  // Tack what we already have onto any errors
  if err != nil {
    return fmt.Sprintf("\\(%s %s\\)", token.Lexeme, middle), err
  }
  // Last portion should be a math nonterminal
  math, err := p.math()
  return fmt.Sprintf("\\(%s %s %s\\)", token.Lexeme, middle, math), err
}

// Parse an <ASSIGN'> nonterminal
func (p *Parser) assignP() (string, error) {
  token, err := p.lexer.Next()
  // lexer errors when there is no more input
  if err != nil {
    return "", fmt.Errorf("expected ',' or ':='")
  }

  switch token.Type {
  // The expression continues to recurse
  case COMMA:
    token, err = p.lexer.Next()
    // Next token should be an identifier
    if err != nil || token.Type != ID {
      return "", fmt.Errorf("expected a variable")
    }
    id := token.Lexeme
    // Recurse
    middle, err := p.assignP()
    // Tack what we already have onto any errors
    if err != nil {
      return fmt.Sprintf(", %s %s", id, middle), err
    }
    // Next portion should be a <MATH> nonterminal
    math, err := p.math()
    // Tack what we already have onto any errors
    if err != nil {
      return fmt.Sprintf(", %s %s %s", id, middle, math), err
    }
    // Last token should be a ','
    token, err = p.lexer.Next()
    if err != nil {
      return fmt.Sprintf(", %s %s %s", id, middle, math), fmt.Errorf("expected ','")
    }
    return fmt.Sprintf(", %s %s %s, ", id, middle, math), nil
  // ':=' is the base case for recursion
  case GETS:
    return " \\coloneqq ", nil
  default:
    return "", fmt.Errorf("expected ',' or ':='")
  }
}

// Parse a <MATH> nonterminal
func (p *Parser) math() (string, error) {
  token, err := p.lexer.Next()
  // lexer errors when there is no more input
  // All math expressions start with an identifier
  if err != nil || token.Type != ID {
    return "", fmt.Errorf("expected a variable")
  }
  // We have an identifier. Now try to parse the rest of the
  // expression.
  next, err := p.mathP()
  // If there was an error, we tack it onto what we have
  // already parsed.
  return token.Lexeme + next, err
}

// Parse a <MATH'> nonterminal
func (p *Parser) mathP() (string, error) {
  token, err := p.lexer.Next()
  // lexer errors when there is no more input
  if err != nil {
    return "", nil
  }
  // Math expressions are always continued with plus signs
  if token.Type != PLUS {
    // Rewind the lexer so the next function gets this token
    p.lexer.Rewind()
    return "", nil
  }
  // We have a plus sign. Now try to parse the rest of the
  // expression.
  next, err := p.math()
  // If there was an error, we tack it onto what we have
  // already parsed.
  return " + " + next, err
}

// Compile a Guarded Command Language string into typeset
// HTML.
func CompileStr(str string) string {
  strReader := strings.NewReader(str)
  lexer := NewLexer(strReader)
  parser := NewParser(lexer)

  out, err := parser.Parse()
  if err != nil {
    // If there was an error, we still display what we
    // managed to parse so the user can more easily locate
    // their mistake.
    return fmt.Sprintf("<p>%s %s</p>", out, err)
  }

  return fmt.Sprintf("<p>%s</p>", out)
}

// Wrap the CompileStr function for JavaScript to use via
// WebAssembly.
func compileStrWrapper() js.Func {
  return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
    if len(args) != 1 {
      return "<p>Error parsing input.</p>"
    } else {
      return CompileStr(args[0].String())
    }
  })
}

func main() {
  // Inject CompileStr into the global JavaScript
  // environment via WebAssembly.
  js.Global().Set("compileStr", compileStrWrapper())
  // We cause the program to hang so the CompileStr function
  // does not get cleaned up.
  select {}
}
