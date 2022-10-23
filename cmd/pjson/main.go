package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charlievieth/pjson"
	"github.com/charlievieth/pjson/termcolor"
	"github.com/spf13/cobra"
)

type statReader struct {
	f *os.File
	n int64
}

func (r *statReader) Read(p []byte) (int, error) {
	n, err := r.f.Read(p)
	r.n += int64(n)
	return n, err
}

func streamFile(name string, stream *pjson.Stream, wr *bufio.Writer) (read, written int64, err error) {
	f, err := os.Open(name)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}

	stream.Reset(f)
	written, err = stream.WriteTo(wr)
	if err != nil {
		return 0, written, err
	}
	return fi.Size(), written, nil
}

const statsFormat = `
  # stats
  time:  %s
  read:  %.2f MB - %.2f MB/s
  write: %.2f MB - %.2f MB/s
`

func main() {
	root := cobra.Command{
		Use: "pjson [flags] [file]...",
	}
	flags := root.Flags()
	indentCount := flags.Int("indent", 4, "Use the given number of spaces for indentation.")
	compact := flags.BoolP("compact", "c", false, "Compact JSON output")
	printStats := flags.Bool("stats", false, "Print stats to STDERR.")
	forceColor := flags.BoolP("color", "C", false,
		"By default, pjson outputs colored JSON if writing to a terminal.\n"+
			"You can force it to produce color even if writing to a pipe or a\n"+
			"file using -C, and disable color with -M.")

	root.RunE = func(cmd *cobra.Command, args []string) error {
		var conf pjson.IndentConfig
		if *forceColor || termcolor.IsTerminal(int(os.Stdout.Fd())) {
			conf = pjson.DefaultIndentConfig
		}
		var indent string
		if *indentCount == 8 {
			indent = "\t"
		} else {
			indent = strings.Repeat(" ", *indentCount)
		}

		// WARN WARN WARN
		if *compact {
			return errors.New("compact not supported")
		}

		start := time.Now()
		stream := pjson.NewStream(nil, &conf)
		stream.SetIndent("", indent)

		statsFn := func(nr, nw int64) {
			if *printStats {
				d := time.Since(start)
				mbr := float64(nr) / float64(1024*1024)
				mbw := float64(nw) / float64(1024*1024)
				fmt.Fprintf(os.Stderr, statsFormat,
					d,
					mbr, mbr/d.Seconds(),
					mbw, mbw/d.Seconds(),
				)
			}
		}

		if len(args) == 0 {
			sr := statReader{f: os.Stdin}
			stream.Reset(&sr)
			nw, err := stream.WriteTo(os.Stdout)
			if err != nil {
				return err
			}
			statsFn(sr.n, nw)
			return err
		}

		var read, written int64
		out := bufio.NewWriterSize(os.Stdout, 96*1024)
		for _, name := range args {
			nr, nw, err := streamFile(name, stream, out)
			read += nr
			written += nw
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", name, err)
				continue
			}
		}
		if err := out.Flush(); err != nil {
			return err
		}
		statsFn(read, written)
		return nil
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
