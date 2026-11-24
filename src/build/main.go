package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	code, err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) (int, error) {
	const usage = "usage: ptsip-build check|source|revision|assets|wheel|wheel-ready|engine-snapshot|engine-dependency-graph|verify-engine-derived -root <source>"
	if len(args) == 0 {
		return 2, fmt.Errorf("%s", usage)
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "PTSIP source checkout")
	output := flags.String("output", "", "destination archive or wheel directory")
	base := flags.String("base", "", "wheel containing the staged Python package and metadata")
	platformTag := flags.String("platform-tag", "", "Python platform tag of the build host")
	syncSHA := flags.String("sync-sha", "", "exact preceding engine version sync commit")
	if err := flags.Parse(args[1:]); err != nil {
		return 2, err
	}
	if flags.NArg() != 0 || (args[0] != "verify-engine-derived" && *syncSHA != "") {
		return 2, fmt.Errorf("%s", usage)
	}
	switch args[0] {
	case "engine-snapshot", "engine-dependency-graph", "verify-engine-derived", "wheel-ready":
		if *base != "" || *output != "" || *platformTag != "" {
			return 2, fmt.Errorf("%s", usage)
		}
		if err := runEngineDerived(args[0], *root, *syncSHA, stdout); err != nil {
			return 1, err
		}
		return 0, nil
	case "assets":
		if *base != "" || *output != "" || *platformTag != "" {
			return 2, fmt.Errorf("%s", usage)
		}
		resolved, err := filepath.Abs(*root)
		if err != nil {
			return 1, err
		}
		files := wheelContents{}
		if err := stageWheelAssets(resolved, files); err != nil {
			return 1, err
		}
		digests := make(map[string]string, len(files))
		for name, file := range files {
			digests[name] = wheelDigest(file.data)
		}
		if err := json.NewEncoder(stdout).Encode(digests); err != nil {
			return 1, err
		}
		return 0, nil
	case "revision":
		if *base != "" || *output != "" || *platformTag != "" {
			return 2, fmt.Errorf("%s", usage)
		}
		revision, err := buildRevision(*root)
		if err != nil {
			return 1, err
		}
		if _, err := fmt.Fprintln(stdout, revision); err != nil {
			return 1, err
		}
		return 0, nil
	case "wheel":
		if *base == "" || *output == "" || *platformTag == "" {
			return 2, fmt.Errorf("wheel requires -base, -output and -platform-tag")
		}
		name, err := buildWheel(*root, *base, *output, *platformTag, stderr)
		if err != nil {
			return 1, err
		}
		if _, err := fmt.Fprintln(stdout, name); err != nil {
			return 1, err
		}
		return 0, nil
	case "check", "source":
		if (args[0] == "source") != (*output != "") || *base != "" || *platformTag != "" {
			return 2, fmt.Errorf("%s", usage)
		}
	default:
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
		if err := stageRegisteredProductAssets(source, wheelContents{}); err != nil {
			return 1, err
		}
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
