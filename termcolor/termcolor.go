package termcolor

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const Reset = "\x1b[0m"

var (
	// NoColor     = IsTerminal(syscall.Stdout)
	NoTrueColor = !TrueColorEnabled()
)

func IsTerminal(fd int) bool { return term.IsTerminal(fd) }

func TrueColorEnabled() bool {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		return true
	}
	return false
}

type Color interface {
	TrueColor() bool
	ANSI() ANSI
	Append(b []byte) []byte
	Format() string // TODO: rename
	Sprintf(format string, v ...any) string
	Fprintf(w io.Writer, format string, v ...interface{}) (int, error)
}

var (
	_ Color = ANSI(0)
	_ Color = SGR(0)
	_ Color = RGB{}
	_ Color = (*XColor)(nil)
)

type NoColor struct{}

func (NoColor) TrueColor() bool        { return false }
func (NoColor) ANSI() ANSI             { return 0 }
func (NoColor) Append(b []byte) []byte { return b }
func (NoColor) Format() string         { return "" }

func (NoColor) Sprintf(format string, v ...any) string {
	return fmt.Sprintf(format, v...)
}

func (NoColor) Fprintf(w io.Writer, format string, v ...interface{}) (int, error) {
	return fmt.Fprintf(w, format, v...)
}

/////////////////////////////////////////////////////////////////

type XColor struct {
	escape string
	attrs  []uint8
}

var emptyXColor XColor

// TODO: return a pointer
func NewXColor(attributes ...uint8) *XColor {
	if len(attributes) == 0 {
		return &emptyXColor // WARN
	}
	var w strings.Builder
	n := len("\x1b[m")
	n += len(attributes) * 2
	for _, a := range attributes {
		if a >= 100 {
			n += 2
		} else if a >= 10 {
			n += 1
		}
	}
	w.Grow(n)
	w.WriteString("\x1b[")
	w.WriteString(strconv.FormatUint(uint64(attributes[0]), 10))
	for i := 1; i < len(attributes); i++ {
		w.WriteByte(';')
		w.WriteString(strconv.FormatUint(uint64(attributes[i]), 10))
	}
	w.WriteByte('m')
	return &XColor{escape: w.String(), attrs: attributes}
}

func (x *XColor) ANSI() ANSI {
	panic("NOT implementedu")
}

func (x *XColor) TrueColor() bool {
	panic("NOT implementedu")
}

func (x *XColor) Format() string {
	if x != nil {
		return x.escape
	}
	return ""
}

func (x *XColor) Append(b []byte) []byte {
	if x != nil {
		b = append(b, x.escape...)
	}
	return b
}

func (x *XColor) Reset() string {
	if x != nil && len(x.escape) != 0 {
		return Reset
	}
	return ""
}

func (x *XColor) Sprintf(format string, v ...any) string {
	return fmt.Sprintf(x.Format()+format+x.Reset(), v...)
}

func (x *XColor) Fprintf(w io.Writer, format string, v ...interface{}) (int, error) {
	return fmt.Fprintf(w, x.Format()+format+x.Reset(), v...)
}

/////////////////////////////////////////////////////////////////

type ANSI uint8

