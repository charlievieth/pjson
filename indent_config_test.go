package pjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/charlievieth/pjson/termcolor"
	"golang.org/x/term"
)

func TestSpaceConstant(t *testing.T) {
	// Sanity check
	if len(spaces) != 512 {
		t.Errorf("len(spaces) = %d; want: %d", len(spaces), 512)
	}
}

type goldenTest struct {
	name     string
	in, want []byte
}

var goldenTests []goldenTest
var goldenTestsOnce sync.Once

func goldenTestInit() {
	goldenTestsOnce.Do(func() {
		paths, err := filepath.Glob("testdata/golden_tests/*.json")
		if err != nil {
			panic(err)
		}
		sort.Strings(paths)
		for _, path := range paths {
			name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			in, err := os.ReadFile(path)
			if err != nil {
				panic(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(path, ".json") + ".out")
			if err != nil {
				panic(err)
			}
			goldenTests = append(goldenTests, goldenTest{
				name: name,
				in:   in,
				want: bytes.TrimRight(want, "\n"), // WARN: remove trailing newline
			})
		}
	})
}

func (test *goldenTest) Run(t *testing.T, fn func(*bytes.Buffer, []byte) error) {
	t.Run(test.name, func(t *testing.T) {
		var dst bytes.Buffer
		dst.Reset()
		if err := fn(&dst, test.in); err != nil {
			t.Fatal(err)
		}
		got := dst.String()
		compareJSON(t, got, string(test.want))
	})

	// TODO: use this!
	//
	// // Test with a variety of prefix/indent combinations
	// prefixIndents := []struct{ prefix, indent string }{
	// 	{"", ""},
	// 	{"", "  "},
	// 	{"", "    "},
	// 	{"  ", "\t"},
	// }
	// indent := func(t *testing.T, data []byte, prefix, indent string) []byte {
	// 	var buf bytes.Buffer
	// 	var err error
	// 	if prefix == "" && indent == "" {
	// 		err = json.Compact(&buf, test.in)
	// 	} else {
	// 		err = json.Indent(&buf, test.in, prefix, indent)
	// 	}
	// 	if err != nil {
	// 		// WARN WARN WARN
	// 		// panic("FIXME")
	// 		// WARN WARN WARN
	// 		// t.Fatalf("prefix: %q indent: %q error: %v", prefix, indent, err)
	// 		t.Logf("prefix: %q indent: %q error: %v", prefix, indent, err)
	// 		return data
	// 	}
	// 	return bytes.TrimRight(buf.Bytes(), "\n")
	// }
	// t.Run(test.name, func(t *testing.T) {
	// 	var dst bytes.Buffer
	// 	for _, x := range prefixIndents {
	// 		t.Run("", func(t *testing.T) {
	// 			dst.Reset()
	// 			if err := fn(&dst, indent(t, test.in, x.prefix, x.indent)); err != nil {
	// 				t.Fatal(err)
	// 			}
	// 			got := dst.String()
	// 			compareJSON(t, got, string(test.want))
	// 			if t.Failed() {
	// 				t.Logf("Prefix: %q Indent: %q", x.prefix, x.indent)
	// 				return
	// 			}
	// 		})
	// 	}
	// })
}

func testIndentConfigIndentGolden(t *testing.T, streamTest bool, fn func(*IndentConfig, *bytes.Buffer, []byte) error) {
	goldenTestInit()

	streamRe := regexp.MustCompile(`^\d+_stream_`)

	// TODO: use the default config
	conf := &IndentConfig{
		Null:        termcolor.BrightBlack, // WARN
		False:       termcolor.Red,
		True:        termcolor.Green,
		Keyword:     termcolor.Blue,
		Quote:       termcolor.Green,
		String:      termcolor.Green,
		Numeric:     termcolor.Magenta,
		Punctuation: termcolor.Yellow,
	}
	for _, test := range goldenTests {
		if streamRe.MatchString(test.name) && !streamTest {
			continue
		}
		test.Run(t, func(dst *bytes.Buffer, data []byte) error {
			return fn(conf, dst, data)
		})
	}
}

func TestIndentConfigIndent(t *testing.T) {
	fn := func(conf *IndentConfig, dst *bytes.Buffer, data []byte) error {
		return conf.Indent(dst, data, "", "    ")
	}
	testIndentConfigIndentGolden(t, false, fn)
}

func TestIndentConfigStream(t *testing.T) {
	fn := func(conf *IndentConfig, dst *bytes.Buffer, data []byte) error {
		s := NewStream(bytes.NewReader(data), conf)
		s.SetIndent("", "    ")
		_, err := s.Indent(dst)
		return err
		// for {
		// 	_, err := s.Indent(dst)
		// 	if err != nil {
		// 		if err != io.EOF {
		// 			return err
		// 		}
		// 	}
		// }
		// return nil
	}
	testIndentConfigIndentGolden(t, true, fn)
}

var indentTestMap = map[string]interface{}{
	"key😃":         "abcd",
	"key👾":         "☺☻☹",
	"key😈":         "日a本b語ç日ð本Ê語þ日¥本¼語i日©",
	"key👻":         "日a本b語ç日ð本Ê語þ日¥本¼語i日©日a本b語ç日ð本Ê語þ日¥本¼語i日©日a本b語ç日ð本Ê語þ日¥本¼語i日©",
	"ArrayNumeric": []int{1, 2, 3},
	"ArrayString":  []string{"v1", "v2"},
	"BoolFalse":    false,
	"BoolTrue":     true,
	"Null":         nil,
	"Numeric":      123456,
	"Object":       struct{ R, G, B int }{1, 2, 3},
}

func TestXXX(t *testing.T) {
	t.Skip("DELETE ME")

	conf := IndentConfig{
		Null:        termcolor.BrightBlack, // WARN
		False:       termcolor.Red,
		True:        termcolor.Green,
		Keyword:     termcolor.Blue,
		Quote:       termcolor.Green,
		String:      termcolor.Green,
		Numeric:     termcolor.Magenta,
		Punctuation: termcolor.Yellow,
	}

	m := map[string]any{
		"":      "", // really empty
		"Empty": "",
		"Array": []string{"", "value", ""},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var dst bytes.Buffer
	if err := conf.Indent(&dst, data, "", "    "); err != nil {
		t.Fatal(err)
	}
	dst.WriteByte('\n')
	if _, err := dst.WriteTo(os.Stdout); err != nil {
		t.Fatal(err)
	}
	// enc := json.NewEncoder(os.Stdout)
	// enc.SetIndent("", "    ")
	// if err := enc.Encode(indentTestMap); err != nil {
	// 	t.Fatal(err)
	// }
}

var ansiRe = regexp.MustCompile(`(?m)` + "\x1b" + `\[(?:\d+(?:;\d+)*)?m`)

func compareJSON(t testing.TB, got, want string) bool {
	t.Helper()

	const diffMax = 8192
	switch {
	case got == want:
		return true
	// case strings.TrimRight(got, "\n") == want:
	// 	t.Error("Newline at end of file")
	// case got == strings.TrimRight(want, "\n"):
	// 	t.Error("No newline at end of file")
	// case ansiRe.ReplaceAllString(got, "") == ansiRe.ReplaceAllString(want, ""):
	// 	t.Error("ANSI color mismatch")
	// 	fallthrough
	default:
		diff(t, []byte(got), []byte(want))
		os.Stdout.WriteString(termcolor.Reset)
		if testing.Verbose() || len(got) <= diffMax && len(want) <= diffMax {
			// diffStrings returns at most 80 lines
			t.Errorf("\n\n### Diff:\n%s\n", diffStrings(t, got, want))

			// Don't print large JSON objects
			if len(got) <= diffMax && len(want) <= diffMax {
				t.Errorf("\nGot:\n###\n%s\n###\nWant:\n###\n%s\n###\n", got, want)
				os.Stdout.WriteString(termcolor.Reset)
			}
		}
	}
	return false
}

// TODO: rename ???
func TestIndentConfigNoColor(t *testing.T) {
	data, err := json.MarshalIndent(indentTestMap, "", " \t \t \n")
	if err != nil {
		t.Fatal(err)
	}

	test := func(t *testing.T, gotFn, wantFn func(*bytes.Buffer, []byte) error) {
		t.Helper()
		var b1 bytes.Buffer
		if err := gotFn(&b1, data); err != nil {
			t.Fatal(err)
		}
		var b2 bytes.Buffer
		if err := wantFn(&b2, data); err != nil {
			t.Fatal(err)
		}
		got := b1.String()
		want := b2.String()
		compareJSON(t, got, want)
	}

	t.Run("Indent", func(t *testing.T) {
		var prefix string
		gotFn := func(dst *bytes.Buffer, src []byte) error {
			var noColor IndentConfig
			return noColor.Indent(dst, src, prefix, "    ")
		}
		wantFn := func(dst *bytes.Buffer, src []byte) error {
			return json.Indent(dst, src, prefix, "    ")
		}
		t.Run("NoPrefix", func(t *testing.T) {
			prefix = ""
			test(t, gotFn, wantFn)
		})
		t.Run("TabPrefix", func(t *testing.T) {
			prefix = "\t"
			test(t, gotFn, wantFn)
		})
	})

	t.Run("IndentStream", func(t *testing.T) {
		var prefix string
		gotFn := func(dst *bytes.Buffer, src []byte) error {
			var noColor IndentConfig
			return noColor.IndentStream(dst, bytes.NewReader(src), prefix, "    ")
		}
		wantFn := func(dst *bytes.Buffer, src []byte) error {
			return json.Indent(dst, src, prefix, "    ")
		}
		t.Run("NoPrefix", func(t *testing.T) {
			prefix = ""
			test(t, gotFn, wantFn)
		})
		t.Run("TabPrefix", func(t *testing.T) {
			prefix = "\t"
			test(t, gotFn, wantFn)
		})
	})

	t.Run("Compact", func(t *testing.T) {
		noColor := &IndentConfig{}
		test(t, noColor.Compact, json.Compact)
	})

	t.Run("CompactStream", func(t *testing.T) {
		gotFn := func(dst *bytes.Buffer, src []byte) error {
			var noColor IndentConfig
			return noColor.CompactStream(dst, bytes.NewReader(src))
		}
		test(t, gotFn, json.Compact)
	})
}

func testIndentConfigIndentStream(t *testing.T, conf *IndentConfig, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	var dst bytes.Buffer
	if err := conf.Indent(&dst, data, "", "    "); err != nil {
		t.Fatal(err)
	}
	want := dst.String()

	dst.Reset()
	if err := conf.IndentStream(&dst, bytes.NewReader(data), "", "    "); err != nil {
		t.Fatal(err)
	}
	got := dst.String()
	compareJSON(t, got, want)

	// if got != want {
	// 	t.Errorf("got:\n%s\nwant:\n%s\n", got, want)
	// 	diff(t, []byte(got), []byte(want))
	// 	t.Log(termcolor.Reset)
	// 	t.Errorf("\n\n### Diff:\n%s\n", diffStrings(t, got, want))
	// }
	// if t.Failed() {
	// 	fmt.Printf("###\n%s\n###\n", got)
	// }
}

type errWriter struct {
	Error error
}

func (w *errWriter) Write(p []byte) (int, error) {
	return 0, w.Error
}

// Test that Indent and IndentStream return identical responses
func TestIndentConfigIndentStream(t *testing.T) {
	t.Run("TestMap", func(t *testing.T) {
		testIndentConfigIndentStream(t, &DefaultIndentConfig, indentTestMap)
	})

	t.Run("IntList", func(t *testing.T) {
		intList := make([]int, 4096)
		for i := 0; i < len(intList); i++ {
			intList[i] = i
		}
		testIndentConfigIndentStream(t, &DefaultIndentConfig, map[string][]int{
			"A": intList,
			"B": intList,
			"C": intList,
			"D": intList,
		})
	})

	t.Run("Large", func(t *testing.T) {
		if testing.Short() {
			t.Skip("short test")
		}
		if codeJSON == nil {
			codeInit()
		}
		testIndentConfigIndentStream(t, &DefaultIndentConfig, codeStruct)
	})

	// TODO: IndentStream should probably have a trailing newline
	t.Run("NoTrailingNewline", func(t *testing.T) {
		conf := DefaultIndentConfig
		var dst bytes.Buffer
		if err := conf.IndentStream(&dst, strings.NewReader("[1,2,3]"), "", "    "); err != nil {
			t.Fatal(err)
		}
		if bytes.HasSuffix(dst.Bytes(), []byte{'\n'}) {
			t.Errorf("Indented JSON ends with a newline: %q", &dst)
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		conf := DefaultIndentConfig
		var dst bytes.Buffer
		err := conf.IndentStream(&dst, strings.NewReader("[{"), "", "    ")
		if err == nil {
			t.Errorf("Expected an error got: %v", err)
			t.Logf("### Buffer:\n%s\n###", &dst)
		}
	})

	t.Run("ReadError", func(t *testing.T) {
		conf := DefaultIndentConfig
		want := errors.New("read error")
		got := conf.IndentStream(io.Discard, iotest.ErrReader(want), "", "    ")
		if !errors.Is(got, want) {
			t.Errorf("ReadError: got: %v want: %v", got, want)
		}
	})

	t.Run("WriteError", func(t *testing.T) {
		conf := DefaultIndentConfig
		want := errors.New("write error")
		got := conf.IndentStream(&errWriter{want}, strings.NewReader("[1,2,3]"), "", "    ")
		if !errors.Is(got, want) {
			t.Errorf("ReadError: got: %v want: %v", got, want)
		}
	})

	encodeFn := func(t *testing.T, indent string, vals ...interface{}) string {
		var dst bytes.Buffer
		enc := json.NewEncoder(&dst)
		if indent != "" {
			enc.SetIndent("", indent)
		}
		for _, v := range vals {
			if err := enc.Encode(v); err != nil {
				t.Fatal(err)
			}
		}
		return dst.String()
	}

	t.Run("MultipleStreams", func(t *testing.T) {
		want := encodeFn(t, "    ", []int{1, 2, 3}, map[string]interface{}{
			"k1": "v1", "k2": false,
		})
		compact := encodeFn(t, "", []int{1, 2, 3}, map[string]interface{}{
			"k1": "v1", "k2": false,
		})

		var noColor IndentConfig
		t.Run("Newline", func(t *testing.T) {
			var dst bytes.Buffer
			err := noColor.IndentStream(&dst, strings.NewReader(compact), "", "    ")
			if err != nil {
				t.Fatalf("Error: %v\n### Buffer:\n%s\n###", err, &dst)
			}
			compareJSON(t, dst.String(), want)
		})

		// Test multiple JSON values on a single line (jq supports this)
		t.Run("SingleLine", func(t *testing.T) {
			var dst bytes.Buffer
			singleLine := strings.ReplaceAll(compact, "\n", " ")
			err := noColor.IndentStream(&dst, strings.NewReader(singleLine), "", "    ")
			if err != nil {
				t.Fatalf("Error: %v\n### Buffer:\n%s\n###", err, &dst)
			}
			compareJSON(t, dst.String(), want)
		})
	})

	t.Run("MultipleStreamsError", func(t *testing.T) {
		compact := encodeFn(t, "", []int{1, 2, 3}, map[string]interface{}{
			"k1": "v1", "k2": false,
		})
		compact += "\n{" // Invalid

		var noColor IndentConfig
		var dst bytes.Buffer
		err := noColor.IndentStream(&dst, strings.NewReader(compact), "", "    ")
		if err == nil {
			t.Errorf("Expected an error got: %v", err)
			t.Logf("### Buffer:\n%s\n###", &dst)
		}
	})
}

func TestIndentConfigIndentEmptyConfig(t *testing.T) {
	t.Skip("FIXME")
	data, err := json.Marshal(indentTestMap)
	if err != nil {
		t.Fatal(err)
	}
	var conf IndentConfig
	var dst bytes.Buffer
	if err := conf.Indent(&dst, data, "", "    "); err != nil {
		t.Fatal(err)
	}
	dst.WriteByte('\n')
	dst.WriteTo(os.Stdout)
}

func TestIndentConfigIndentUnicode(t *testing.T) {
	t.Skip("FIXME")
	data, err := json.Marshal(indentTestMap)
	if err != nil {
		t.Fatal(err)
	}
	conf := JQIndentConfig
	var dst bytes.Buffer
	if err := conf.Indent(&dst, data, "", "    "); err != nil {
		t.Fatal(err)
	}
	dst.WriteByte('\n')
	dst.WriteTo(os.Stdout)
}

func TestIndentConfigCompact(t *testing.T) {
	t.Skip("FIXME")
	data, err := json.Marshal(indentTestMap)
	if err != nil {
		t.Fatal(err)
	}
	conf := DefaultIndentConfig
	var dst bytes.Buffer
	if err := conf.Compact(&dst, data); err != nil {
		t.Fatal(err)
	}
	dst.WriteByte('\n')
	dst.WriteTo(os.Stdout)
}

func diffStrings(t testing.TB, got, want string) string {
	if _, err := exec.LookPath("git"); err != nil {
		t.Log("Skipping:", err)
		return err.Error()
	}

	tempdir, err := os.MkdirTemp("", "termcolor-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !t.Failed() {
			os.RemoveAll(tempdir)
		} else {
			t.Logf("TEMPDIR: %s", tempdir)
		}
	})

	if err := os.WriteFile(filepath.Join(tempdir, "got.txt"),
		[]byte(strings.ReplaceAll(got, "\x1b", `\x1b`)), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempdir, "want.txt"),
		[]byte(strings.ReplaceAll(want, "\x1b", `\x1b`)), 0644); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"diff",
		"--no-index",
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		args = append(args, "--color=always")
	}
	args = append(args, "got.txt", "want.txt")

	cmd := exec.Command("git", args...)
	cmd.Dir = tempdir
	out, _ := cmd.CombinedOutput()

	out = bytes.TrimSpace(out)
	sep := []byte{'\n'}
	if bytes.Count(out, sep) > 80 {
		a := bytes.Split(out, sep)
		if n := len(a); n > 80 {
			a = a[:80]
			a = append(a, []byte(fmt.Sprintf("### Omitting %d lines ... ###\n", n-80)))
		}
		out = bytes.Join(a, sep)
	}

	return string(bytes.TrimSpace(out))
}

