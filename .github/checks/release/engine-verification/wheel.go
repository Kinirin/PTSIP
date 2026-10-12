package enginewheel

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Options struct{ Kind, Target, Binary, Output, SourceSHA string }
type Result struct {
	Status  string `json:"status"`
	Wheel   string `json:"wheel"`
	Receipt string `json:"receipt"`
}
type engine struct {
	Kind     string `json:"kind"`
	ID       string `json:"engine_id"`
	Version  string `json:"version"`
	Manifest string `json:"manifest_ref"`
	Binding  string `json:"execution_binding_ref"`
}
type registration struct {
	ID           string   `json:"engine_id"`
	Version      string   `json:"version"`
	Distribution string   `json:"package_distribution"`
	Format       string   `json:"distribution"`
	Installation string   `json:"installation"`
	Targets      []string `json:"build_targets"`
	Permission   *bool    `json:"permission_from_availability"`
	Authorized   *bool    `json:"installed_is_execution_authorization"`
}
type host struct {
	Platform, Suffix string
	Magic            [][]byte
}

var hosts = map[string]host{
	"x86_64-pc-windows-msvc":   {"win_amd64", ".exe", [][]byte{[]byte("MZ")}},
	"x86_64-unknown-linux-gnu": {"manylinux_2_39_x86_64", "", [][]byte{[]byte("\x7fELF")}},
	"aarch64-apple-darwin":     {"macosx_11_0_arm64", "", [][]byte{{0xcf, 0xfa, 0xed, 0xfe}, {0xfe, 0xed, 0xfa, 0xcf}}},
}
var sourcePattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
var kinds = []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"}

