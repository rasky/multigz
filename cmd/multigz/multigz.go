package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rasky/multigz"

	"github.com/djherbis/atime"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh/terminal"
)

const VERSION = "1.0"

var flagStdout = pflag.BoolP("stdout", "c", false, "write on standard output, keep original files unchanged")
var flagDecompress = pflag.BoolP("decompress", "d", false, "decompress")
var flagForce = pflag.BoolP("force", "f", false, "force overwrite of output file")
var flagHelp = pflag.BoolP("help", "h", false, "give this help")
var flagKeep = pflag.BoolP("keep", "k", false, "keep (don't delete) input files")
var flagLicense = pflag.BoolP("license", "L", false, "display software license")
var flagTest = pflag.BoolP("test", "t", false, "test compressed file integrity")
var flagTestMultigz = pflag.BoolP("testmulti", "T", false, "like -t, but also test it is a multigzip")
var flagVersion = pflag.BoolP("version", "V", false, "display version number")
var flagL0 = pflag.Bool("0", false, "")
var flagL1 = pflag.BoolP("fast", "1", false, "compress faster")
var flagL2 = pflag.Bool("2", false, "")
var flagL3 = pflag.Bool("3", false, "")
var flagL4 = pflag.Bool("4", false, "")
var flagL5 = pflag.Bool("5", false, "")
var flagL6 = pflag.Bool("6", false, "")
var flagL7 = pflag.Bool("7", false, "")
var flagL8 = pflag.Bool("8", false, "")
var flagL9 = pflag.BoolP("best", "9", false, "compress better")
var flagRsyncable = pflag.Bool("rsyncable", false, "make rsync-friendly archive")

const (
	ModeCompress = iota
	ModeDecompress
	ModeTest
)

var Mode = ModeCompress
var Level int = 6
var Files []string
var OutFn string
var IsStdinTerm bool = terminal.IsTerminal(0)
var IsStdoutTerm bool = terminal.IsTerminal(1)

func main() {
	pflag.Parse()
	if *flagHelp {
		Usage()
		return
	}
	if *flagLicense {
		License()
	}

	switch {
	case *flagL0:
		Level = 0
	case *flagL1:
		Level = 1
	case *flagL2:
		Level = 2
	case *flagL3:
		Level = 3
	case *flagL4:
		Level = 4
	case *flagL5:
		Level = 5
	case *flagL6:
		Level = 6
	case *flagL7:
		Level = 7
	case *flagL8:
		Level = 8
	case *flagL9:
		Level = 9
	}

	Files = pflag.Args()
	if len(Files) == 0 {
		Files = []string{"-"}
	}

	binname := filepath.Base(os.Args[0])

	if *flagDecompress || strings.Contains(binname, "gunz") {
		Mode = ModeDecompress
	}
	if *flagTest {
		Mode = ModeTest
	}
	if strings.Contains(binname, "zcat") {
		Mode = ModeDecompress
		*flagStdout = true
	}

	SetSignalHandler()
	os.Exit(Compress())
}

func SetSignalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-ch
		os.Remove(OutFn)
	}()
}

func CopyStat(w *os.File, f *os.File) {
	fi, err := f.Stat()
	if err == nil {
		w.Chmod(fi.Mode())
		if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
			w.Chown(int(sys.Uid), int(sys.Gid))
			os.Chtimes(w.Name(), atime.Get(fi), fi.ModTime())
		}
	}
}

func fatal(args ...interface{}) {
	fmt.Fprint(os.Stderr, "multigz: ")
	fmt.Fprintln(os.Stderr, args...)
}

type nopCloser struct{ io.Writer }

func (n nopCloser) Close() error { return nil }

func compressFile(fn string) bool {
	var f *os.File
	var w *os.File

	outStdout := *flagStdout
	if fn == "-" {
		f = os.Stdin
		outStdout = true
	} else {
		var err error
		f, err = os.Open(fn)
		if err != nil {
			fatal(err)
			return false
		}
		defer f.Close()
	}

	if outStdout {
		w = os.Stdout
		if Mode == ModeCompress && IsStdoutTerm && !*flagForce {
			fatal("cannot compress to terminal (use -f to force)")
			return false
		}
	} else {
		var outfn string
		var force bool

		switch Mode {
		case ModeCompress:
			outfn = fn + ".gz"
			force = *flagForce
		case ModeDecompress:
			ext := filepath.Ext(fn)
			if ext != ".gz" && ext != ".Z" {
				fatal(fn, "unknown suffix -- ignored")
				return true
			}
			outfn = fn[:len(fn)-len(ext)]
			force = *flagForce
		case ModeTest:
			outfn = "/dev/null"
			force = true
		}

		if !force {
			if _, err := os.Stat(outfn); err == nil {
				fmt.Printf("multigz: %s already exists; do you wish to overwrite (y or n)? ", outfn)
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if input[0] != 'y' {
					fmt.Println("\tnot overwritten")
					return true
				}
			}
		}

		var err error
		w, err = os.Create(outfn)
		if err != nil {
			fatal(err)
			return false
		}
		if Mode != ModeTest {
			// Setup the global used by the signal handler, so that if we
			// interrupt before the compression/decompression is finished,
			// the temporary file will be deleted
			OutFn = outfn
			defer func() { os.Remove(OutFn) }()
		}
		defer w.Close()
	}

	var zf io.Reader
	var zw io.WriteCloser
	var err error

	switch Mode {
	case ModeCompress:
		if *flagRsyncable {
			zw, err = multigz.NewWriterLevelRsyncable(w, Level)
		} else {
			zw, err = multigz.NewWriterLevel(w, Level, multigz.DefaultBlockSize)
		}
		zf = f
	case ModeDecompress, ModeTest:
		zf, err = gzip.NewReader(f)
		zw = w
	}
	if err != nil {
		fatal(err)
		return false
	}
	defer zw.Close()

	_, err = io.Copy(zw, zf)
	if err != nil {
		fatal(err)
		return false
	}

	zw.Close()
	OutFn = ""
	if Mode != ModeTest {
		CopyStat(w, f)
		if !*flagKeep {
			os.Remove(fn)
		}
	}
	return true
}

func Compress() int {
	for _, fn := range Files {
		if !compressFile(fn) {
			return 1
		}
	}
	return 0
}

func Usage() {
	// We prefer not ot use pflag.Usage for the following reason:
	// 1) It orders by longname option, which is confusing for this option set
	// 2) It shows "[=false]" next to all boolean options
	fmt.Println(`Usage: multigz [OPTION]... [FILE]...
Compress or uncompress FILEs (by default, compress FILES in-place).

Mandatory arguments to long options are mandatory for short options too.

  -c, --stdout      write on standard output, keep original files unchanged
  -d, --decompress  decompress
  -f, --force       force overwrite of output file and compress links
  -h, --help        give this help
  -k, --keep        keep (don't delete) input files
  -L, --license     display software license
  -t, --test        test compressed file integrity
  -v, --verbose     verbose mode
  -V, --version     display version number
  -1, --fast        compress faster
  -9, --best        compress better
      --rsyncable   make rsync-friendly archive

With no FILE, or when FILE is -, read standard input.

Report bugs to <rasky@develer.com>.
`)
}

func License() {
	fmt.Println("multigz", VERSION)
	fmt.Println("Copyright (C) 2015 Giovanni Bajo.")
	fmt.Println(`
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.`)
}
