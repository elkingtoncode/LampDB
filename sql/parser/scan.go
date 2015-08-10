// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Peter Mattis (peter@cockroachlabs.com)

package parser

// TODO(pmattis):
//   - Add support for hexadecimal integer literals. Perhaps octal (0o) and
//     binary (0b) as well.

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const eof = -1
const errUnterminated = "unterminated string"
const errUnsupportedEscape = "octal, hex and unicode escape not supported"

type scanner struct {
	in        string
	pos       int
	tokBuf    sqlSymType
	lastTok   sqlSymType
	nextTok   *sqlSymType
	lastError string
	stmts     []Statement
}

func newScanner(s string) *scanner {
	return &scanner{in: s}
}

func (s *scanner) Lex(lval *sqlSymType) int {
	// The core lexing takes place in scan(). Here we do a small bit of post
	// processing of the lexical tokens so that the grammar only requires
	// one-token lookahead despite SQL requiring multi-token lookahead in some
	// cases. These special cases are handled below and the returned tokens are
	// adjusted to reflect the lookahead (LA) that occurred.

	if s.nextTok != nil {
		*lval = *s.nextTok
		s.nextTok = nil
	} else {
		s.scan(lval)
	}

	switch lval.id {
	case NOT, NULLS, WITH:
	default:
		s.lastTok = *lval
		return lval.id
	}

	s.nextTok = &s.tokBuf
	s.scan(s.nextTok)

	switch lval.id {
	case NOT:
		switch s.nextTok.id {
		case BETWEEN, IN, LIKE, SIMILAR:
			lval.id = NOT_LA
		}

	case NULLS:
		switch s.nextTok.id {
		case FIRST, LAST:
			lval.id = NULLS_LA
		}

	case WITH:
		switch s.nextTok.id {
		case TIME, ORDINALITY:
			lval.id = WITH_LA
		}
	}

	s.lastTok = *lval
	return lval.id
}

func (s *scanner) Error(e string) {
	var buf bytes.Buffer
	if s.lastTok.id == ERROR {
		fmt.Fprintf(&buf, "%s", s.lastTok.str)
	} else {
		fmt.Fprintf(&buf, "%s at or near \"%s\"", e, s.lastTok.str)
	}

	// Find the end of the line containing the last token.
	i := strings.Index(s.in[s.lastTok.pos:], "\n")
	if i == -1 {
		i = len(s.in)
	} else {
		i += s.lastTok.pos
	}
	// Find the beginning of the line containing the last token. Note that
	// LastIndex returns -1 if "\n" could not be found.
	j := strings.LastIndex(s.in[:s.lastTok.pos], "\n") + 1
	// Output everything up to and including the line containing the last token.
	fmt.Fprintf(&buf, "\n%s\n", s.in[:i])
	// Output a caret indicating where the last token starts.
	fmt.Fprintf(&buf, "%s^\n", strings.Repeat(" ", s.lastTok.pos-j))

	s.lastError = buf.String()
}

