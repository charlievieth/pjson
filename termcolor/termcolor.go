package termcolor

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/term"
)

const Reset = "\x1b[0m"

var (
	// NoColor     = IsTerminal(syscall.Stdout)
	NoTrueColor = !TrueColorEnabled()
)

var (
	isTermStdout bool
	isTermStderr bool
	initStdOnce  sync.Once
)

var (
	fdStdout = int(os.Stdout.Fd())
	fdStderr = int(os.Stderr.Fd())
)

func initStdTerms() {
	isTermStdout = term.IsTerminal(fdStdout)
	isTermStderr = term.IsTerminal(fdStderr)
}

//go:generate stringer -type=Attribute

type Attribute uint8

// Base attributes
const (
	None Attribute = iota // TODO: rename to "Reset"
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
	// TODO: remove these if not used
	DoublyUnderlined Attribute = 21
	Framed           Attribute = 51
	Encircled        Attribute = 52
)

// Foreground text colors
const (
	FgBlack Attribute = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// Foreground Hi-Intensity text colors
const (
	FgBrightBlack Attribute = iota + 90
	FgBrightRed
	FgBrightGreen
	FgBrightYellow
	FgBrightBlue
	FgBrightMagenta
	FgBrightCyan
	FgBrightWhite
)

// Background text colors
const (
	BgBlack Attribute = iota + 40
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

// Background Hi-Intensity text colors
const (
	BgHiBlack Attribute = iota + 100
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

var (
	// Foreground text colors
	Black   = NewColor(FgBlack)
	Red     = NewColor(FgRed)
	Green   = NewColor(FgGreen)
	Yellow  = NewColor(FgYellow)
	Blue    = NewColor(FgBlue)
	Magenta = NewColor(FgMagenta)
	Cyan    = NewColor(FgCyan)
	White   = NewColor(FgWhite)

	// Foreground Hi-Intensity text colors
	BrightBlack   = NewColor(FgBrightBlack)
	BrightRed     = NewColor(FgBrightRed)
	BrightGreen   = NewColor(FgBrightGreen)
	BrightYellow  = NewColor(FgBrightYellow)
	BrightBlue    = NewColor(FgBrightBlue)
	BrightMagenta = NewColor(FgBrightMagenta)
	BrightCyan    = NewColor(FgBrightCyan)
	BrightWhite   = NewColor(FgBrightWhite)
)

type Color struct {
	escape string // TODO: rename to "code"
	attrs  []Attribute
}

// NoColor has no color
var NoColor Color

func buildEscape(attrs []Attribute) string {
	if len(attrs) == 0 {
		return ""
	}
	n := len("\x1b[m")
	n += len(attrs) * 2
	for _, a := range attrs {
		if a >= 100 {
			n += 2
		} else if a >= 10 {
			n += 1
		}
	}

	buf := make([]byte, 0, 3)
	var w strings.Builder
	w.Grow(n)
	w.WriteString("\x1b[")

	buf = strconv.AppendUint(buf[:0], uint64(attrs[0]), 10)
	w.Write(buf)
	for i := 1; i < len(attrs); i++ {
		w.WriteByte(';')
		buf = strconv.AppendUint(buf[:0], uint64(attrs[i]), 10)
		w.Write(buf)
	}
	w.WriteByte('m')
	return w.String()
}

// TODO: return a pointer
func NewColor(attributes ...Attribute) *Color {
	if len(attributes) == 0 {
		return &NoColor
	}

	// Create a copy
	attrs := make([]Attribute, len(attributes))
	copy(attrs, attributes)
	return &Color{escape: buildEscape(attrs), attrs: attrs}
}

// 256-color mode — foreground: ESC[38;5;#m   background: ESC[48;5;#m

func (c *Color) String() string {
	if c == nil || len(c.attrs) == 0 {
		return "<nil>"
	}
	var w strings.Builder
	w.WriteString(c.attrs[0].String())
	for i := 1; i < len(c.attrs); i++ {
		w.WriteByte(';')
		w.WriteString(c.attrs[i].String())
	}
	return w.String()
}

func (c *Color) Has(attr Attribute) bool {
	if c != nil {
		for _, a := range c.attrs {
			if a == attr {
				return true
			}
		}
	}
	return false
}

func (c *Color) Set(attr Attribute) *Color {
	if c == nil {
		return NewColor(attr)
	}
	if c.Has(attr) {
		return c
	}
	attrs := make([]Attribute, len(c.attrs)+1)
	copy(attrs, c.attrs)
	attrs[len(attrs)-1] = attr
	return &Color{escape: buildEscape(attrs), attrs: attrs}
}

func (c *Color) IsZero() bool {
	return c == nil || len(c.escape) == 0
}

func (c *Color) Equal(o *Color) bool {
	if c == nil {
		return o == nil
	}
	return o != nil && c.escape == o.escape
}

func (x *Color) Format() string {
	if !x.IsZero() {
		return x.escape
	}
	return ""
}

func (x *Color) Append(b []byte) []byte {
	if !x.IsZero() {
		b = append(b, x.escape...)
	}
	return b
}

func (x *Color) Reset() string {
	if !x.IsZero() {
		return Reset
	}
	return ""
}

func (x *Color) Sprintf(format string, v ...any) string {
	if !x.IsZero() {
		return fmt.Sprintf(x.escape+format+Reset, v...)
	}
	return fmt.Sprintf(format, v...)
}

func (x *Color) Fprintf(w io.Writer, format string, v ...interface{}) (int, error) {
	return fmt.Fprintf(w, x.Format()+format+x.Reset(), v...)
}

// TODO: use this
type Buffer struct {
	*bytes.Buffer
}

func NewBuffer(dst *bytes.Buffer) *Buffer {
	return &Buffer{Buffer: dst}
}

func (b *Buffer) Write(c *Color, p []byte) (int, error) {
	b.Buffer.WriteString(c.Format())
	b.Buffer.Write(p)
	b.Buffer.WriteString(c.Reset())
	return len(p), nil
}

func (b *Buffer) WriteByte(c *Color, ch byte) error {
	b.Buffer.WriteString(c.Format())
	b.Buffer.WriteByte(ch)
	b.Buffer.WriteString(c.Reset())
	return nil
}

func (b *Buffer) WriteRune(c *Color, r rune) (int, error) {
	b.Buffer.WriteString(c.Format())
	n, _ := b.Buffer.WriteRune(r)
	b.Buffer.WriteString(c.Reset())
	return n, nil
}

func (b *Buffer) WriteString(c *Color, s string) (int, error) {
	b.Buffer.WriteString(c.Format())
	b.Buffer.WriteString(s)
	b.Buffer.WriteString(c.Reset())
	return len(s), nil
}

// IsTerminal returns whether the given file descriptor is a terminal.
func IsTerminal(fd int) bool {
	// WARN: this breaks if someone changes Stdout or Stderr
	if fd == fdStdout || fd == fdStderr {
		initStdOnce.Do(initStdTerms)
		if fd == fdStdout {
			return isTermStdout
		}
		return isTermStderr
	}
	// TODO: consider caching the result of this, but note that
	// caching breaks if FDs are reused.
	return term.IsTerminal(fd)
}

func TrueColorEnabled() bool {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		return true
	}
	return false
}

/////////////////////////////////////////////////////////////////

type RGB struct {
	R, G, B uint8
}

func (r RGB) ANSI() Attribute {
	if r.R == r.G && r.R == r.B {
		if r.R < 8 {
			return 16
		}
		if r.R > 248 {
			return 231
		}
		return Attribute(math.Round(((float64(r.R)-8)/247)*24)) + 232
	}
	ansi := 16 + (math.Round(float64(r.R)/255*5) * 36) +
		(math.Round(float64(r.G)/255*5) * 6) +
		math.Round(float64(r.B)/255*5)
	return Attribute(ansi)
}
