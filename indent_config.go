package pjson

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/charlievieth/pjson/termcolor"
)

// TODO: make this just the color config
type IndentConfig struct {
	Null        *termcolor.Color
	False       *termcolor.Color
	True        *termcolor.Color
	Keyword     *termcolor.Color
	Quote       *termcolor.Color // WARN: unused
	String      *termcolor.Color
	Numeric     *termcolor.Color
	Punctuation *termcolor.Color
	// TODO: remove this
	// ConvertUnicode bool            // print escaped unicode
}

// var noColor = termcolor.NoColor{}

// func colorOr(c termcolor.Color) termcolor.Color {
// 	if c != nil {
// 		return c
// 	}
// 	return noColor
// }

// func (c *IndentConfig) init() IndentConfig {
// 	if c == nil {
// 		c = &DefaultIndentConfig
// 	}
// 	return IndentConfig{
// 		Null:        colorOr(c.Null),
// 		False:       colorOr(c.False),
// 		True:        colorOr(c.True),
// 		Keyword:     colorOr(c.Keyword),
// 		Quote:       colorOr(c.Quote),
// 		String:      colorOr(c.String),
// 		Numeric:     colorOr(c.Numeric),
// 		Punctuation: colorOr(c.Punctuation),
// 	}
// }

var DefaultIndentConfig = IndentConfig{
	Null:        termcolor.Yellow, // WARN
	False:       termcolor.Yellow,
	True:        termcolor.Yellow,
	Keyword:     termcolor.Blue,
	Quote:       termcolor.Green, // WARN
	String:      termcolor.Green,
	Numeric:     termcolor.Magenta,
	Punctuation: termcolor.Yellow,
}

// JQIndentConfig matches the default color scheme of `jq`
// (https://stedolan.github.io/jq/).
var JQIndentConfig = IndentConfig{
	Null:        termcolor.BrightBlack,
	False:       termcolor.BrightWhite,
	True:        termcolor.BrightWhite,
	Keyword:     termcolor.Blue,
	Quote:       termcolor.Green,
	String:      termcolor.Green,
	Numeric:     termcolor.White,
	Punctuation: termcolor.White,
}

func NewIndentConfig() *IndentConfig {
	return nil
}

type byteStringWriter interface {
	io.ByteWriter
	io.StringWriter
}

func writeByte(dst byteStringWriter, color *termcolor.Color, ch byte) {
	dst.WriteString(color.Format())
	dst.WriteByte(ch)
	dst.WriteString(color.Reset())
}

// func writeByteBufio(dst *bufio.Writer, color *termcolor.Color, ch byte) {
// 	dst.WriteString(color.Format())
// 	dst.WriteByte(ch)
// 	dst.WriteString(color.Reset())
// }

var bufioReaderPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReader(nil)
	},
}

var bufioWriterPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewWriter(nil)
	},
}

func freeBufioScanner(w *bufio.Writer, r *bufio.Reader, s *Scanner) {
	w.Reset(nil) // remove reference
	r.Reset(nil) // remove reference
	bufioWriterPool.Put(w)
	bufioReaderPool.Put(r)
	freeScanner(s)
}

func newBuffers(wr io.Writer, rd io.Reader) (*bufio.Writer, *bufio.Reader) {
	w := bufioWriterPool.Get().(*bufio.Writer)
	r := bufioReaderPool.Get().(*bufio.Reader)
	w.Reset(wr)
	r.Reset(rd)
	return w, r
}

func isAllSpaces(indent string) bool {
	for i := 0; i < len(indent); i++ {
		if indent[i] != ' ' {
			return false
		}
	}
	return true
}