func (s *scanner) scan(lval *sqlSymType) {
	lval.id = 0
	lval.pos = s.pos
	lval.str = "EOF"

	if _, ok := s.skipWhitespace(lval, true); !ok {
		return
	}

	ch := s.next()
	if ch == eof {
		lval.pos = s.pos
		return
	}

	lval.id = int(ch)
	lval.pos = s.pos - 1
	lval.str = s.in[lval.pos:s.pos]

	switch ch {
	case '$':
		// param? $[0-9]+
		if isDigit(s.peek()) {
			s.scanParam(lval)
			return
		}
		return

	case '"':
		// "[^"]"{whitespace}*
		if s.scanString(lval, '"', false) {
			lval.id = IDENT
		}
		return

	case '\'':
		// '[^']'{whitespace}*
		if s.scanString(lval, '\'', false) {
			lval.id = SCONST
		}
		return

	case 'b', 'B':
		// Bit string?
		if s.peek() == '\'' {
			// [bB]'[^']'{whitespace}*
			s.pos++
			if s.scanString(lval, '\'', false) {
				lval.id = BCONST
			}
			return
		}
		s.scanIdent(lval, ch)
		return

	case 'e', 'E':
		// Escaped string?
		if s.peek() == '\'' {
			// [eE]'[^']'{whitespace}*
			s.pos++
			if s.scanString(lval, '\'', true) {
				lval.id = SCONST
			}
			return
		}
		s.scanIdent(lval, ch)
		return

	case 'x', 'X':
		// Hexadecimal string?
		if s.peek() == '\'' {
			// [xX]'[^']'{whitespace}*
			s.pos++
			if s.scanString(lval, '\'', false) {
				lval.id = XCONST
			}
			return
		}
		s.scanIdent(lval, ch)
		return

	case '.':
		switch t := s.peek(); {
		case t == '.': // ..
			s.pos++
			lval.id = DOT_DOT
			return
		case isDigit(t):
			s.scanNumber(lval, ch)
			return
		}
		return

	case '!':
		switch s.peek() {
		case '=': // !=
			s.pos++
			lval.id = NOT_EQUALS
			return
		}
		return

	case '<':
		switch s.peek() {
		case '>': // <>
			s.pos++
			lval.id = NOT_EQUALS
			return
		case '=': // <=
			s.pos++
			lval.id = LESS_EQUALS
			return
		}
		return

	case '>':
		switch s.peek() {
		case '=': // >=
			s.pos++
			lval.id = GREATER_EQUALS
			return
		}
		return

	case ':':
		switch s.peek() {
		case ':': // ::
			s.pos++
			lval.id = TYPECAST
			return
		}
		return

	case '|':
		switch s.peek() {
		case '|': // ||
			s.pos++
			lval.id = CONCAT
			return
		}
		return

	default:
		if isDigit(ch) {
			s.scanNumber(lval, ch)
			return
		}
		if isIdentStart(ch) {
			s.scanIdent(lval, ch)
			return
		}
	}

	// Everything else is a single character token which we already initialized
	// lval for above.
}

func (s *scanner) peek() int {
	if s.pos >= len(s.in) {
		return eof
	}
	return int(s.in[s.pos])
}

func (s *scanner) next() int {
	ch := s.peek()
	if ch != eof {
		s.pos++
	}
	return ch
}

func (s *scanner) skipWhitespace(lval *sqlSymType, allowComments bool) (newline, ok bool) {
	newline = false
	for {
		ch := s.peek()
		if ch == '\n' {
			s.pos++
			newline = true
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\f' {
			s.pos++
			continue
		}
		if allowComments {
			if present, cok := s.scanComment(lval); !cok {
				return false, false
			} else if present {
				continue
			}
		}
		break
	}
	return newline, true
}

func (s *scanner) scanComment(lval *sqlSymType) (present, ok bool) {
	start := s.pos
	ch := s.peek()

	if ch == '/' {
		s.pos++
		if s.peek() != '*' {
			s.pos--
			return false, true
		}
		s.pos++
		depth := 1
		for {
			switch s.next() {
			case '*':
				if s.peek() == '/' {
					s.pos++
					depth--
					if depth == 0 {
						return true, true
					}
					continue
				}

			case '/':
				if s.peek() == '*' {
					s.pos++
					depth++
					continue
				}

			case eof:
				lval.id = ERROR
				lval.pos = start
				lval.str = "unterminated comment"
				return false, false
			}
		}
	}

	if ch == '-' {
		s.pos++
		if s.peek() != '-' {
			s.pos--
			return false, true
		}
		for {
			switch s.next() {
			case eof, '\n':
				return true, true
			}
		}
	}

	return false, true
}

func (s *scanner) scanIdent(lval *sqlSymType, ch int) {
	start := s.pos - 1
	for {
		ch := s.peek()
		if isIdentMiddle(ch) {
			s.pos++
			continue
		}
		break
	}
	lval.str = s.in[start:s.pos]
	uppered := strings.ToUpper(lval.str)
	if id, ok := keywords[uppered]; ok {
		lval.id = id
		return
	}
	lval.id = IDENT
}

