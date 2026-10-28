package enginewheel

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	boundary "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"
	"go.yaml.in/yaml/v3"
)

const PublicationContract = boundary.Path

func validateCanonicalVersion(root, kind, projected string) error {
	p, err := inside(root, "src/ptsip/engines/"+kind+"/version.yaml")
	if err != nil {
		return err
	}
	data, err := os.ReadFile(p)
	if err != nil || len(data) > 65536 {
		return fmt.Errorf("canonical engine version is missing or oversized")
	}
	var version map[string]any
	if err := yaml.Unmarshal(data, &version); err != nil || len(version) != 3 || version["schema_version"] != "ptsip-engine-version/v1" || version["engine"] != kind || version["version"] != projected || !versionPattern.MatchString(projected) {
		return fmt.Errorf("canonical version.yaml and registered engine projection diverged")
	}
	return nil
}

type PublicationOptions struct {
	GateOptions
	VerifiedEngines, Input, Output string
}
type PublicationBundle struct {
	SchemaVersion     string            `json:"schema_version"`
	Kind              string            `json:"kind"`
	EngineID          string            `json:"engine_id"`
	Version           string            `json:"version"`
	Distribution      string            `json:"distribution"`
	Tag               string            `json:"tag"`
	SourceSHA         string            `json:"source_sha"`
	VerificationRunID int64             `json:"verification_run_id"`
	Files             map[string]string `json:"files_sha256"`
	Wheels            []string          `json:"wheels"`
	PyPIPending       bool              `json:"-"`
	Directory         string            `json:"-"`
}

func AdmitPublication(root string) error { return boundary.RequirePublication(root) }