// Foreground text colors
const (
	FgBlack ANSI = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

const (
	FgBrightBlack ANSI = iota + 90
	FgBrightRed
	FgBrightGreen
	FgBrightYellow
	FgBrightBlue
	FgBrightMagenta
	FgBrightCyan
	FgBrightWhite
)

var fgColors = [...]string{
	FgBlack:         "\x1b[30m",
	FgRed:           "\x1b[31m",
	FgGreen:         "\x1b[32m",
	FgYellow:        "\x1b[33m",
	FgBlue:          "\x1b[34m",
	FgMagenta:       "\x1b[35m",
	FgCyan:          "\x1b[36m",
	FgWhite:         "\x1b[37m",
	FgBrightBlack:   "\x1b[90m",
	FgBrightRed:     "\x1b[91m",
	FgBrightGreen:   "\x1b[92m",
	FgBrightYellow:  "\x1b[93m",
	FgBrightBlue:    "\x1b[94m",
	FgBrightMagenta: "\x1b[95m",
	FgBrightCyan:    "\x1b[96m",

	// WARN WARN WARN WARN
	FgBrightWhite: "\x1b[0;39m", // WARN
	// FgBrightWhite:   "\x1b[97m",
}

// TODO: remove if not used!
// Standard Colors
const (
	Black   ANSI = 0
	Red     ANSI = 1
	Green   ANSI = 2
	Yellow  ANSI = 3
	Blue    ANSI = 4
	Magenta ANSI = 5
	Cyan    ANSI = 6
	White   ANSI = 7
)

// TODO: remove if not used!
// High-intensity colors
const (
	BrightBlack   ANSI = 8
	BrightRed     ANSI = 9
	BrightGreen   ANSI = 10
	BrightYellow  ANSI = 11
	BrightBlue    ANSI = 12
	BrightMagenta ANSI = 13
	BrightCyan    ANSI = 14
	BrightWhite   ANSI = 15
)

func (ANSI) TrueColor() bool { return false }

func (c ANSI) ANSI() ANSI { return c }

func (c ANSI) Append(b []byte) []byte {
	if uint(c) < uint(len(fgColors)) {
		if s := fgColors[c]; len(s) != 0 {
			return append(b, s...)
		}
	}
	b = append(b, Reset...)
	b = strconv.AppendUint(b, uint64(c), 10)
	return append(b, 'm')
}

func (c ANSI) Format() string {
	if uint(c) < uint(len(fgColors)) {
		if s := fgColors[c]; len(s) != 0 {
			return s
		}
	}
	return "\x1b[0;" + strconv.FormatUint(uint64(c), 10) + "m"
}

// TODO: remove
// func (c ANSI) WriteTo(w io.Writer) (int64, error) {
// 	n, err := fmt.Fprintf(w, "\x1b[38;5;%d", uint8(c))
// 	return int64(n), err
// }

func (c ANSI) format(format string) []byte {
	b := make([]byte, 0, len("\x1b[0;255m\x1b[0m")+len(format))
	b = c.Append(b)
	b = append(b, format...)
	b = append(b, Reset...)
	return b
}

func (c ANSI) Sprintf(format string, v ...interface{}) string {
	return fmt.Sprintf(string(c.format(format)), v...)
}

func (c ANSI) Fprintf(w io.Writer, format string, v ...interface{}) (int, error) {
	return fmt.Fprintf(w, string(c.format(format)), v...)
}

type SGR uint16

func (s SGR) Hex() string {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(s))
	return hex.EncodeToString(b[:])
}

func NewSGR(attr, color uint8) SGR {
	return SGR(attr) | SGR(color)<<8
}

func (s SGR) TrueColor() bool  { return false }
func (s SGR) Attribute() uint8 { return uint8(s) }
func (s SGR) Color() uint8     { return uint8(s >> 8) }
func (s SGR) ANSI() ANSI       { return ANSI(s.Color()) }

func (s SGR) Append(b []byte) []byte {
	b = append(b, "\x1b["...)
	b = strconv.AppendUint(b, uint64(s.Attribute()), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(s.Color()), 10)
	b = append(b, 'm')
	return b
}

func (s SGR) Format() string {
	attr := s.Attribute()
	if attr == 0 {
		return s.ANSI().Format()
	}
	return string(s.Append(make([]byte, 0, len("\x1b[255;255m"))))
}

func (s SGR) format(format string) []byte {
	b := make([]byte, 0, len("\x1b[255;255m\x1b[0m")+len(format))
	b = s.Append(b)
	b = append(b, format...)
	b = append(b, Reset...)
	return b
}

func (s SGR) Sprintf(format string, v ...interface{}) string {
	return fmt.Sprintf(string(s.format(format)), v...)
}

func (s SGR) Fprintf(w io.Writer, format string, v ...interface{}) (int, error) {
	return fmt.Fprintf(w, string(s.format(format)), v...)
}

type RGB struct {
	R, G, B uint8
}

func (RGB) TrueColor() bool { return true }

func (r RGB) Format() string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", uint8(r.R), uint8(r.G), uint8(r.B))
}

func (r RGB) Append(b []byte) []byte {
	b = append(b, "\x1b[38;2;"...)
	b = strconv.AppendUint(b, uint64(r.R), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(r.G), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(r.B), 10)
	return append(b, 'm')
}

func (r RGB) Sprintf(format string, v ...interface{}) string {
	b := make([]byte, 0, len("\x1b[38;2;255;255;255m\x1b[0m")+len(format))
	b = r.Append(b)
	b = append(b, format...)
	b = append(b, Reset...)
	return fmt.Sprintf(string(b), v...)
}

func (r RGB) Fprintf(w io.Writer, format string, v ...interface{}) (int, error) {
	b := make([]byte, 0, len("\x1b[38;2;255;255;255m\x1b[0m")+len(format))
	b = r.Append(b)
	b = append(b, format...)
	b = append(b, Reset...)
	return fmt.Fprintf(w, string(b), v...)
}

func (r RGB) ANSI() ANSI {
	if r.R == r.G && r.R == r.B {
		if r.R < 8 {
			return 16
		}
		if r.R > 248 {
			return 231
		}
		return ANSI(math.Round(((float64(r.R)-8)/247)*24)) + 232
	}
	ansi := 16 + (math.Round(float64(r.R)/255*5) * 36) +
		(math.Round(float64(r.G)/255*5) * 6) +
		math.Round(float64(r.B)/255*5)
	return ANSI(ansi)
}
