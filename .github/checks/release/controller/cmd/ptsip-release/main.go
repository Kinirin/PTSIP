package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "registered release command required")
		os.Exit(2)
	}
	command := os.Args[1]
	f := flag.NewFlagSet(command, flag.ExitOnError)
	root := f.String("root", ".", "repository root")
	targets := f.String("targets", "package", "registered component selection")
	optional := f.String("optional-platforms", "", "explicit optional hosts")
	output := f.String("output", "", "derived selection output")
	selector := f.String("version", "", "Package base or exact identity")
	registryPath := f.String("registry", ".github/checks/release/package-registry.json", "Package registry")
	python := f.String("python", "", "explicit target Python executable")
	selectionRef := f.String("selection", ".artifacts/release/selection.json", "verified selection")
	runID := f.Int64("run-id", 0, "original tooling-test run")
	resumeID := f.Int64("release-id", 0, "existing verified-source GitHub release ID")
	destination := f.String("destination", "github-pypi", "registered publication destination")
	inputRef := f.String("input", ".artifacts/release/verified", "original verified artifacts")
	f.Parse(os.Args[2:])
	var value any
	var err error
	switch command {
	case "prepare-publication":
		value, err = release.PreparePublication(*root, *targets, *optional, *output)
	case "admit", "prepare-shipping", "publish-carrier", "pypi-stage", "pypi-verify", "record-package":
		var s release.Selection
		if command != "admit" && command != "prepare-shipping" {
			var manifest release.ShippingManifest
			err = release.ReadJSON(*root, *output+"/shipping-manifest.json", &manifest)
			s = manifest.Selection
		} else {
			err = release.ReadJSON(*root, *selectionRef, &s)
		}
		if err != nil {
			break
		}
		if command == "admit" {
			requested, e := release.Select(*root, *targets, *optional)
			if e != nil || requested.SelectionSHA != s.SelectionSHA {
				err = fmt.Errorf("requested publication selection differs from verified selection")
				break
			}
			err = release.AdmitVerification(*root, s, *runID)
			value = map[string]string{"status": "ADMITTED", "source_sha": s.SourceSHA, "selection_sha256": s.SelectionSHA}
			if err == nil && os.Getenv("GITHUB_OUTPUT") != "" {
				matrix := struct {
					Include []map[string]string `json:"include"`
				}{Include: []map[string]string{}}
				for _, c := range s.Components {
					if c.Kind != "profiles" {
						matrix.Include = append(matrix.Include, map[string]string{"component": c.ID})
					}
				}
				b, _ := json.Marshal(matrix)
				file, e := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_WRONLY|os.O_APPEND, 0600)
				if e != nil {
					err = e
					break
				}
				fmt.Fprintf(file, "publication_matrix=%s\npypi_count=%d\n", b, len(matrix.Include))
				file.Close()
			}
		} else if command == "prepare-shipping" {
			value, err = release.PrepareShipping(*root, s, *runID, *inputRef, *output)
		} else if command == "pypi-stage" || command == "pypi-verify" {
			if command == "pypi-stage" {
				err = release.AdmitPublisherScope(*root, s, *destination)
				if err != nil {
					break
				}
			}
			err = release.PyPIStage(*root, s, *output, command == "pypi-verify")
			if err == nil && command == "pypi-stage" && os.Getenv("GITHUB_OUTPUT") != "" {
				matrix := struct {
					Include []map[string]string `json:"include"`
				}{Include: []map[string]string{}}
				for _, c := range s.Components {
					entries, e := os.ReadDir(filepath.Join(*root, *output, "pypi", c.ID))
					if e != nil && !os.IsNotExist(e) {
						err = e
						break
					}
					if len(entries) > 0 && *destination == "github-pypi" {
						environment, e := release.PublisherEnvironment(*root, c)
						if e != nil {
							err = e
							break
						}
						matrix.Include = append(matrix.Include, map[string]string{"component": c.ID, "environment": environment})
					}
				}
				if err == nil {
					b, _ := json.Marshal(matrix)
					file, e := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_WRONLY|os.O_APPEND, 0600)
					if e != nil {
						err = e
					} else {
						fmt.Fprintf(file, "publication_matrix=%s\npypi_count=%d\n", b, len(matrix.Include))
						file.Close()
					}
				}
			}
			value = map[string]string{"status": "PASS"}
		} else if command == "record-package" {
			err = release.RecordPackage(*root, s, *output)
			value = map[string]string{"status": "RECORDED"}
		} else {
			var manifest release.ShippingManifest
			manifest, err = release.Shipping(*root, s, *output)
			if err == nil {
				err = release.AdmitShipping(*root, manifest)
			}
			if err == nil {
				err = release.PublishCarrier(*root, s, *output, *resumeID)
			}
			value = map[string]string{"status": "PUBLISHED", "tag": s.Tag}
		}
	case "select":
		value, err = release.Select(*root, *targets, *optional)
	case "resolve-package", "install-package", "identities":
		var registry release.Registry
		registry, err = release.PublishedRegistry(*root, *registryPath, "")
		if err != nil {
			break
		}
		var snapshot release.Snapshot
		snapshot, err = release.ResolvePackage(registry, *selector)
		if err != nil {
			break
		}
		value = snapshot
		if command == "install-package" {
			platform := runtime.GOOS + "-x64"
			if runtime.GOARCH != "amd64" && !(runtime.GOOS == "darwin" && runtime.GOARCH == "arm64") {
				err = fmt.Errorf("unregistered native Package installation host")
				break
			}
			if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
				platform = "macos-arm64"
			}
			err = release.InstallPackage(snapshot, platform, *python)
		}
		if command == "identities" {
			composition, e := release.ReadComposition(*root)
			if e != nil {
				err = e
				break
			}
			fmt.Print(release.IdentityText(snapshot, composition.SpecRevision, composition.SpecSource))
			return
		}
	default:
		err = fmt.Errorf("unregistered release command: %s", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *output != "" && (command == "select" || command == "resolve-package") {
		p := *output
		if !filepath.IsAbs(p) {
			p = filepath.Join(*root, p)
		}
		if err := release.WriteJSON(p, value); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	json.NewEncoder(os.Stdout).Encode(value)
}
