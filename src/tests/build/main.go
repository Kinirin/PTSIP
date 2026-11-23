package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

type report struct {
	Stage                 string   `json:"stage"`
	SourceSHA             string   `json:"source_sha,omitempty"`
	Contract              string   `json:"contract,omitempty"`
	ContractSHA256        string   `json:"contract_sha256,omitempty"`
	ArtifactSHA256        string   `json:"artifact_sha256,omitempty"`
	AssetCount            int      `json:"registered_asset_count"`
	Status                string   `json:"status"`
	Problems              []string `json:"problems,omitempty"`
	PublicationAuthorized bool     `json:"publication_authorized"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("verify-build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "PTSIP checkout root")
	stage := flags.String("stage", "prebuild", "prebuild, sdist, wheel or installed")
	artifact := flags.String("artifact", "", "sdist or wheel file")
	python := flags.String("python", "", "absolute Python path for isolated installed verification")
	wheelhouse := flags.String("wheelhouse", "", "optional local dependency wheels for offline installed verification")
	receipt := flags.String("receipt", "", "write an exact-source native Wheel receipt after installed verification")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return 0
	} else if err != nil {
		return 2
	}
	validStage := *stage == "prebuild" || *stage == "sdist" || *stage == "wheel" || *stage == "installed"
	if !validStage || flags.NArg() != 0 || (*stage == "prebuild") == (*artifact != "") ||
		(*stage == "installed") != (*python != "") || (*stage != "installed" && (*wheelhouse != "" || *receipt != "")) {
		fmt.Fprintln(stderr, "Select a verification stage and supply its artifact and Python inputs as required.")
		flags.PrintDefaults()
		return 2
	}

	result := report{Stage: *stage, Status: "FAIL"}
	if err := verify(*root, *artifact, *python, *wheelhouse, *receipt, &result); err != nil {
		result.Problems = append(result.Problems, err.Error())
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if result.Status == "PASS" {
		return 0
	}
	return 1
}

func verify(root, artifact, python, wheelhouse, receipt string, result *report) error {
	source, err := openSource(root)
	if err != nil {
		return err
	}
	result.SourceSHA = source.revision
	contract, relative, hash, err := loadContract(source)
	if err != nil {
		return err
	}
	result.Contract, result.ContractSHA256 = relative, hash
	assets, err := loadAssets(source, contract.Source)
	if err != nil {
		return err
	}
	result.AssetCount = len(assets.digests)
	if result.Stage == "prebuild" {
		result.Problems = prebuildProblems(source, contract)
		if len(result.Problems) != 0 {
			result.Status = "BLOCKED"
			return nil
		}
	} else {
		var wheel verifiedWheel
		file, err := os.Open(artifact)
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%s must be a regular artifact file", artifact)
		}
		result.ArtifactSHA256, err = digestReader(file)
		if err != nil {
			return err
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if result.Stage == "sdist" {
			if err := verifySDist(file, source.revision, contract.SDist, assets); err != nil {
				return err
			}
		} else {
			if err := verifyWheel(file, info.Size(), source.revision, contract, assets, &wheel); err != nil {
				return err
			}
			if result.Stage == "installed" {
				if err := verifyInstalled(file, artifact, python, wheelhouse, contract, assets, result.ArtifactSHA256); err != nil {
					return err
				}
			}
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		after, err := digestReader(file)
		if err != nil || after != result.ArtifactSHA256 {
			return errors.New("artifact changed during verification")
		}
		if receipt != "" {
			if err := writeNativeReceipt(receipt, artifact, source, assets, wheel, after); err != nil {
				return err
			}
		}
	}
	result.Status = "PASS"
	return nil
}