// WARN WARN WARN
//
// Disallow multiple JSON on the same line: `{}{}`
//
// WARN WARN WARN
func (conf *IndentConfig) IndentStream(wr io.Writer, rd io.Reader, prefix, indent string) error {
	dst, r := newBuffers(wr, rd)
	scan := newScanner()
	defer freeBufioScanner(dst, r, scan)

	allSpaces := isAllSpaces(indent)
	needIndent := false
	depth := 0
	var resetBytes int64
	var err error
	for {
		var c byte
		c, err = r.ReadByte()
		if err != nil {
			break
		}
		v := scan.Step(c)
		if v == ScanSkipSpace {
			continue
		}
		if v == ScanError {
			break
		}
		// WARN: we should change this to read one JSON value at a time
		// WARN: try to read the last byte early
		// TODO: should probably flush here
		if v == ScanEnd && scan.EndTop() {
			scan.Reset()
			resetBytes = scan.Bytes()
			dst.WriteByte('\n')
			// c = '\n' // WARN
			continue
		}
		if needIndent && v != ScanEndObject && v != ScanEndArray {
			needIndent = false
			depth++
			newlineBufio(dst, prefix, indent, depth, allSpaces)
		}
		var clr *termcolor.Color
		if v == ScanBeginLiteral {
			switch scan.CurrentParseState() {
			case ParseObjectKey:
				// TODO: do we want to use different quote colors here?
				clr = conf.Keyword
			case ParseObjectValue, ParseArrayValue:
				// TODO: use Quote color
				switch c {
				case '"':
					clr = conf.String
				case 'n':
					clr = conf.Null
				case 't':
					clr = conf.True
				case 'f':
					clr = conf.False
				default:
					clr = conf.Numeric
				}

				// TODO: error here ???
				// default:
				// 	err = errors.New("pjson: invalid parse state")
				// 	break
			}

			// Instead of reading/writing byte-by-byte use the
			// bytes the Reader already has buffered.
			dst.WriteString(clr.Format())
			dst.WriteByte(c)
		InnerLoop:
			for {
				n := r.Buffered()
				if n <= 0 {
					n = 1 // trigger a re-fill
				}
				b, e := r.Peek(n)
				if e != nil && e != bufio.ErrBufferFull {
					err = e
					break
				}
				for i := 0; i < len(b); i++ {
					c = b[i]
					v = scan.Step(c)
					if v != ScanContinue {
						dst.Write(b[:i])
						r.Discard(i + 1)
						break InnerLoop
					}
				}
				dst.Write(b)
				r.Discard(len(b))
			}
			// Check error from InnerLoop
			if err != nil && err != bufio.ErrBufferFull {
				break
			}
			// NOTE: we check some, but not all write errors since
			// once the bufio.Writer encounters an error it will
			// always return it.
			if _, err = dst.WriteString(clr.Reset()); err != nil {
				break
			}
			if v == ScanSkipSpace {
				continue
			}
		}

		// Add spacing around real punctuation.
		switch c {
		case '{', '[':
			// delay indent so that empty object and array are formatted as {} and [].
			needIndent = true
			writeByte(dst, conf.Punctuation, c)

		case ',':
			writeByte(dst, conf.Punctuation, c)
			newlineBufio(dst, prefix, indent, depth, allSpaces)

		case ':':
			writeByte(dst, conf.Punctuation, c)
			dst.WriteByte(' ')

		case '}', ']':
			if needIndent {
				// suppress indent in empty object/array
				needIndent = false
			} else {
				depth--
				newlineBufio(dst, prefix, indent, depth, allSpaces)
			}
			writeByte(dst, conf.Punctuation, c)

		default:
			dst.WriteByte(c)
		}
	}

	// Flush before checking for read/scan errors
	ferr := dst.Flush()

	if err != nil && err != io.EOF {
		return err // TODO: wrap this error
	}

	// TODO: we can / should just check if the scan is empty
	//
	// TODO: return both scan and write errors?
	if scan.EOF() == ScanError {
		// Check if we just reset the scanner
		if resetBytes == 0 || scan.Bytes() != resetBytes {
			// dst.Flush() // WARN WARN WARN
			return scan.Err()
		}
	}
	// Return Flush error, if any
	if ferr != nil {
		return ferr
	}
	return nil
}

