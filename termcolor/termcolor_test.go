package termcolor

import (
	"bytes"
	"flag"
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"text/tabwriter"
)

// reference implementation
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

func TestRGBToANSI(t *testing.T) {
	numCPU := runtime.NumCPU()
	if numCPU > 16 {
		numCPU = 16
	}
	var baseR int32
	var wg sync.WaitGroup
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				r := atomic.AddInt32(&baseR, 5) - 1
				if r >= 256 {
					return
				}
				for g := 0; g <= 255; g += 5 {
					for b := 0; b <= 255; b += 5 {
						rgb := RGB{uint8(r), uint8(g), uint8(b)}
						c := rgb.ANSI()
						exp := RGBToAnsi256(int(r), g, b)
						if int(c) != exp {
							t.Errorf("R:%d G:%d B:%d got: %d want: %d", r, g, b, c, exp)
						}
					}
				}
			}
		}()
	}
	wg.Wait()
}

type bufferTest struct {
	color    *Color
	in, want string
}

var bufferTests = []bufferTest{
	{Red, "Hello", Red.Format() + "Hello" + Red.Reset()},
	{nil, "Hello", "Hello"},
	{new(Color), "Hello", "Hello"},
}

func TestBuffer(t *testing.T) {
	var dst bytes.Buffer
	buf := Buffer{&dst}

	assert := func(t *testing.T, buf *Buffer, test bufferTest) {
		defer buf.Reset()
		got := buf.String()
		if got != test.want {
			t.Errorf("WriteString(%s, %q) = %s; want: %s", test.color, test.in, got, test.want)
		}
	}

	t.Run("Write", func(t *testing.T) {
		for _, test := range bufferTests {
			buf.Write(test.color, []byte(test.in))
			assert(t, &buf, test)
		}
	})

	t.Run("WriteString", func(t *testing.T) {
		for _, test := range bufferTests {
			buf.WriteString(test.color, test.in)
			assert(t, &buf, test)
		}
	})
}

type colorTest struct {
	color *Color
	want  string
}

var colorTests = []colorTest{
	{Black, "30"},
	{Red, "31"},
	{Green, "32"},
	{Yellow, "33"},
	{Blue, "34"},
	{Magenta, "35"},
	{Cyan, "36"},
	{White, "37"},
	{BrightBlack, "90"},
	{BrightRed, "91"},
	{BrightGreen, "92"},
	{BrightYellow, "93"},
	{BrightBlue, "94"},
	{BrightMagenta, "95"},
	{BrightCyan, "96"},
	{BrightWhite, "97"},
}

func TestNewColor(t *testing.T) {
	t.Run("NoColor", func(t *testing.T) {
		c := NewColor()
		if c != &NoColor {
			t.Error("NewColor() should return a pointer to noColor")
		}
	})

	t.Run("CopyAttr", func(t *testing.T) {
		attrs := []Attribute{Bold, Underline, FgRed}
		want := append([]Attribute(nil), attrs...)

		got := NewColor(attrs...).attrs
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Attributes: got: %v want: %v", got, want)
		}

		// Reverse input to test we made a copy
		for i := len(attrs)/2 - 1; i >= 0; i-- {
			opp := len(attrs) - 1 - i
			attrs[i], attrs[opp] = attrs[opp], attrs[i]
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Attributes: got: %v want: %v", got, want)
		}
	})
}

func testNoColor(t *testing.T, c *Color) {
	want := "hello"
	got := c.Sprintf("hello")
	if got != want {
		t.Errorf("got: %q want: %q", got, want)
	}
}

func TestNoColor(t *testing.T) {
	var c Color
	testNoColor(t, &c)
}

func TestNilColor(t *testing.T) {
	var c *Color = nil
	testNoColor(t, c)
	if c.Format() != "" {
		t.Errorf("Format() = %q; want: %q", c.Format(), "")
	}
	if c.Reset() != "" {
		t.Errorf("Reset() = %q; want: %q", c.Reset(), "")
	}
	if c.Has(0) {
		t.Errorf("Has(0) = %t; want: %t", c.Has(0), false)
	}
	x := c.Set(1) // Make sure this does not panic
	if !x.Has(1) {
		t.Errorf("x.Has(1) = %t; want: %t", x.Has(1), true)
	}
}

func TestDefaultColors(t *testing.T) {
	for _, test := range colorTests {
		want := "\x1b[" + test.want + "mHello" + Reset
		got := test.color.Sprintf("%s", "Hello")
		if got != want {
			t.Errorf("%s: got: %q want: %q", test.color, got, want)
		}
	}
}

