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

package parse

import (
  "fmt"

  "github.com/dillmo/gcl-playground/src/lex"
)

// Parser is a recursive-descent parser
type Parser struct {
  lexer *lex.Lexer
}

func NewParser(l *lex.Lexer) *Parser {
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
  case lex.ID:
    // Rewind the lexer for the next parser call
    p.lexer.Rewind()
    result, err = p.assign()
  // If the next token is "skip", then this is a skip
  // command
  case lex.SKIP:
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
  if token.Type != lex.SEMICOLON {
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
  if err != nil || token.Type != lex.ID {
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
  case lex.COMMA:
    token, err = p.lexer.Next()
    // Next token should be an identifier
    if err != nil || token.Type != lex.ID {
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
  case lex.GETS:
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
  if err != nil || token.Type != lex.ID {
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
  if token.Type != lex.PLUS {
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

