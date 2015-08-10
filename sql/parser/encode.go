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

// This code was derived from https://github.com/youtube/vitess.
//
// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file

package parser

import (
	"bytes"
	"fmt"
	"strings"
)

var (
	dontEscape = byte(255)
	// encodeMap specifies how to escape binary data with '\'.
	encodeMap [256]byte
	// decodeMap is the reverse of encodeMap
	decodeMap [256]byte
	hexMap    [256][]byte
)

func encodeSQLString(buf []byte, in []byte) []byte {
	// See http://www.postgresql.org/docs/9.4/static/sql-syntax-lexical.html
	start := 0
	for i, ch := range in {
		if encodedChar := encodeMap[ch]; encodedChar != dontEscape {
			if start == 0 {
				buf = append(buf, 'e', '\'') // begin e'xxx' string
			}
			buf = append(buf, in[start:i]...)
			buf = append(buf, '\\', encodedChar)
			start = i + 1
		}
	}
	if start == 0 {
		buf = append(buf, '\'') // begin 'xxx' string if nothing was escaped
	}
	buf = append(buf, in[start:]...)
	buf = append(buf, '\'')
	return buf
}

// TODO(pmattis): This method needs testing.
func encodeSQLIdent(buf *bytes.Buffer, s string) {
	if _, ok := keywords[strings.ToUpper(s)]; ok {
		fmt.Fprintf(buf, "\"%s\"", s)
		return
	}
	// The string needs quoting if it does not match the ident format.
	if isIdent(s) {
		_, _ = buf.WriteString(s)
		return
	}

	// The only character that requires escaping is a double quote.
	_ = buf.WriteByte('"')
	start := 0
	for i, n := 0, len(s); i < n; i++ {
		ch := s[i]
		if ch == '"' {
			if start != i {
				_, _ = buf.WriteString(s[start:i])
			}
			start = i + 1
			_ = buf.WriteByte(ch)
			_ = buf.WriteByte(ch) // add extra copy of ch
		}
	}
	if start < len(s) {
		_, _ = buf.WriteString(s[start:])
	}
	_ = buf.WriteByte('"')
}

// Convenience around encodeSQLIdent.
// TODO(tschottdorf): always use this? After all, performance not an issue.
func encIdent(s string) string {
	var buf bytes.Buffer
	encodeSQLIdent(&buf, s)
	return buf.String()
}

func encodeSQLBytes(buf []byte, v []byte) []byte {
	buf = append(buf, "x'"...)
	for _, d := range v {
		buf = append(buf, hexMap[d]...)
	}
	buf = append(buf, '\'')
	return buf
}

func init() {
	encodeRef := map[byte]byte{
		'\x00': '0',
		'\b':   'b',
		'\f':   'f',
		'\n':   'n',
		'\r':   'r',
		'\t':   't',
		'\\':   '\\',
		'\'':   '\'',
	}

	for i := range encodeMap {
		encodeMap[i] = dontEscape
		decodeMap[i] = dontEscape
	}
	for i := range encodeMap {
		if to, ok := encodeRef[byte(i)]; ok {
			encodeMap[byte(i)] = to
			decodeMap[to] = byte(i)
		}
	}

	for i := range hexMap {
		hexMap[i] = []byte(fmt.Sprintf("%02x", i))
	}
}