func BenchmarkIndentConfigIndent_Indent(b *testing.B) {
	b.ReportAllocs()
	if codeJSON == nil {
		b.StopTimer()
		codeInit()
		b.StartTimer()
	}

	b.ResetTimer()

	b.Run("Color", func(b *testing.B) {
		b.SetBytes(int64(len(codeJSON)))
		var dst bytes.Buffer
		conf := DefaultIndentConfig
		for i := 0; i < b.N; i++ {
			dst.Reset()
			conf.Indent(&dst, codeJSON, "", "    ")
		}
	})
	b.Run("Baseline", func(b *testing.B) {
		b.SetBytes(int64(len(codeJSON)))
		var dst bytes.Buffer
		for i := 0; i < b.N; i++ {
			dst.Reset()
			Indent(&dst, codeJSON, "", "    ")
		}
	})
}

func BenchmarkIndentConfigIndent_IndentStream(b *testing.B) {
	b.ReportAllocs()
	if codeJSON == nil {
		b.StopTimer()
		codeInit()
		b.StartTimer()
	}

	r := bytes.NewReader(codeJSON)
	b.ResetTimer()

	b.SetBytes(int64(r.Len()))
	conf := DefaultIndentConfig

	for i := 0; i < b.N; i++ {
		r.Reset(codeJSON)
		conf.IndentStream(io.Discard, r, "", "    ")
	}
}

