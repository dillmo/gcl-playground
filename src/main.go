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

package main

import (
  "fmt"
  "strings"
  "syscall/js"

  "github.com/dillmo/gcl-playground/src/lex"
  "github.com/dillmo/gcl-playground/src/parse"
)

// Compile a Guarded Command Language string into typeset
// HTML.
func CompileStr(str string) string {
  strReader := strings.NewReader(str)
  lexer := lex.NewLexer(strReader)
  parser := parse.NewParser(lexer)

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