func (conf *IndentConfig) Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	origLen := dst.Len()
	scan := newScanner()
	defer freeScanner(scan)

	allSpaces := isAllSpaces(indent)
	needIndent := false
	depth := 0
	for i := 0; i < len(src); i++ {
		c := src[i]
		v := scan.Step(c)
		// leave here for debugging
		if false {
			fmt.Printf("'%c' %s\n", c, ScanStateString(v))
			fmt.Printf("    %s\n", scan.parseState)
		}
		if v == ScanSkipSpace {
			continue
		}
		if v == ScanError {
			break
		}
		if needIndent && v != ScanEndObject && v != ScanEndArray {
			needIndent = false
			depth++
			newline(dst, prefix, indent, depth, allSpaces)
		}
		var clr *termcolor.Color
		// var quote *termcolor.Color
		if v == ScanBeginLiteral {
			switch scan.CurrentParseState() {
			case ParseObjectKey:
				clr = conf.Keyword
				// WARN: quote handling
				// if !conf.Quote.IsZero() && !clr.Equal(conf.Quote) {
				// 	quote = conf.Quote
				// }
			case ParseObjectValue, ParseArrayValue:
				// TODO: use Quote color
				switch c {
				case '"':
					clr = conf.String
					// WARN: quote handling
					// if !conf.Quote.IsZero() && !clr.Equal(conf.Quote) {
					// 	quote = conf.Quote
					// }
				case 'n':
					clr = conf.Null
				case 't':
					clr = conf.True
				case 'f':
					clr = conf.False
				default:
					clr = conf.Numeric
				}
			}
			// if quote != nil {
			// 	writeByte(dst, quote, c)
			// 	i++
			// 	if i >= len(src) {
			// 		break // TODO: handle
			// 	}
			// 	c = src[i]
			// 	v = scan.Step(c)
			// }
			j := i
			for i++; i < len(src); i++ {
				c = src[i]
				v = scan.Step(c)
				if v != ScanContinue {
					break
				}
			}
			dst.WriteString(clr.Format())
			dst.Write(src[j:i])
			dst.WriteString(clr.Reset())
			// WARN: quote handling
			// if quote != nil {
			// 	dst.Write(src[j : i-1])
			// 	dst.WriteString(clr.Reset())
			// 	writeByte(dst, quote, '"')
			// } else {
			// 	dst.Write(src[j:i])
			// 	dst.WriteString(clr.Reset())
			// }
			if v == ScanSkipSpace {
				continue
			}
		}

		// Add spacing around real punctuation.
		switch c {
		case '{', '[':
			// delay indent so that empty object and array are formatted as {} and [].
			needIndent = true
			writeByte(dst, conf.Punctuation, c)

		case ',':
			writeByte(dst, conf.Punctuation, c)
			newline(dst, prefix, indent, depth, allSpaces)

		case ':':
			writeByte(dst, conf.Punctuation, c)
			dst.WriteByte(' ')

		case '}', ']':
			if needIndent {
				// suppress indent in empty object/array
				needIndent = false
			} else {
				depth--
				newline(dst, prefix, indent, depth, allSpaces)
			}
			writeByte(dst, conf.Punctuation, c)

		default:
			dst.WriteByte(c)
		}
	}
	if scan.EOF() == ScanError {
		dst.Truncate(origLen)
		return scan.Err()
	}
	return nil
}

func (conf *IndentConfig) CompactStream(wr io.Writer, rd io.Reader) error {
	dst, r := newBuffers(wr, rd)
	scan := newScanner()
	defer freeBufioScanner(dst, r, scan)

	var err error
	for {
		var c byte
		c, err = r.ReadByte()
		if err != nil {
			break
		}
		v := scan.Step(c)
		// leave here for debugging
		if false {
			fmt.Printf("'%c' %s\n", c, ScanStateString(v))
			fmt.Printf("    %s\n", scan.parseState)
		}
		if v == ScanSkipSpace {
			continue
		}
		if v == ScanError {
			break
		}
		if v == ScanBeginLiteral {
			var clr *termcolor.Color
			switch scan.CurrentParseState() {
			case ParseObjectKey:
				// TODO: do we want to use different quote colors here?
				clr = conf.Keyword
			case ParseObjectValue, ParseArrayValue:
				// TODO: use Quote color
				switch c {
				case '"':
					clr = conf.String
				case 'n':
					clr = conf.Null
				case 't':
					clr = conf.True
				case 'f':
					clr = conf.False
				default:
					clr = conf.Numeric
				}
			}
			// Instead of reading/writing byte-by-byte use the
			// bytes the Reader already has buffered.
			dst.WriteString(clr.Format())
			dst.WriteByte(c)
		InnerLoop:
			for {
				n := r.Buffered()
				if n <= 0 {
					n = 1 // trigger a re-fill
				}
				var b []byte
				b, err = r.Peek(n)
				if err != nil && err != bufio.ErrBufferFull {
					break
				}
				for i := 0; i < len(b); i++ {
					c = b[i]
					v = scan.Step(c)
					if v != ScanContinue {
						dst.Write(b[:i])
						r.Discard(i + 1)
						break InnerLoop
					}
				}
				dst.Write(b)
				r.Discard(len(b))
			}
			// Check error from InnerLoop
			if err != nil && err != bufio.ErrBufferFull {
				break
			}
			// NOTE: we check some, but not all write errors since
			// once the bufio.Writer encounters an error it will
			// always return it.
			if _, err = dst.WriteString(clr.Reset()); err != nil {
				break
			}
			if v == ScanSkipSpace {
				continue
			}
		}

		// Colorize punctuation.
		switch c {
		case '{', '[', ',', ':', '}', ']':
			// delay indent so that empty object and array are formatted as {} and [].
			writeByte(dst, conf.Punctuation, c)
		default:
			dst.WriteByte(c)
		}
	}

	if err != nil && err != io.EOF {
		return err // TODO: wrap this error
	}
	// TODO: return both scan and write errors?
	if scan.EOF() == ScanError {
		return scan.err
	}
	if err := dst.Flush(); err != nil {
		return err
	}
	return nil
}