// Reconstruct each wheel with the current registered assembler and its embedded
// binary. Byte equality covers membership, metadata, RECORD, license and origin.
func validatePublicationBundle(root string, opts PublicationOptions) (PublicationBundle, error) {
	var bundle PublicationBundle
	if !has(kinds, opts.Kind) || !sourcePattern.MatchString(opts.SourceSHA) || opts.RunID <= 0 {
		return bundle, fmt.Errorf("one registered engine and exact verification source/run are required")
	}
	input, err := inside(root, opts.Input)
	if err != nil {
		return bundle, err
	}
	var index struct {
		Entries []engine `json:"official_engine_entries"`
	}
	if err := document(root, "src/ptsip/engines/index.json", &index); err != nil {
		return bundle, err
	}
	var selected engine
	for _, candidate := range index.Entries {
		if strings.EqualFold(candidate.Kind, opts.Kind) {
			selected = candidate
		}
	}
	if selected.ID == "" || !versionPattern.MatchString(selected.Version) {
		return bundle, fmt.Errorf("engine identity/version is not registered")
	}
	if err := validateCanonicalVersion(root, opts.Kind, selected.Version); err != nil {
		return bundle, err
	}
	var catalog struct {
		Prefixes map[string]string `json:"tag_prefixes"`
	}
	if err := document(root, ".ptsip/contracts/release/release-contract.json", &catalog); err != nil {
		return bundle, err
	}
	if catalog.Prefixes["engine"] == "" {
		return bundle, fmt.Errorf("engine tag prefix is not registered")
	}
	files := map[string]string{}
	err = filepath.WalkDir(input, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked publication input is forbidden")
		}
		if !entry.IsDir() && (strings.HasSuffix(p, ".whl") || strings.HasSuffix(p, ".whl.receipt.json")) {
			if _, ok := files[entry.Name()]; ok {
				return fmt.Errorf("duplicated publication input: %s", entry.Name())
			}
			files[entry.Name()] = p
		}
		return nil
	})
	if err != nil || len(files) != 6 {
		return bundle, fmt.Errorf("exact three wheels and three receipts required: %v", err)
	}
	work, err := os.MkdirTemp("", "ptsip-engine-publication-")
	if err != nil {
		return bundle, err
	}
	defer os.RemoveAll(work)
	bundle = PublicationBundle{SchemaVersion: "ptsip-engine-publication-receipt/v1", Kind: opts.Kind, EngineID: selected.ID, Version: selected.Version, Distribution: "ptsip-engine-" + opts.Kind, Tag: catalog.Prefixes["engine"] + opts.Kind + "-v" + selected.Version, SourceSHA: opts.SourceSHA, VerificationRunID: opts.RunID, Files: map[string]string{}, Wheels: []string{}}
	wd := strings.ReplaceAll(bundle.Distribution, "-", "_")
	for target, host := range hosts {
		name := wd + "-" + selected.Version + "-py3-none-" + host.Platform + ".whl"
		wheel, err := os.ReadFile(files[name])
		if err != nil || len(wheel) > 268435456 {
			return PublicationBundle{}, fmt.Errorf("missing or oversized wheel: %s", name)
		}
		archive, err := zip.NewReader(bytes.NewReader(wheel), int64(len(wheel)))
		if err != nil || len(archive.File) != 6 {
			return PublicationBundle{}, fmt.Errorf("wheel archive membership mismatch: %s", name)
		}
		binaryName := bundle.Distribution + host.Suffix
		scriptName := wd + "-" + selected.Version + ".data/scripts/" + binaryName
		var binary []byte
		for _, file := range archive.File {
			if file.Name != scriptName {
				continue
			}
			r, err := file.Open()
			if err != nil {
				return PublicationBundle{}, err
			}
			binary, err = io.ReadAll(io.LimitReader(r, 268435457))
			r.Close()
			if err != nil || len(binary) > 268435456 {
				return PublicationBundle{}, fmt.Errorf("native binary size is invalid")
			}
		}
		if target == "x86_64-unknown-linux-gnu" {
			if err := linuxABI(binary); err != nil {
				return PublicationBundle{}, err
			}
		}
		binaryPath := filepath.Join(work, binaryName)
		if err := os.WriteFile(binaryPath, binary, 0755); err != nil {
			return PublicationBundle{}, err
		}
		generated, err := Build(root, Options{Kind: opts.Kind, Target: target, Binary: binaryPath, Output: work, SourceSHA: opts.SourceSHA})
		if err != nil {
			return PublicationBundle{}, err
		}
		expected, err := os.ReadFile(generated.Wheel)
		if err != nil || !bytes.Equal(expected, wheel) {
			return PublicationBundle{}, fmt.Errorf("wheel is not the exact registered source-bound artifact: %s", name)
		}
		var actualReceipt, expectedReceipt map[string]any
		original, err := os.ReadFile(files[name+".receipt.json"])
		if err != nil || json.Unmarshal(original, &actualReceipt) != nil {
			return PublicationBundle{}, fmt.Errorf("wheel receipt is missing or malformed")
		}
		generatedReceipt, err := os.ReadFile(generated.Receipt)
		if err != nil || json.Unmarshal(generatedReceipt, &expectedReceipt) != nil || !reflect.DeepEqual(actualReceipt, expectedReceipt) {
			return PublicationBundle{}, fmt.Errorf("wheel receipt is unrelated or tampered: %s", name)
		}
		bundle.Files[name] = digest(wheel)
		bundle.Files[name+".receipt.json"] = digest(original)
		bundle.Wheels = append(bundle.Wheels, name)
	}
	sort.Strings(bundle.Wheels)
	output, err := filepath.Abs(filepath.Join(root, opts.Output))
	if err != nil || !strings.HasPrefix(output, root+string(os.PathSeparator)) {
		return PublicationBundle{}, fmt.Errorf("publication output must remain inside the repository")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return PublicationBundle{}, err
	}
	bundle.Directory = output
	for name, source := range files {
		data, err := os.ReadFile(source)
		if err != nil || digest(data) != bundle.Files[name] {
			return PublicationBundle{}, fmt.Errorf("artifact changed during publication preparation")
		}
		if err := os.WriteFile(filepath.Join(output, name), data, 0644); err != nil {
			return PublicationBundle{}, err
		}
	}
	return bundle, nil
}