func BenchmarkIndentConfigIndent_IndentStream_File(b *testing.B) {
	b.ReportAllocs()
	if codeJSON == nil {
		b.StopTimer()
		codeInit()
		b.StartTimer()
	}
	name := filepath.Join(b.TempDir(), "code.json")
	if err := os.WriteFile(name, codeJSON, 0644); err != nil {
		b.Fatal(err)
	}

	f, err := os.Open(name)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(fi.Size())
	b.ResetTimer()

	conf := DefaultIndentConfig
	for i := 0; i < b.N; i++ {
		if _, err := f.Seek(0, 0); err != nil {
			b.Fatal(err)
		}
		conf.IndentStream(io.Discard, f, "", "    ")
	}
}

func BenchmarkIndentConfigIndent_Compact(b *testing.B) {
	b.ReportAllocs()
	if codeJSON == nil {
		b.StopTimer()
		codeInit()
		b.StartTimer()
	}

	b.Run("Color", func(b *testing.B) {
		b.SetBytes(int64(len(codeJSON)))
		conf := DefaultIndentConfig
		var dst bytes.Buffer
		for i := 0; i < b.N; i++ {
			dst.Reset()
			conf.Compact(&dst, codeJSON)
		}
	})

	b.Run("Baseline", func(b *testing.B) {
		b.SetBytes(int64(len(codeJSON)))
		var dst bytes.Buffer
		for i := 0; i < b.N; i++ {
			dst.Reset()
			json.Compact(&dst, codeJSON)
		}
	})
}
