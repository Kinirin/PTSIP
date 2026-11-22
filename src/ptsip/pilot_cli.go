package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/pilot"
)

var errPilotSnapshotChanged = errors.New("repository changed during the pilot")

func runPilot(args []string) error {
	flags := flag.NewFlagSet("ptsip pilot", flag.ContinueOnError)
	reportPath := flags.String("report", "", "explicit report destination")
	packageRoot := flags.String("ptsip-root", os.Getenv("PTSIP_PACKAGE_ROOT"), "installed PTSIP package root")
	scripts := flags.String("scripts-dir", "", "installed native script directory")
	asJSON := flags.Bool("json", false, "print a JSON result")
	target := "."
	hasTarget := len(args) > 0 && !strings.HasPrefix(args[0], "-")
	if hasTarget {
		target, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() > 1 || hasTarget && flags.NArg() == 1 {
		return fmt.Errorf("pilot accepts one repository path")
	}
	if flags.NArg() == 1 {
		target = flags.Arg(0)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	if *scripts == "" {
		*scripts = filepath.Dir(executable)
	}
	if *packageRoot == "" {
		*packageRoot, err = installedPilotPackageRoot(filepath.Dir(executable))
		if err != nil {
			return err
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, err := pilot.OpenInstalled(executable, *packageRoot, *scripts).Run(ctx, target)
	if err != nil {
		return err
	}
	written, err := pilot.WriteReport(&result, *reportPath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	if !*asJSON {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(struct {
		pilot.Report
		Path string `json:"report_path"`
	}{result, written}); err != nil {
		return err
	}
	if !result.Snapshot.Comparison.Stable {
		return errPilotSnapshotChanged
	}
	return nil
}

func installedPilotPackageRoot(scripts string) (string, error) {
	base := filepath.Dir(scripts)
	locations := []string{filepath.Join(base, "Lib", "site-packages", "ptsip")}
	for _, pattern := range []string{"site-packages", "dist-packages"} {
		candidates, err := filepath.Glob(filepath.Join(base, "lib", "python*", pattern, "ptsip"))
		if err != nil {
			return "", err
		}
		locations = append(locations, candidates...)
	}
	var roots []string
	seen := map[string]bool{}
	for _, candidate := range locations {
		root, err := filepath.EvalSymlinks(candidate)
		if err != nil || seen[root] {
			continue
		}
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(engineIndex))); err == nil && info.Mode().IsRegular() {
			roots = append(roots, root)
			seen[root] = true
		}
	}
	if len(roots) != 1 {
		return "", fmt.Errorf("pilot could not select one installed PTSIP engine index, supply --ptsip-root")
	}
	return roots[0], nil
}
