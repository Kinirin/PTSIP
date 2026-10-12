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
)

const PublicationContract = "developer/policy/contracts/engine-publication.v1.json"

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

func AdmitPublication(root string) error {
	if err := Admit(root); err != nil {
		return err
	}
	ctrl, err := policySection(root, "MPD-CTRL-0001", "release_execution_suspension")
	if err != nil {
		return err
	}
	scope, ok := ctrl["engine_publication"].(map[string]any)
	if !ok || scope["state"] != "ALLOWED" || scope["operation"] != "RELEASE" || scope["scope"] != "SELECTED_REGISTERED_ENGINES_ONLY" || scope["contract_ref"] != PublicationContract || scope["tag_github_release_and_pypi_publication"] != "ALLOWED_AFTER_EXACT_SOURCE_GATE" || scope["existing_tag_or_distribution_overwrite"] != "FORBIDDEN" || scope["runtime_activation_from_publication"] != false {
		return fmt.Errorf("independent engine publication is not admitted")
	}
	assure, err := policySection(root, "MPD-ASSURE-0001", "independent_engine_publication")
	if err != nil || assure["contract_ref"] != PublicationContract || assure["approval_ref"] != scope["approval_ref"] {
		return fmt.Errorf("engine publication verification owner mismatch")
	}
	identity, err := policySection(root, "MPD-INFO-0001", "independent_release_versioning")
	if err != nil || identity["contract_ref"] != ".github/checks/release/release-contract.json" || identity["changed_source_same_component_version"] != "FAIL_CLOSED" {
		return fmt.Errorf("immutable component version owner mismatch")
	}
	var contract map[string]any
	if err := document(root, PublicationContract, &contract); err != nil {
		return err
	}
	if contract["schema_version"] != "ptsip-engine-publication/v1" || contract["approval_ref"] != scope["approval_ref"] || contract["operation"] != "RELEASE" || contract["scope"] != scope["scope"] || contract["workflow_phase"] != "publish" || contract["existing_tag_or_distribution_overwrite"] != false || contract["tool_profile_package_publication"] != false || contract["runtime_activation_from_publication"] != false || contract["linux_wheel_tag"] != hosts["x86_64-unknown-linux-gnu"].Platform || !reflect.DeepEqual(contract["required_platforms"], []any{"windows-x64", "linux-x64", "macos-arm64"}) {
		return fmt.Errorf("publication contract expanded or inconsistent")
	}
	return nil
}

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
	var catalog struct {
		Prefixes map[string]string `json:"tag_prefixes"`
	}
	if err := document(root, ".github/checks/release/release-contract.json", &catalog); err != nil {
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