func (conf *IndentConfig) Compact(dst *bytes.Buffer, src []byte) error {
	origLen := dst.Len()
	scan := newScanner()
	defer freeScanner(scan)

	for i := 0; i < len(src); i++ {
		c := src[i]
		v := scan.Step(c)
		// leave here for debugging
		if false {
			fmt.Printf("'%c' %s\n", c, ScanStateString(v))
			fmt.Printf("    %s\n", scan.parseState)
		}
		if v == ScanSkipSpace {
			continue
		}
		if v == ScanError {
			break
		}
		if v == ScanBeginLiteral {
			var clr *termcolor.Color
			switch scan.CurrentParseState() {
			case ParseObjectKey:
				// TODO: do we want to use different quote colors here?
				clr = conf.Keyword
			case ParseObjectValue, ParseArrayValue:
				// TODO: use Quote color
				switch c {
				case '"':
					clr = conf.String
				case 'n':
					clr = conf.Null
				case 't':
					clr = conf.True
				case 'f':
					clr = conf.False
				default:
					clr = conf.Numeric
				}
			}
			j := i
			for i++; i < len(src); i++ {
				c = src[i]
				v = scan.Step(c)
				if v != ScanContinue {
					break
				}
			}
			dst.WriteString(clr.Format())
			dst.Write(src[j:i])
			dst.WriteString(clr.Reset())
			if v == ScanSkipSpace {
				continue
			}
		}

		// Colorize punctuation.
		switch c {
		case '{', '[', ',', ':', '}', ']':
			// delay indent so that empty object and array are formatted as {} and [].
			writeByte(dst, conf.Punctuation, c)
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

type Stream struct {
	// WARN: just use an io.Reader
	r *bufio.Reader // TODO: lazily setup Reader?

	scan    *Scanner // TODO: don't use a pointer
	conf    *IndentConfig
	buf     []byte
	scanp   int   // start of unread data in buf
	scanned int64 // amount of data already scanned
	scratch bytes.Buffer
	indent  string
	prefix  string
	newline string // WARN: use or remove
	err     error
}

// TODO: swap arg positions
func NewStream(rd io.Reader, conf *IndentConfig) *Stream {
	// r := bufioReaderPool.Get().(*bufio.Reader)
	// r.Reset(rd)
	dupe := *conf
	return &Stream{
		r:       bufio.NewReader(rd),
		scan:    newScanner(),
		conf:    &dupe,
		newline: "\n",
	}
}

func (s *Stream) SetConfig(conf *IndentConfig) {
	if conf == nil {
		panic("pjson: nil IndentConfig")
	}
	if s.conf == nil {
		s.conf = new(IndentConfig)
	}
	*s.conf = *conf
}

func (s *Stream) SetIndent(prefix, indent string) {
	s.prefix = prefix
	s.indent = indent
}

func (s *Stream) SetNewline(newline string) {
	s.newline = newline
}

func (s *Stream) WriteTo(wr io.Writer) (int64, error) {
	panic("implement")
}

func (dec *Stream) refill() error {
	// Make room to read more into the buffer.
	// First slide down data already consumed.
	if dec.scanp > 0 {
		dec.scanned += int64(dec.scanp)
		n := copy(dec.buf, dec.buf[dec.scanp:])
		dec.buf = dec.buf[:n]
		dec.scanp = 0
	}

	// Grow buffer if not large enough.
	const minRead = 512
	if cap(dec.buf)-len(dec.buf) < minRead {
		newBuf := make([]byte, len(dec.buf), 2*cap(dec.buf)+minRead)
		copy(newBuf, dec.buf)
		dec.buf = newBuf
	}

	// Read. Delay error for next iteration (after scan).
	n, err := dec.r.Read(dec.buf[len(dec.buf):cap(dec.buf)])
	dec.buf = dec.buf[0 : len(dec.buf)+n]

	return err
}

// readValue reads a JSON value into dec.buf.
// It returns the length of the encoding.
func (dec *Stream) readValue() (int, error) {
	dec.scan.Reset()

	scanp := dec.scanp
	var err error
Input:
	// help the compiler see that scanp is never negative, so it can remove
	// some bounds checks below.
	for scanp >= 0 {

		// Look in the buffer for a new value.
		for ; scanp < len(dec.buf); scanp++ {
			c := dec.buf[scanp]
			dec.scan.bytes++
			switch dec.scan.step(dec.scan, c) {
			case ScanEnd:
				// scanEnd is delayed one byte so we decrement
				// the scanner bytes count by 1 to ensure that
				// this value is correct in the next call of Decode.
				dec.scan.bytes--
				break Input
			case ScanEndObject, ScanEndArray:
				// scanEnd is delayed one byte.
				// We might block trying to get that byte from src,
				// so instead invent a space byte.
				if stateEndValue(dec.scan, ' ') == ScanEnd {
					scanp++
					break Input
				}
			case ScanError:
				dec.err = dec.scan.err
				return 0, dec.scan.err
			}
		}

		// Did the last read have an error?
		// Delayed until now to allow buffer scan.
		if err != nil {
			if err == io.EOF {
				if dec.scan.step(dec.scan, ' ') == ScanEnd {
					break Input
				}
				if nonSpace(dec.buf) {
					err = io.ErrUnexpectedEOF
				}
			}
			dec.err = err
			return 0, err
		}

		n := scanp - dec.scanp
		err = dec.refill()
		scanp = dec.scanp + n
	}
	return scanp - dec.scanp, nil
}

// WARN: rename
func (s *Stream) Next() ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}

	// WARN WARN WARN WARN WARN WARN WARN
	// if err := dec.tokenPrepareForDecode(); err != nil {
	// 	return err
	// }
	// WARN WARN WARN WARN WARN WARN WARN

	n, err := s.readValue()
	if err != nil {
		return nil, err
	}
	val := s.buf[s.scanp : s.scanp+n]
	s.scanp += n

	s.scratch.Reset()
	if err := s.conf.Indent(&s.scratch, val, s.prefix, s.indent); err != nil {
		// panic(fmt.Sprintf("error: %v n: %d scanp: %d\n###\n%q\n###", err, n, s.scanp, val))
		return nil, err
	}
	s.scratch.WriteByte('\n')
	out := make([]byte, s.scratch.Len())
	copy(out, s.scratch.Bytes())
	return out, nil
}

func (s *Stream) Indent(wr io.Writer) (int, error) {
	if s.err != nil {
		return 0, s.err
	}

	var nn int // WARN: use an int64
	for i := 0; ; i++ {
		b, err := s.Next()
		if err != nil {
			return nn, err
		}
		n, err := wr.Write(b)
		nn += n
		if err != nil {
			return nn, err
		}
		if i > 1000 {
			panic("WAT")
		}
	}
	return nn, nil
}

// WARN: make sure we return io.EOF
// WARN: what is the right signature for this?
func (s *Stream) IndentOld(wr io.Writer) (int, error) {
	if s.err != nil {
		return 0, s.err
	}

	r := s.r
	dst := bufioWriterPool.Get().(*bufio.Writer)
	dst.Reset(wr)
	scan := s.scan
	scan.Reset()
	// TODO: support compact
	// prefix := s.prefix
	// indent := s.indent
	conf := s.conf

	allSpaces := isAllSpaces(s.indent)
	needIndent := false
	depth := 0
	var resetBytes int64
	var err error
	for {
		var c byte
		c, err = r.ReadByte()
		if err != nil {
			break
		}
		v := scan.Step(c)
		if v == ScanSkipSpace {
			continue
		}
		if v == ScanError {
			break
		}
		// // WARN: we should change this to read one JSON value at a time
		// // WARN: try to read the last byte early
		// // TODO: should probably flush here
		// if v == ScanEnd && scan.EndTop() {
		// 	scan.reset()
		// 	resetBytes = scan.bytes
		// 	c = '\n' // WARN
		// }
		if needIndent && v != ScanEndObject && v != ScanEndArray {
			needIndent = false
			depth++
			newlineBufio(dst, s.prefix, s.indent, depth, allSpaces)
		}
		var clr *termcolor.Color
		if v == ScanBeginLiteral {
			switch scan.CurrentParseState() {
			case ParseObjectKey:
				// TODO: do we want to use different quote colors here?
				clr = conf.Keyword
			case ParseObjectValue, ParseArrayValue:
				// TODO: use Quote color
				switch c {
				case '"':
					clr = conf.String
				case 'n':
					clr = conf.Null
				case 't':
					clr = conf.True
				case 'f':
					clr = conf.False
				default:
					clr = conf.Numeric
				}

				// TODO: error here ???
				// default:
				// 	err = errors.New("pjson: invalid parse state")
				// 	break
			}

			// Instead of reading/writing byte-by-byte use the
			// bytes the Reader already has buffered.
			dst.WriteString(clr.Format())
			dst.WriteByte(c)
		InnerLoop:
			for {
				n := r.Buffered()
				if n <= 0 {
					n = 1 // trigger a re-fill
				}
				b, e := r.Peek(n)
				if e != nil && e != bufio.ErrBufferFull {
					err = e
					break
				}
				for i := 0; i < len(b); i++ {
					c = b[i]
					v = scan.Step(c)
					if v != ScanContinue {
						dst.Write(b[:i])
						r.Discard(i + 1)
						break InnerLoop
					}
				}
				dst.Write(b)
				r.Discard(len(b))
			}
			// Check error from InnerLoop
			if err != nil && err != bufio.ErrBufferFull {
				break
			}
			// NOTE: we check some, but not all write errors since
			// once the bufio.Writer encounters an error it will
			// always return it.
			if _, err = dst.WriteString(clr.Reset()); err != nil {
				break
			}
			if v == ScanSkipSpace {
				continue
			}
		}

		// Add spacing around real punctuation.
		switch c {
		case '{', '[':
			// delay indent so that empty object and array are formatted as {} and [].
			needIndent = true
			writeByte(dst, conf.Punctuation, c)

		case ',':
			writeByte(dst, conf.Punctuation, c)
			newlineBufio(dst, s.prefix, s.indent, depth, allSpaces)

		case ':':
			writeByte(dst, conf.Punctuation, c)
			dst.WriteByte(' ')

		case '}', ']':
			if needIndent {
				// suppress indent in empty object/array
				needIndent = false
			} else {
				depth--
				newlineBufio(dst, s.prefix, s.indent, depth, allSpaces)
			}
			writeByte(dst, conf.Punctuation, c)

		default:
			dst.WriteByte(c)
		}
	}

	// Flush before checking for read/scan errors
	ferr := dst.Flush()

	if err != nil && err != io.EOF {
		s.err = err
		return 0, s.err // TODO: wrap this error
	}

	// TODO: return both scan and write errors?
	if scan.EOF() == ScanError {
		// Check if we just reset the scanner
		if resetBytes == 0 || scan.Bytes() != resetBytes {
			// dst.Flush() // WARN WARN WARN
			s.err = scan.err
			return 0, s.err
		}
	}
	// Return Flush error, if any
	if ferr != nil {
		s.err = ferr
		return 0, s.err
	}
	return 0, nil
}

func (s *Stream) Close() error {
	if s.r != nil {
		s.r.Reset(nil)
		bufioReaderPool.Put(s.r)
		s.r = nil
	}
	if s.scan != nil {
		freeScanner(s.scan)
		s.scan = nil
	}
	// TODO: error if already closed?
	if s.err != nil {
		return s.err
	}
	return nil
}

func RGBToAnsi256(r, g, b int) int {
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return int(math.Round(((float64(r)-8)/247)*24)) + 232
	}
	ansi := 16 + (36 * math.Round(float64(r)/255*5)) +
		(6 * math.Round(float64(g)/255*5)) + math.Round(float64(b)/255*5)
	return int(ansi)
}

