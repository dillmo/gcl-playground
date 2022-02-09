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

package lex

import (
  "io"
  "fmt"
  "strings"
)

// Token constants
const (
  ID = iota
  PLUS
  GETS
  COMMA
  SEMICOLON
  SKIP
  LBRACE
  RBRACE
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
  // Left curly brace
  case r == '{':
    token, err = &Token{Type: LBRACE}, nil
  // Right curly brace
  case r == '}':
    token, err = &Token{Type: RBRACE}, nil
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

