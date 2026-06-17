// Command arkitecture is the CLI for the arkitecture library: it reads a .ark
// file, runs the pipeline, and writes an SVG. All compilation logic lives in
// the library (github.com/kurrik/arkitecture); this binary only handles flags
// and file I/O.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kurrik/arkitecture"
)

const version = "0.2.0-dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("arkitecture", flag.ContinueOnError)
	var (
		verbose      bool
		watch        bool
		validateOnly bool
		fontSize     int
		fontFamily   string
		showVersion  bool
	)
	fs.BoolVar(&verbose, "verbose", false, "Show detailed processing information")
	fs.BoolVar(&verbose, "v", false, "Show detailed processing information (shorthand)")
	fs.BoolVar(&watch, "watch", false, "Watch the input file and regenerate on change")
	fs.BoolVar(&watch, "w", false, "Watch the input file and regenerate on change (shorthand)")
	fs.BoolVar(&validateOnly, "validate-only", false, "Parse and validate without generating SVG")
	fs.IntVar(&fontSize, "font-size", 0, "Override the default font size (12)")
	fs.StringVar(&fontFamily, "font-family", "", "Override the default font family (Arial)")
	fs.BoolVar(&showVersion, "version", false, "Print version and exit")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "Usage: arkitecture [options] <input.ark> [output.svg]")
		fmt.Fprintln(out, "\nOptions:")
		fs.PrintDefaults()
		fmt.Fprintln(out, "\nExamples:")
		fmt.Fprintln(out, "  arkitecture diagram.ark diagram.svg")
		fmt.Fprintln(out, "  arkitecture diagram.ark --validate-only")
	}

	// Allow flags and positional arguments to be interspersed. Go's flag
	// package stops at the first positional, so parse in a loop: consume
	// leading flags, take one positional, repeat.
	var positional []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		rest = fs.Args()[1:]
	}

	if showVersion {
		fmt.Println(version)
		return 0
	}

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "error: missing input file")
		fs.Usage()
		return 2
	}
	input := positional[0]
	output := ""
	if len(positional) >= 2 {
		output = positional[1]
	} else {
		output = strings.TrimSuffix(input, filepath.Ext(input)) + ".svg"
	}

	if watch {
		// fsnotify-based watch mode is part of the generator/CLI port still in
		// progress; surface that clearly rather than silently ignoring -w.
		fmt.Fprintln(os.Stderr, "error: watch mode is not yet ported to Go")
		return 2
	}

	return process(input, output, validateOnly, verbose, fontSize, fontFamily)
}

func process(input, output string, validateOnly, verbose bool, fontSize int, fontFamily string) int {
	if verbose {
		fmt.Printf("Processing: %s -> %s\n", input, output)
	}

	data, err := os.ReadFile(input)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			fmt.Fprintf(os.Stderr, "File not found: %s\n", input)
		case os.IsPermission(err):
			fmt.Fprintf(os.Stderr, "Permission denied: %s\n", input)
		default:
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", input, err)
		}
		return 2
	}

	res := arkitecture.ToSVG(string(data), &arkitecture.Options{
		ValidateOnly: validateOnly,
		FontSize:     fontSize,
		FontFamily:   fontFamily,
	})

	if !res.Success {
		fmt.Fprintln(os.Stderr, "Errors:")
		printErrors(res.Errors)
		return 1
	}

	if validateOnly {
		fmt.Println("✓ DSL is valid")
		return 0
	}

	if err := os.WriteFile(output, []byte(res.SVG), 0o644); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "Permission denied writing to: %s\n", output)
			return 2
		}
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", output, err)
		return 2
	}

	if verbose {
		fmt.Printf("Wrote %d bytes to %s\n", len(res.SVG), output)
	}
	fmt.Printf("✓ Generated SVG: %s\n", output)
	return 0
}

func printErrors(errs []arkitecture.Error) {
	for _, e := range errs {
		loc := ""
		if e.Line > 0 {
			loc = fmt.Sprintf(" (line %d, column %d)", e.Line, e.Column)
		}
		fmt.Fprintf(os.Stderr, "  %s%s: %s\n", strings.ToUpper(string(e.Type)), loc, e.Message)
	}
}
