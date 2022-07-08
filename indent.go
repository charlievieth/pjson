// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pjson

import (
	"bufio"
	"bytes"
)

// Compact appends to dst the JSON-encoded src with
// insignificant space characters elided.
func Compact(dst *bytes.Buffer, src []byte) error {
	return compact(dst, src, false)
}

func compact(dst *bytes.Buffer, src []byte, escape bool) error {
	origLen := dst.Len()
	scan := newScanner()
	defer freeScanner(scan)
	start := 0
	for i, c := range src {
		if escape && (c == '<' || c == '>' || c == '&') {
			if start < i {
				dst.Write(src[start:i])
			}
			dst.WriteString(`\u00`)
			dst.WriteByte(hex[c>>4])
			dst.WriteByte(hex[c&0xF])
			start = i + 1
		}
		// Convert U+2028 and U+2029 (E2 80 A8 and E2 80 A9).
		if escape && c == 0xE2 && i+2 < len(src) && src[i+1] == 0x80 && src[i+2]&^1 == 0xA8 {
			if start < i {
				dst.Write(src[start:i])
			}
			dst.WriteString(`\u202`)
			dst.WriteByte(hex[src[i+2]&0xF])
			start = i + 3
		}
		v := scan.step(scan, c)
		if v >= ScanSkipSpace {
			if v == ScanError {
				break
			}
			if start < i {
				dst.Write(src[start:i])
			}
			start = i + 1
		}
	}
	if scan.EOF() == ScanError {
		dst.Truncate(origLen)
		return scan.err
	}
	if start < len(src) {
		dst.Write(src[start:])
	}
	return nil
}

const (
	_s     = "                                                                " // 64
	spaces = _s + _s + _s + _s + _s + _s + _s + _s                              // 512

	// TOOD: remove tabs if not used
	_t   = "\t\t\t\t\t\t\t\t"                    // 8
	tabs = _t + _t + _t + _t + _t + _t + _t + _t // 64
)

// type indentMode int8
//
// const (
// 	indentMixed indentMode = iota
// 	indentSpaces
// 	indentTabs
// )

func newline(dst *bytes.Buffer, prefix, indent string, depth int, allSpaces bool) {
	dst.WriteByte('\n')
	if len(prefix) != 0 {
		dst.WriteString(prefix)
	}
	if allSpaces {
		n := len(indent) * depth
		for n > 0 {
			i := n
			if i >= len(spaces) {
				i = len(spaces)
			}
			dst.WriteString(spaces[:i])
			n -= i
		}
		return
	}
	for i := 0; i < depth; i++ {
		dst.WriteString(indent)
	}
}

// TODO: use an interface for this
func newlineBufio(dst *bufio.Writer, prefix, indent string, depth int, allSpaces bool) {
	dst.WriteByte('\n')
	if len(prefix) != 0 {
		dst.WriteString(prefix)
	}
	if allSpaces {
		n := len(indent) * depth
		for n > 0 {
			i := n
			if i >= len(spaces) {
				i = len(spaces)
			}
			dst.WriteString(spaces[:i])
			n -= i
		}
		return
	}
	for i := 0; i < depth; i++ {
		dst.WriteString(indent)
	}
}

// Indent appends to dst an indented form of the JSON-encoded src.
// Each element in a JSON object or array begins on a new,
// indented line beginning with prefix followed by one or more
// copies of indent according to the indentation nesting.
// The data appended to dst does not begin with the prefix nor
// any indentation, to make it easier to embed inside other formatted JSON data.
// Although leading space characters (space, tab, carriage return, newline)
// at the beginning of src are dropped, trailing space characters
// at the end of src are preserved and copied to dst.
// For example, if src has no trailing spaces, neither will dst;
// if src ends in a trailing newline, so will dst.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	origLen := dst.Len()
	scan := newScanner()
	defer freeScanner(scan)
	needIndent := false
	depth := 0
	for _, c := range src {
		scan.bytes++
		v := scan.step(scan, c)
		if v == ScanSkipSpace {
			continue
		}
		if v == ScanError {
			break
		}
		if needIndent && v != ScanEndObject && v != ScanEndArray {
			needIndent = false
			depth++
			newline(dst, prefix, indent, depth, false)
		}

		// Emit semantically uninteresting bytes
		// (in particular, punctuation in strings) unmodified.
		if v == ScanContinue {
			dst.WriteByte(c)
			continue
		}

		// Add spacing around real punctuation.
		switch c {
		case '{', '[':
			// delay indent so that empty object and array are formatted as {} and [].
			needIndent = true
			dst.WriteByte(c)

		case ',':
			dst.WriteByte(c)
			newline(dst, prefix, indent, depth, false)

		case ':':
			dst.WriteByte(c)
			dst.WriteByte(' ')

		case '}', ']':
			if needIndent {
				// suppress indent in empty object/array
				needIndent = false
			} else {
				depth--
				newline(dst, prefix, indent, depth, false)
			}
			dst.WriteByte(c)

		default:
			dst.WriteByte(c)
		}
	}
	if scan.EOF() == ScanError {
		dst.Truncate(origLen)
		return scan.err
	}
	return nil
}