// type bufioWriter struct {
// 	*bufio.Writer
// 	pooled bool
// }
//
// type bufioReader struct {
// 	*bufio.Reader
// 	pooled bool
// }
//
// func freeBufioScanner(w bufioWriter, r bufioReader, s *Scanner) {
// 	if w.pooled {
// 		w.Reset(nil) // remove reference
// 		bufioWriterPool.Put(w.Writer)
// 	}
// 	if r.pooled {
// 		r.Reset(nil) // remove reference
// 		bufioReaderPool.Put(r.Reader)
// 	}
// 	freeScanner(s)
// }
//
// func newBuffers(wr io.Writer, rd io.Reader) (bufioWriter, bufioReader) {
// 	// TODO: support using bufio.{Reader,Writer} if provided
// 	bw, wok := wr.(*bufio.Writer)
// 	if !wok {
// 		bw = bufioWriterPool.Get().(*bufio.Writer)
// 		bw.Reset(wr)
// 	}
// 	br, rok := rd.(*bufio.Reader)
// 	if !rok {
// 		br = bufioReaderPool.Get().(*bufio.Reader)
// 		br.Reset(rd)
// 	}
// 	return bufioWriter{bw, wok}, bufioReader{br, rok}
// }

/*
func (conf *IndentConfig) Indent_OLD(dst *bytes.Buffer, src []byte) error {
	origLen := dst.Len()
	scan := newScanner()
	defer freeScanner(scan)
	needIndent := false
	needReset := false
	_ = needReset
	depth := 0
	for _, c := range src {
		scan.bytes++
		v := scan.step(scan, c)
		if false {
			// leave here for debugging
			fmt.Printf("'%c' %s\n", c, ScanStateString(v))
			fmt.Printf("    %s\n", scan.parseState)
		}
		if v == ScanSkipSpace {
			continue
		}
		if v == ScanError {
			break
		}
		if needIndent && v != ScanEndObject && v != ScanEndArray {
			needIndent = false
			depth++
			newline(dst, conf.PrefixString, conf.IndentString, depth)
		}
		if v == ScanBeginLiteral {
			switch scan.CurrentParseState() {
			case ParseObjectKey:
				// TODO: do we want to use different quote colors here?
				dst.WriteString(conf.Keyword.Format())
			case ParseObjectValue, ParseArrayValue:
				// WARN: need to check if the value is a string or not
				if c == '"' {
					dst.WriteString(conf.String.Format())
				} else {
					dst.WriteString(conf.Numeric.Format())
				}
			}
			needReset = true
		}
		if v == ScanObjectKey || v == ScanObjectValue || v == ScanArrayValue || v == ScanEndArray {
			dst.WriteString(termcolor.Reset)
			needReset = false
		}

		// Emit semantically uninteresting bytes
		// (in particular, punctuation in strings) unmodified.
		if v == ScanContinue {
			dst.WriteByte(c)
			continue
		}
		// if needReset {
		// 	dst.WriteString(termcolor.Reset)
		// 	needReset = false
		// }

		// Add spacing around real punctuation.
		switch c {
		case '{', '[':
			// delay indent so that empty object and array are formatted as {} and [].
			needIndent = true
			// dst.WriteByte(c)
			writeByte(dst, conf.Punctuation, c)

		case ',':
			// dst.WriteByte(c)
			writeByte(dst, conf.Punctuation, c)
			newline(dst, conf.PrefixString, conf.IndentString, depth)

		case ':':
			// dst.WriteByte(c)
			writeByte(dst, conf.Punctuation, c)
			dst.WriteByte(' ')

		case '}', ']':
			if needIndent {
				// suppress indent in empty object/array
				needIndent = false
			} else {
				depth--
				newline(dst, conf.PrefixString, conf.IndentString, depth)
			}
			// dst.WriteByte(c)
			writeByte(dst, conf.Punctuation, c)

		default:
			// fmt.Printf("D: '%c'\n", c)
			dst.WriteByte(c)
			// dst.WriteString(termcolor.Reset)
		}
	}
	if scan.EOF() == ScanError {
		dst.Truncate(origLen)
		return scan.err
	}
	return nil
}
*/

// stateBeginValueOrEmpty
// stateBeginValue
// stateBeginStringOrEmpty
// stateBeginString
// stateEndValue
// stateEndTop
// stateInString
// stateInStringEsc
// stateInStringEscU
// stateInStringEscU1
// stateInStringEscU12
// stateInStringEscU123
// stateNeg
// state1
// state0
// stateDot
// stateDot0
// stateE
// stateESign
// stateE0
// stateT
// stateTr
// stateTru
// stateF
// stateFa
// stateFal
// stateFals
// stateN
// stateNu
// stateNul
// stateError