func inside(root, ref string) (string, error) {
	if ref == "" || ref == "." || ref == ".." || path.IsAbs(ref) || filepath.IsAbs(ref) || strings.ContainsAny(ref, "\\:") || path.Clean(ref) != ref || strings.HasPrefix(ref, "../") {
		return "", fmt.Errorf("untrusted repository reference: %s", ref)
	}
	target := root
	for _, part := range strings.Split(ref, "/") {
		target = filepath.Join(target, part)
		info, err := os.Lstat(target)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlinked package source is not admitted")
		}
	}
	return target, nil
}
func document(root, ref string, value any) error {
	p, err := inside(root, ref)
	if err != nil {
		return err
	}
	info, err := os.Stat(p)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 2097152 {
		return fmt.Errorf("contract metadata too large or not regular: %s", ref)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, value)
}
func digest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func has(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// Build assembles one registered, already-built engine. It grants no release,
// installation or execution permission and never invokes Python or Cargo.
func Build(root string, opts Options) (Result, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Result{}, err
	}
	if !sourcePattern.MatchString(opts.SourceSHA) {
		return Result{}, fmt.Errorf("exact 40-character source SHA required")
	}
	if !has(kinds, strings.ToLower(opts.Kind)) {
		return Result{}, fmt.Errorf("engine kind not registered")
	}
	h, ok := hosts[opts.Target]
	if !ok {
		return Result{}, fmt.Errorf("unsupported official engine build target")
	}
	var index struct {
		Entries []engine `json:"official_engine_entries"`
	}
	var plan struct {
		Packages []registration `json:"packages"`
	}
	if err = document(root, "src/ptsip/engines/index.json", &index); err != nil {
		return Result{}, err
	}
	if err = document(root, "src/ptsip/engines/contracts/installation-plan.json", &plan); err != nil {
		return Result{}, err
	}
	if len(index.Entries) != 7 || len(plan.Packages) != 7 {
		return Result{}, fmt.Errorf("exact seven independent official engines required")
	}
	ids := map[string]bool{}
	registeredKinds := map[string]bool{}
	packages := map[string]registration{}
	var selected engine
	matches := 0
	for _, e := range index.Entries {
		kind := strings.ToLower(e.Kind)
		if !has(kinds, kind) || registeredKinds[kind] {
			return Result{}, fmt.Errorf("unregistered or duplicate engine kind")
		}
		registeredKinds[kind] = true
		if e.ID == "" || ids[e.ID] {
			return Result{}, fmt.Errorf("duplicate official engine identity")
		}
		ids[e.ID] = true
		if strings.EqualFold(e.Kind, opts.Kind) {
			selected = e
			matches++
		}
	}
	for _, p := range plan.Packages {
		if p.ID == "" {
			return Result{}, fmt.Errorf("missing package identity")
		}
		if _, ok := packages[p.ID]; ok {
			return Result{}, fmt.Errorf("duplicate official package identity")
		}
		if !ids[p.ID] {
			return Result{}, fmt.Errorf("official registry and installation contract diverged")
		}
		packages[p.ID] = p
	}
	if matches != 1 {
		return Result{}, fmt.Errorf("engine kind not registered exactly once")
	}
	kind := strings.ToLower(selected.Kind)
	distribution := "ptsip-engine-" + kind
	p := packages[selected.ID]
	if !versionPattern.MatchString(selected.Version) || p.Version != selected.Version || p.Distribution != distribution || p.Format != "INDEPENDENT_VERSIONED_NATIVE_WHEEL" || p.Installation != "AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY" || !has(p.Targets, opts.Target) || p.Permission == nil || *p.Permission || p.Authorized == nil || *p.Authorized {
		return Result{}, fmt.Errorf("independent auto-installed package contract mismatch")
	}
	var manifest struct {
		ID      string `json:"engine_id"`
		Version string `json:"implementation_version"`
	}
	if err = document(root, selected.Manifest, &manifest); err != nil {
		return Result{}, err
	}
	if manifest.ID != selected.ID || manifest.Version != selected.Version {
		return Result{}, fmt.Errorf("engine manifest identity/version mismatch")
	}
	var bindings struct {
		Entries []struct {
			ID         string `json:"engine_id"`
			Version    string `json:"engine_version"`
			Entrypoint string `json:"entrypoint"`
		} `json:"bindings"`
	}
	if err = document(root, selected.Binding, &bindings); err != nil {
		return Result{}, err
	}
	if len(bindings.Entries) != 1 || bindings.Entries[0].ID != selected.ID || bindings.Entries[0].Version != selected.Version || bindings.Entries[0].Entrypoint != distribution {
		return Result{}, fmt.Errorf("registered execution binding mismatch")
	}
	binary, err := filepath.Abs(opts.Binary)
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(binary)
	if err != nil {
		return Result{}, err
	}
	if !info.Mode().IsRegular() || filepath.Base(binary) != distribution+h.Suffix || info.Size() < 64 || info.Size() > 268435456 {
		return Result{}, fmt.Errorf("wrong independent executable name or size")
	}
	contents, err := os.ReadFile(binary)
	if err != nil {
		return Result{}, err
	}
	valid := false
	for _, magic := range h.Magic {
		valid = valid || bytes.HasPrefix(contents, magic)
	}
	if !valid {
		return Result{}, fmt.Errorf("engine artifact does not match declared platform")
	}
	wd := strings.ReplaceAll(distribution, "-", "_")
	filename := wd + "-" + selected.Version + "-py3-none-" + h.Platform + ".whl"
	di := wd + "-" + selected.Version + ".dist-info/"
	script := wd + "-" + selected.Version + ".data/scripts/" + distribution + h.Suffix
	licensePath, err := inside(root, "License-Authority/LICENSE.md")
	if err != nil {
		return Result{}, err
	}
	license, err := os.ReadFile(licensePath)
	if err != nil {
		return Result{}, err
	}
	metadata := fmt.Sprintf("Metadata-Version: 2.4\nName: %s\nVersion: %s\nSummary: Independently versioned official PTSIP native engine\nRequires-Python: >=3.11\nLicense-Expression: LicenseRef-PTSIP-Community-Reciprocity-1.0\nLicense-File: LICENSE.md\nDescription-Content-Type: text/markdown\n\n# PTSIP Engine %s\n\nIndependent native engine version `%s`.\n\nEngine identity: `%s`\n\nSource revision: `%s`\n\nSource, platform and binary digests are bound by the packaged origin and release receipt. Installed availability does not grant runtime authorization.\n", distribution, selected.Version, kind, selected.Version, selected.ID, opts.SourceSHA)
	wheelMetadata := fmt.Sprintf("Wheel-Version: 1.0\nGenerator: ptsip-independent-engine-wheel-go/1\nRoot-Is-Purelib: false\nTag: py3-none-%s\n", h.Platform)
	origin, _ := json.Marshal(map[string]any{"schema_version": "ptsip-independent-wheel-origin/v1", "source_sha": opts.SourceSHA, "engine_id": selected.ID, "engine_version": selected.Version, "host_target": opts.Target, "entrypoint": distribution + h.Suffix, "binary_sha256": digest(contents), "execution_authorized": false})
	members := map[string][]byte{script: contents, di + "METADATA": []byte(metadata), di + "WHEEL": []byte(wheelMetadata), di + "licenses/LICENSE.md": license, di + "ptsip-origin.json": append(origin, '\n')}
	names := []string{}
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	var record bytes.Buffer
	csvWriter := csv.NewWriter(&record)
	for _, name := range names {
		sum := sha256.Sum256(members[name])
		_ = csvWriter.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), strconv.Itoa(len(members[name]))})
	}
	recordName := di + "RECORD"
	_ = csvWriter.Write([]string{recordName, "", ""})
	csvWriter.Flush()
	if err = csvWriter.Error(); err != nil {
		return Result{}, err
	}
	members[recordName] = record.Bytes()
	names = append(names, recordName)
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
		mode := os.FileMode(0644)
		if name == script {
			mode = 0755
		}
		header.SetMode(mode)
		w, e := zw.CreateHeader(header)
		if e != nil {
			return Result{}, e
		}
		if _, e = w.Write(members[name]); e != nil {
			return Result{}, e
		}
	}
	if err = zw.Close(); err != nil {
		return Result{}, err
	}
	output, err := filepath.Abs(opts.Output)
	if err != nil {
		return Result{}, err
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return Result{}, err
	}
	destination := filepath.Join(output, filename)
	receipt := destination + ".receipt.json"
	raw, _ := json.MarshalIndent(map[string]any{"schema_version": "ptsip-independent-wheel-receipt/v1", "source_sha": opts.SourceSHA, "engine_id": selected.ID, "version": selected.Version, "target": opts.Target, "wheel": filename, "wheel_sha256": digest(archive.Bytes()), "binary_sha256": digest(contents), "install_authorized": false, "runtime_authorized": false, "release_verification": "NOT_PERFORMED_BY_ASSEMBLER"}, "", "  ")
	if err = os.WriteFile(destination, archive.Bytes(), 0644); err != nil {
		return Result{}, err
	}
	if err = os.WriteFile(receipt, append(raw, '\n'), 0644); err != nil {
		return Result{}, err
	}
	return Result{"PACKAGED_UNRELEASED", destination, receipt}, nil
}