func (s *scanner) scanNumber(lval *sqlSymType, ch int) {
	start := s.pos - 1
	hasDecimal := ch == '.'
	hasExponent := false

	for {
		ch := s.peek()
		if isDigit(ch) {
			s.pos++
			continue
		}
		if ch == '.' {
			if hasDecimal || hasExponent {
				break
			}
			s.pos++
			if s.peek() == '.' {
				// Found ".." while scanning a number: back up to the end of the
				// integer.
				s.pos--
				break
			}
			hasDecimal = true
			continue
		}
		if ch == 'e' || ch == 'E' {
			if hasExponent {
				break
			}
			hasExponent = true
			s.pos++
			ch = s.peek()
			if ch == '-' || ch == '+' {
				s.pos++
			}
			ch = s.peek()
			if !isDigit(ch) {
				lval.id = ERROR
				lval.str = "invalid floating point constant"
				return
			}
			continue
		}
		break
	}

	lval.str = s.in[start:s.pos]
	if hasDecimal || hasExponent {
		lval.id = FCONST
		return
	}

	var err error
	lval.ival, err = strconv.Atoi(lval.str)
	if err != nil {
		lval.id = ERROR
		lval.str = err.Error()
		return
	}

	lval.id = ICONST
}

func (s *scanner) scanParam(lval *sqlSymType) {
	start := s.pos
	for isDigit(s.peek()) {
		s.pos++
	}
	lval.str = s.in[start:s.pos]

	var err error
	lval.ival, err = strconv.Atoi(lval.str)
	if err != nil {
		lval.id = ERROR
		lval.str = err.Error()
		return
	}

	lval.id = PARAM
}

func (s *scanner) scanString(lval *sqlSymType, ch int, allowEscapes bool) bool {
	lval.str = ""
	start := s.pos
	for {
		switch s.next() {
		case ch:
			lval.str += s.in[start : s.pos-1]
			if s.peek() == ch {
				// Double quote is translated into a single quote that is part of the
				// string.
				start = s.pos
				s.pos++
				continue
			}

			if newline, ok := s.skipWhitespace(lval, false); !ok {
				return false
			} else if !newline {
				return true
			}
			// SQL allows joining adjacent strings separated by whitespace as long as
			// that whitespace contains at least one newline. Kind of strange to
			// require the newline, but that is the standard.
			if s.peek() != ch {
				return true
			}
			s.pos++
			start = s.pos
			continue

		case '\\':
			t := s.peek()
			if allowEscapes {
				lval.str += s.in[start : s.pos-1]
				if t == ch || t == '\\' {
					start = s.pos
					s.pos++
					continue
				}

				switch t {
				// TODO(pmattis): Handle other back-slash escapes? Octal? Hexadecimal?
				// Unicode?
				case 'b', 'f', 'n', 'r', 't', '\'':
					lval.str += string(decodeMap[byte(t)])
					s.pos++
					start = s.pos
					continue
				case 'x', 'u', 'U':
					fallthrough
				case '0', '1', '2', '3', '4', '5', '6', '7':
					lval.id = ERROR
					lval.str = errUnsupportedEscape
					return false
				}

				// If we end up here, it's a redundant escape - simply drop the
				// backslash. For example, e'\"' is equivalent to e'"', and
				// e'\a\b' to e'a\b'. This is what Postgres does:
				// http://www.postgresql.org/docs/9.4/static/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS-ESCAPE
				start = s.pos
			}

		case eof:
			lval.id = ERROR
			lval.str = errUnterminated
			return false
		}
	}
}

func isDigit(ch int) bool {
	return ch >= '0' && ch <= '9'
}

func isIdent(s string) bool {
	if len(s) == 0 || !isIdentStart(int(s[0])) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isIdentMiddle(int(s[i])) {
			return false
		}
	}
	return true
}

func isIdentStart(ch int) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 128 && ch <= 255) ||
		(ch == '_')
}

func isIdentMiddle(ch int) bool {
	return isIdentStart(ch) || isDigit(ch) || ch == '$'
}