func TestColorString(t *testing.T) {
	std := map[string]*Color{
		"FgBlack":              Black,
		"FgRed":                Red,
		"FgGreen":              Green,
		"FgYellow":             Yellow,
		"FgBlue":               Blue,
		"FgMagenta":            Magenta,
		"FgCyan":               Cyan,
		"FgWhite":              White,
		"FgBrightBlack":        BrightBlack,
		"FgBrightRed":          BrightRed,
		"FgBrightGreen":        BrightGreen,
		"FgBrightYellow":       BrightYellow,
		"FgBrightBlue":         BrightBlue,
		"FgBrightMagenta":      BrightMagenta,
		"FgBrightCyan":         BrightCyan,
		"FgBrightWhite":        BrightWhite,
		"Bold;FgRed":           NewColor(Bold, FgRed),
		"Bold;Underline;FgRed": NewColor(Bold, Underline, FgRed),
	}
	for want, color := range std {
		got := color.String()
		if got != want {
			t.Errorf("%s.String() = %s; want: %s", color, got, want)
		}
	}
}

type buildEscapeTest struct {
	attrs []Attribute
	want  string
}

var buildEscapeTests = []buildEscapeTest{
	{nil, ""},
	{[]Attribute{FgRed}, "\x1b[31m"},
	{[]Attribute{Bold, FgRed}, "\x1b[1;31m"},
	{[]Attribute{Bold, Underline, FgRed}, "\x1b[1;4;31m"},
	{[]Attribute{Bold, Underline, FgRed, 150, 255}, "\x1b[1;4;31;150;255m"},
}

func TestBuildEscape(t *testing.T) {
	for _, test := range buildEscapeTests {
		got := buildEscape(test.attrs)
		if got != test.want {
			t.Errorf("buildEscape(%d) = %q; want: %q", test.attrs, got, test.want)
		}
	}
}

func TestBuildEscapeAllocs(t *testing.T) {
	for _, test := range buildEscapeTests {
		allocs := testing.AllocsPerRun(10, func() {
			buildEscape(test.attrs)
		})
		if allocs > 1 {
			t.Errorf("buildEscape(%d) allocs = %.2f want: %d", test.attrs, allocs, 1)
		}
	}
}

var printDefaults = flag.Bool("print-defaults", false, "Print default colors to STDOUT")

func TestPrintDefaultColors(t *testing.T) {
	if !*printDefaults {
		t.Skip("Provide the `-print-defaults` flag to run these tests")
	}
	colors := []struct {
		name  string
		color *Color
	}{
		{"Black", Black},
		{"Red", Red},
		{"Green", Green},
		{"Yellow", Yellow},
		{"Blue", Blue},
		{"Magenta", Magenta},
		{"Cyan", Cyan},
		{"White", White},
		{"BrightBlack", BrightBlack},
		{"BrightRed", BrightRed},
		{"BrightGreen", BrightGreen},
		{"BrightYellow", BrightYellow},
		{"BrightBlue", BrightBlue},
		{"BrightMagenta", BrightMagenta},
		{"BrightCyan", BrightCyan},
		{"BrightWhite", BrightWhite},
	}
	fmt.Fprintln(os.Stdout, "################################")
	w := tabwriter.NewWriter(os.Stdout, 4, 8, 2, ' ', 0)
	for _, c := range colors {
		fmt.Fprintf(w, "%s:\t", c.name)
		c.color.Fprintf(w, "%s\n", c.name)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, "################################")
}

func BenchmarkNewColor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewColor(Bold, FgWhite)
	}
}

func BenchmarkBuffer(b *testing.B) {
	var dst bytes.Buffer
	buf := Buffer{&dst}
	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.WriteString(BrightGreen, "Hello, World!\n")
	}
}

func BenchmarkIsTerminal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = IsTerminal(syscall.Stdout)
	}
}

func TestXXX(t *testing.T) {
	// 256-color mode — foreground: ESC[38;5;#m   background: ESC[48;5;#m
	for i := 0; i < 256; i++ {
		c := NewColor(48, 5, Attribute(i))
		c.Fprintf(os.Stdout, "COLOR: %d", i)
		fmt.Println()
	}

	// const (
	// 	Bold             = 1  // Bold
	// 	Faint            = 2  // Faint
	// 	Italic           = 3  // Italic
	// 	Underline        = 4  // Underline
	// 	SlowBlink        = 5  // Slow
	// 	RapidBlink       = 6  // Rapid
	// 	CrossedOut       = 9  // Crossed
	// 	DoublyUnderlined = 21 // Doubly
	// 	Framed           = 51 // Framed
	// 	Encircled        = 52 // Encircled
	// )

	// x := NewColor(Bold, Underline, CrossedOut, FgRed)
	// if x.IsZero() {
	// 	t.Fatal("IsZero = true")
	// }
	// fmt.Fprintln(os.Stderr, x.Format()+"HELLO, WORLD!"+x.Reset())
	// fmt.Printf("%q\n", x.Format()+"HELLO, WORLD!"+x.Reset())
	// x = NewColor(Bold, Underline, CrossedOut, FgCyan)
	// fmt.Fprintln(os.Stderr, x.Format()+"HELLO, WORLD!"+x.Reset())
}

