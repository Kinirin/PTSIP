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
	"os/exec"
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
// nativeComponent supplies build identity, never public pip distribution policy.
type nativeComponent struct {
	ID         string   `json:"engine_id"`
	Version    string   `json:"version"`
	Entrypoint string   `json:"entrypoint"`
	Targets    []string `json:"build_targets"`
	Permission *bool    `json:"permission_from_availability"`
	Authorized *bool    `json:"installed_is_execution_authorization"`
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

// Build assembles one registered, already-built engine into an internal
// per-engine verification artifact for subsequent integration into a complete
// PTSIP platform wheel. The intermediate transport Wheel is NOT a public consumer
// dependency or a publication grant. Build never invokes Python or Cargo.
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
		Schema     string            `json:"schema_version"`
		Selection  string            `json:"selection"`
		DefaultIDs []string          `json:"default_available_engine_ids"`
		Components []nativeComponent `json:"components"`
	}
	if err = document(root, "src/ptsip/engines/index.json", &index); err != nil {
		return Result{}, err
	}
	if err = document(root, "src/ptsip/engines/contracts/installation-plan.json", &plan); err != nil {
		return Result{}, err
	}
	// Intermediate verification owns native source/identity/target only.
	// Consumer distribution is checked at the complete PTSIP Wheel join.
	if len(index.Entries) != 7 || len(plan.Components) != 7 ||
		plan.Schema != "ptsip-engine-installation-plan/v2" ||
		plan.Selection != "REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL" {
		return Result{}, fmt.Errorf("exact seven registered native component identities required")
	}
	ids := map[string]bool{}
	registeredKinds := map[string]bool{}
	components := map[string]nativeComponent{}
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
	for _, component := range plan.Components {
		if component.ID == "" {
			return Result{}, fmt.Errorf("missing native component identity")
		}
		if _, exists := components[component.ID]; exists {
			return Result{}, fmt.Errorf("duplicate registered native component identity")
		}
		if !ids[component.ID] {
			return Result{}, fmt.Errorf("native registry and component identity contract differ")
		}
		components[component.ID] = component
	}
	if len(plan.DefaultIDs) != 7 {
		return Result{}, fmt.Errorf("exact seven default native component identities required")
	}
	defaults := map[string]bool{}
	for _, id := range plan.DefaultIDs {
		if id == "" || defaults[id] || !ids[id] {
			return Result{}, fmt.Errorf("default native component registry is incomplete or duplicated")
		}
		defaults[id] = true
	}
	if matches != 1 {
		return Result{}, fmt.Errorf("engine kind not registered exactly once")
	}
	kind := strings.ToLower(selected.Kind)
	entrypoint := "ptsip-engine-" + kind
	component := components[selected.ID]
	if !versionPattern.MatchString(selected.Version) ||
		component.Version != selected.Version || component.Entrypoint != entrypoint ||
		!has(component.Targets, opts.Target) ||
		component.Permission == nil || *component.Permission ||
		component.Authorized == nil || *component.Authorized {
		return Result{}, fmt.Errorf("native component identity, host or nonauthority contract mismatch")
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
	if len(bindings.Entries) != 1 || bindings.Entries[0].ID != selected.ID || bindings.Entries[0].Version != selected.Version || bindings.Entries[0].Entrypoint != entrypoint {
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
	if !info.Mode().IsRegular() || filepath.Base(binary) != entrypoint+h.Suffix || info.Size() < 64 || info.Size() > 268435456 {
		return Result{}, fmt.Errorf("wrong native component executable name or size")
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
	wd := strings.ReplaceAll(entrypoint, "-", "_")
	filename := wd + "-" + selected.Version + "-py3-none-" + h.Platform + ".whl"
	di := wd + "-" + selected.Version + ".dist-info/"
	script := wd + "-" + selected.Version + ".data/scripts/" + entrypoint + h.Suffix
	if _, err := inside(root, "License-Authority/LICENSE.md"); err != nil {
		return Result{}, err
	}
	license, err := exec.Command("git", "-C", root, "cat-file", "blob", opts.SourceSHA+":License-Authority/LICENSE.md").Output()
	if err != nil {
		return Result{}, fmt.Errorf("canonical source license Git blob unavailable: %w", err)
	}
	metadata := fmt.Sprintf("Metadata-Version: 2.4\nName: %s\nVersion: %s\nSummary: Native PTSIP build component for integrated host package\nRequires-Python: >=3.11\nLicense-Expression: LicenseRef-PTSIP-Community-Reciprocity-1.0\nLicense-File: LICENSE.md\nDescription-Content-Type: text/markdown\n\n# PTSIP Engine %s\n\nNative engine component version `%s` (internal verification artifact).\n\nEngine identity: `%s`\n\nSource revision: `%s`\n\nSource, platform and binary digests are bound by the packaged origin and release receipt. Installed availability does not grant runtime authorization.\n", entrypoint, selected.Version, kind, selected.Version, selected.ID, opts.SourceSHA)
	wheelMetadata := fmt.Sprintf("Wheel-Version: 1.0\nGenerator: ptsip-native-component-artifact-go/1\nRoot-Is-Purelib: false\nTag: py3-none-%s\n", h.Platform)
	origin, _ := json.Marshal(map[string]any{"schema_version": "ptsip-independent-wheel-origin/v1", "source_sha": opts.SourceSHA, "engine_id": selected.ID, "engine_version": selected.Version, "host_target": opts.Target, "entrypoint": entrypoint + h.Suffix, "binary_sha256": digest(contents), "execution_authorized": false})
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
