package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	code, err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) (int, error) {
	const usage = "usage: ptsip-build check -root <checkout> | ptsip-build source -root <checkout> -output <archive.tar.gz>"
	if len(args) == 0 || (args[0] != "check" && args[0] != "source") {
		return 2, fmt.Errorf("%s", usage)
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "PTSIP source checkout")
	output := flags.String("output", "", "destination for the product source archive")
	if err := flags.Parse(args[1:]); err != nil {
		return 2, err
	}
	if flags.NArg() != 0 || (args[0] == "source") != (*output != "") {
		return 2, fmt.Errorf("%s", usage)
	}
	source, err := loadGitSource(*root)
	if err != nil {
		return 1, err
	}
	report, err := inspect(source)
	if err != nil {
		return 1, err
	}
	if args[0] == "source" {
		if err := writeSourceArchive(source, *output); err != nil {
			return 1, err
		}
		report.SourceArchive = *output
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return 1, err
	}
	return 0, nil
}
