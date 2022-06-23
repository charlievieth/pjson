package termcolor

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
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
				r := atomic.AddInt32(&baseR, 1) - 1
				if r >= 256 {
					return
				}
				for g := 0; g <= 255; g++ {
					for b := 0; b <= 255; b++ {
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

func TestRGB(t *testing.T) {
	t.Skip("WARN: delete me!")
	for r := uint8(0); r < math.MaxUint8; r++ {
		for g := uint8(0); g < math.MaxUint8; g++ {
			for b := uint8(0); b < math.MaxUint8; b++ {
				c := RGB{r, g, b}
				fmt.Println(c.Sprintf("R: %d G: %d B: %d", r, g, b))
			}
		}
	}
}

func TestSGR(t *testing.T) {
	failed := 0
	rr := rand.New(rand.NewSource(123456))
	for i := 0; i < 100; i++ {
		attr := uint8(rr.Intn(100))
		color := uint8(rr.Intn(100))
		sgr := NewSGR(attr, color)
		if sgr.Attribute() != attr {
			t.Errorf("%s - Attribute = %d; want: %d", sgr.Hex(), sgr.Attribute(), attr)
		}
		if sgr.Color() != color {
			t.Errorf("%s - Color = %d; want: %d", sgr.Hex(), sgr.Color(), color)
		}
		if t.Failed() {
			failed++
		}
		if failed >= 10 {
			break
		}
	}
}

func TestANSI(t *testing.T) {
	colors := map[string]ANSI{
		"FgRed":     FgRed,
		"FgGreen":   FgGreen,
		"FgYellow":  FgYellow,
		"FgBlue":    FgBlue,
		"FgMagenta": FgMagenta,
		"FgCyan":    FgCyan,
		"FgWhite":   FgWhite,
	}
	for name, color := range colors {
		fmt.Println(color.Sprintf("%s", name))
	}
}

func TestXXX(t *testing.T) {
	// 1	Bold or increased intensity	As with faint, the color change is a PC (SCO / CGA) invention.[38][better source needed]
	// 2	Faint, decreased intensity, or dim	May be implemented as a light font weight like bold.[39]
	// 3	Italic	Not widely supported. Sometimes treated as inverse or blink.[38]
	// 4	Underline	Style extensions exist for Kitty, VTE, mintty and iTerm2.[40][41]
	// 5	Slow blink	Sets blinking to less than 150 times per minute
	// 6	Rapid blink	MS-DOS ANSI.SYS, 150+ per minute; not widely supported

	const (
		Bold             = 1  // Bold
		Faint            = 2  // Faint
		Italic           = 3  // Italic
		Underline        = 4  // Underline
		SlowBlink        = 5  // Slow
		RapidBlink       = 6  // Rapid
		CrossedOut       = 9  // Crossed
		DoublyUnderlined = 21 // Doubly
		Framed           = 51 // Framed
		Encircled        = 52 // Encircled
	)

	x := NewXColor(Bold, Underline, CrossedOut, uint8(FgRed))
	fmt.Println(x.Format() + "HELLO, WORLD!" + x.Reset())
	x = NewXColor(Bold, Underline, CrossedOut, uint8(FgCyan))
	fmt.Println(x.Format() + "HELLO, WORLD!" + x.Reset())

	// attrs := []uint8{
	// 	1,  // Bold or increased intensity	As with faint, the color change is a PC (SCO / CGA) invention.[38][better source needed]
	// 	2,  // Faint, decreased intensity, or dim	May be implemented as a light font weight like bold.[39]
	// 	3,  // Italic	Not widely supported. Sometimes treated as inverse or blink.[38]
	// 	4,  // Underline	Style extensions exist for Kitty, VTE, mintty and iTerm2.[40][41]
	// 	5,  // Slow blink	Sets blinking to less than 150 times per minute
	// 	6,  // Rapid blink	MS-DOS ANSI.SYS, 150+ per minute; not widely supported
	// 	9,  // Crossed-out, or strike
	// 	21, // Doubly underlined; or: not bold
	// 	51, // Framed
	// 	52, // Encircled
	// }
	// for _, attr := range attrs {
	// 	s := NewSGR(attr, uint8(FgRed))
	// 	fmt.Printf("%d: %s\n", int(attr), s.Format()+"RED"+Reset)
	// }

	// colors := []struct {
	// 	name  string
	// 	color ANSI
	// }{
	// 	{"FgBlack", FgBlack},
	// 	{"FgRed", FgRed},
	// 	{"FgGreen", FgGreen},
	// 	{"FgYellow", FgYellow},
	// 	{"FgBlue", FgBlue},
	// 	{"FgMagenta", FgMagenta},
	// 	{"FgCyan", FgCyan},
	// 	{"FgWhite", FgWhite},
	// 	{"FgBrightBlack", FgBrightBlack},
	// 	{"FgBrightRed", FgBrightRed},
	// 	{"FgBrightGreen", FgBrightGreen},
	// 	{"FgBrightYellow", FgBrightYellow},
	// 	{"FgBrightBlue", FgBrightBlue},
	// 	{"FgBrightMagenta", FgBrightMagenta},
	// 	{"FgBrightCyan", FgBrightCyan},
	// 	{"FgBrightWhite", FgBrightWhite},
	// }
	// for i, c := range colors {
	// 	fmt.Printf("%d: ", i)
	// 	c.color.Fprintf(os.Stdout, "%s\n", c.name)
	// }
}

func BenchmarkANSI_Format(b *testing.B) {
	// "FgMagenta": FgMagenta,
	// c := FgMagenta
	var colors = &[...]ANSI{
		FgBlack,
		FgRed,
		FgGreen,
		FgYellow,
		FgBlue,
		FgMagenta,
		FgCyan,
		FgWhite,
		FgBrightBlack,
		FgBrightRed,
		FgBrightGreen,
		FgBrightYellow,
		FgBrightBlue,
		FgBrightMagenta,
		FgBrightCyan,
		FgBrightWhite,
	}
	n := 0
	for _, c := range colors {
		n += len(c.Format())
	}
	b.SetBytes(int64(n / len(colors)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = colors[i%len(colors)].Format()
		// c.Format()
	}
}

func BenchmarkANSI_Sprintf(b *testing.B) {
	// "FgMagenta": FgMagenta,
	c := FgMagenta
	b.SetBytes(int64(len(c.Sprintf("%s", "FgMagenta"))))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Sprintf("%s", "FgMagenta")
	}
}

func BenchmarkSGR_Append(b *testing.B) {
	c := NewSGR(1, 32)
	buf := make([]byte, 0, 32)
	buf = c.Append(buf)
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Append(buf[:0])
	}
}

func BenchmarkRGB_Append(b *testing.B) {
	// "FgMagenta": FgMagenta,
	c := RGB{R: 50, G: 50, B: 50}
	buf := make([]byte, 0, 32)
	buf = c.Append(buf)
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Append(buf[:0])
	}
}

func BenchmarkRGB_Sprintf(b *testing.B) {
	// "FgMagenta": FgMagenta,
	c := RGB{R: 255, G: 255, B: 255}
	b.SetBytes(int64(len(c.Sprintf("%s", "RGG_White"))))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Sprintf("%s", "RGG_White")
	}
}

func BenchmarkIsTerminal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = IsTerminal(syscall.Stdout)
	}
}

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