// func TestRGB(t *testing.T) {
// 	t.Skip("WARN: delete me!")
// 	for r := uint8(0); r < math.MaxUint8; r++ {
// 		for g := uint8(0); g < math.MaxUint8; g++ {
// 			for b := uint8(0); b < math.MaxUint8; b++ {
// 				c := RGB{r, g, b}
// 				fmt.Println(c.Sprintf("R: %d G: %d B: %d", r, g, b))
// 			}
// 		}
// 	}
// }

// func TestSGR(t *testing.T) {
// 	failed := 0
// 	rr := rand.New(rand.NewSource(123456))
// 	for i := 0; i < 100; i++ {
// 		attr := uint8(rr.Intn(100))
//
// 		color := uint8(rr.Intn(100))
// 		sgr := NewSGR(attr, color)
// 		if sgr.Attribute() != attr {
// 			t.Errorf("%s - Attribute = %d; want: %d", sgr.Hex(), sgr.Attribute(), attr)
// 		}
// 		if sgr.Color() != color {
// 			t.Errorf("%s - Color = %d; want: %d", sgr.Hex(), sgr.Color(), color)
// 		}
// 		if t.Failed() {
// 			failed++
// 		}
// 		if failed >= 10 {
// 			break
// 		}
// 	}
// }

// func TestANSI(t *testing.T) {
// 	colors := map[string]ANSI{
// 		"FgRed":     FgRed,
// 		"FgGreen":   FgGreen,
// 		"FgYellow":  FgYellow,
// 		"FgBlue":    FgBlue,
// 		"FgMagenta": FgMagenta,
// 		"FgCyan":    FgCyan,
// 		"FgWhite":   FgWhite,
// 	}
// 	for name, color := range colors {
// 		fmt.Println(color.Sprintf("%s", name))
// 	}
// }

// func BenchmarkANSI_Format(b *testing.B) {
// 	// "FgMagenta": FgMagenta,
// 	// c := FgMagenta
// 	var colors = &[...]ANSI{
// 		FgBlack,
// 		FgRed,
// 		FgGreen,
// 		FgYellow,
// 		FgBlue,
// 		FgMagenta,
// 		FgCyan,
// 		FgWhite,
// 		FgBrightBlack,
// 		FgBrightRed,
// 		FgBrightGreen,
// 		FgBrightYellow,
// 		FgBrightBlue,
// 		FgBrightMagenta,
// 		FgBrightCyan,
// 		FgBrightWhite,
// 	}
// 	n := 0
// 	for _, c := range colors {
// 		n += len(c.Format())
// 	}
// 	b.SetBytes(int64(n / len(colors)))
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		_ = colors[i%len(colors)].Format()
// 		// c.Format()
// 	}
// }

// func BenchmarkANSI_Sprintf(b *testing.B) {
// 	// "FgMagenta": FgMagenta,
// 	c := FgMagenta
// 	b.SetBytes(int64(len(c.Sprintf("%s", "FgMagenta"))))
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		c.Sprintf("%s", "FgMagenta")
// 	}
// }

// func BenchmarkSGR_Append(b *testing.B) {
// 	c := NewSGR(1, 32)
// 	buf := make([]byte, 0, 32)
// 	buf = c.Append(buf)
// 	b.SetBytes(int64(len(buf)))
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		c.Append(buf[:0])
// 	}
// }

// func BenchmarkRGB_Append(b *testing.B) {
// 	// "FgMagenta": FgMagenta,
// 	c := RGB{R: 50, G: 50, B: 50}
// 	buf := make([]byte, 0, 32)
// 	buf = c.Append(buf)
// 	b.SetBytes(int64(len(buf)))
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		c.Append(buf[:0])
// 	}
// }

// func BenchmarkRGB_Sprintf(b *testing.B) {
// 	// "FgMagenta": FgMagenta,
// 	c := RGB{R: 255, G: 255, B: 255}
// 	b.SetBytes(int64(len(c.Sprintf("%s", "RGG_White"))))
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		c.Sprintf("%s", "RGG_White")
// 	}
// }

// func BenchmarkRGBToANSI_Equal(b *testing.B) {
// 	rgb := RGB{128, 128, 128}
// 	for i := 0; i < b.N; i++ {
// 		_ = rgb.ANSI()
// 	}
// }

// func BenchmarkRGBToANSI_RGB(b *testing.B) {
// 	rgb := RGB{1, 128, 255}
// 	for i := 0; i < b.N; i++ {
// 		_ = rgb.ANSI()
// 	}
// }
