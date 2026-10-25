package engines_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

// Candidate engineering evidence only: a PASS does not authorize publication.
// Per-engine archive receipts are intermediate inputs to a complete PTSIP Wheel.
// Native binaries are probed directly; standalone pip installation is retired.
func TestIndependentEngineWheel(t *testing.T) {
	target, platform, suffix, label, err := wheelHost()
	if err != nil {
		t.Fatal(err)
	}
	root := testrepo.Root(t)
	// A revision label alone is not enough if tracked sources were edited.
	workingTree, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=no").Output()
	if err != nil {
		t.Fatalf("tracked source status unavailable: %v", err)
	}
	if len(bytes.TrimSpace(workingTree)) != 0 {
		t.Fatalf("tracked source differs from HEAD; rebuild from a clean revision")
	}
	sourceOutput, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("exact source revision unavailable: %v", err)
	}
	sourceSHA := strings.TrimSpace(string(sourceOutput))
	if len(sourceSHA) != 40 {
		t.Fatalf("unusable source revision: %q", sourceSHA)
	}
	wheelDir := os.Getenv("PTSIP_ENGINE_WHEEL_DIR")
	if wheelDir == "" {
		wheelDir = filepath.Join(root, ".artifacts", "release", "engine-wheels", label)
	} else if !filepath.IsAbs(wheelDir) {
		wheelDir = filepath.Join(root, wheelDir)
	}
	plan := document(t, "src/ptsip/engines/contracts/installation-plan.json")
	if plan["schema_version"] != "ptsip-engine-installation-plan/v2" ||
		plan["selection"] != "REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL" {
		t.Fatal("current integrated native component plan required")
	}
	components := plan["components"].([]any)
	for _, value := range entries(t) {
		entry := value.(object)
		kind := strings.ToLower(entry["kind"].(string))
		t.Run(kind, func(t *testing.T) {
			version := entry["version"].(string)
			engineID := entry["engine_id"].(string)
			distribution := "ptsip-engine-" + kind
			matched := 0
			for _, raw := range components {
				p := raw.(object)
				if p["engine_id"] == engineID {
					matched++
					if p["version"] != version || p["entrypoint"] != distribution ||
						p["distribution"] != "PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT" ||
						p["installation"] != "INSTALLED_AS_PART_OF_PTSIP_PLATFORM_WHEEL" ||
						p["installed_is_execution_authorization"] != false ||
						!containsString(p["build_targets"], target) {
						t.Fatal("registered integrated native component contract mismatch")
					}
				}
			}
			if matched != 1 {
				t.Fatalf("exact one registered native component expected: %d", matched)
			}
			filename := strings.ReplaceAll(distribution, "-", "_") + "-" + version + "-py3-none-" + platform + ".whl"
			wheel := filepath.Join(wheelDir, filename)
			receipt := wheel + ".receipt.json"
			expected := wheelExpectation{kind, engineID, version, distribution, target, platform, suffix, sourceSHA}
			if err := verifyIndependentWheel(wheel, receipt, expected); err != nil {
				t.Fatalf("wheel conformance rejected: %v", err)
			}
			// Mutation check: the verifier must reject evidence from a different source.
			wrong := expected
			wrong.sourceSHA = strings.Repeat("0", 40)
			if wrong.sourceSHA == expected.sourceSHA {
				wrong.sourceSHA = strings.Repeat("f", 40)
			}
			if err := verifyIndependentWheel(wheel, receipt, wrong); err == nil {
				t.Fatal("source SHA mismatch was accepted")
			}
			// A receipt or ZIP changed after packaging is never evidence of PASS.
			receiptBytes, err := os.ReadFile(receipt)
			if err != nil {
				t.Fatal(err)
			}
			var altered map[string]any
			if err := json.Unmarshal(receiptBytes, &altered); err != nil {
				t.Fatal(err)
			}
			altered["wheel_sha256"] = strings.Repeat("0", 64)
			alteredBytes, err := json.Marshal(altered)
			if err != nil {
				t.Fatal(err)
			}
			tamperedReceipt := filepath.Join(t.TempDir(), "tampered-receipt.json")
			if err := os.WriteFile(tamperedReceipt, alteredBytes, 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyIndependentWheel(wheel, tamperedReceipt, expected); err == nil {
				t.Fatal("modified wheel digest receipt was accepted")
			}
			if err := probeNativeComponentBinary(t, wheel, expected); err != nil {
				t.Fatalf("native component executable probe failed: %v", err)
			}
		})
	}
}

type wheelExpectation struct {
	kind, engineID, version, distribution, target, platform, suffix, sourceSHA string
}

func wheelHost() (target, platform, suffix, label string, err error) {
	switch {
	case runtime.GOOS == "windows" && runtime.GOARCH == "amd64":
		return "x86_64-pc-windows-msvc", "win_amd64", ".exe", "windows-x64", nil
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "x86_64-unknown-linux-gnu", "manylinux_2_39_x86_64", "", "linux-x64", nil
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return "aarch64-apple-darwin", "macosx_11_0_arm64", "", "macos-arm64", nil
	default:
		return "", "", "", "", fmt.Errorf("unregistered official wheel test host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func verifyIndependentWheel(wheelPath, receiptPath string, expected wheelExpectation) error {
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		return fmt.Errorf("missing independent receipt: %w", err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(data, &receipt); err != nil {
		return fmt.Errorf("invalid receipt: %w", err)
	}
	wheelData, err := os.ReadFile(wheelPath)
	if err != nil {
		return fmt.Errorf("missing independent wheel: %w", err)
	}
	if err := requireString(receipt, "schema_version", "ptsip-independent-wheel-receipt/v1"); err != nil {
		return err
	}
	for key, want := range map[string]string{
		"source_sha": expected.sourceSHA, "engine_id": expected.engineID,
		"version": expected.version, "target": expected.target,
		"wheel": filepath.Base(wheelPath), "wheel_sha256": hexSHA(wheelData),
		"release_verification": "NOT_PERFORMED_BY_ASSEMBLER",
	} {
		if err := requireString(receipt, key, want); err != nil {
			return err
		}
	}
	if err := requireFalse(receipt, "install_authorized"); err != nil {
		return err
	}
	if err := requireFalse(receipt, "runtime_authorized"); err != nil {
		return err
	}
	zr, err := zip.OpenReader(wheelPath)
	if err != nil {
		return fmt.Errorf("invalid wheel ZIP: %w", err)
	}
	defer zr.Close()
	base := strings.ReplaceAll(expected.distribution, "-", "_") + "-" + expected.version
	di := base + ".dist-info/"
	script := base + ".data/scripts/" + expected.distribution + expected.suffix
	recordName := di + "RECORD"
	wanted := map[string]bool{
		script: true, di + "METADATA": true, di + "WHEEL": true,
		di + "licenses/LICENSE.md": true, di + "ptsip-origin.json": true,
		recordName: true,
	}
	files := map[string]*zip.File{}
	for _, zf := range zr.File {
		if !wanted[zf.Name] || files[zf.Name] != nil {
			return fmt.Errorf("unexpected or duplicate wheel member: %s", zf.Name)
		}
		files[zf.Name] = zf
	}
	if len(files) != len(wanted) {
		return fmt.Errorf("wheel member count: got %d, want %d", len(files), len(wanted))
	}
	read := func(name string) ([]byte, error) {
		r, err := files[name].Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(io.LimitReader(r, 270_000_000))
	}
	meta, err := read(di + "METADATA")
	if err != nil {
		return err
	}
	for _, field := range []string{"Metadata-Version: 2.4", "Name: " + expected.distribution, "Version: " + expected.version, "Requires-Python: >=3.11", "Description-Content-Type: text/markdown"} {
		if !hasMetadataLine(meta, field) {
			return fmt.Errorf("wheel metadata mismatch: %s", field)
		}
	}
	if bytes.Contains(meta, []byte("\nX-PTSIP-")) {
		return fmt.Errorf("unregistered custom Core Metadata field")
	}
	wheelMeta, err := read(di + "WHEEL")
	if err != nil {
		return err
	}
	if !hasMetadataLine(wheelMeta, "Root-Is-Purelib: false") || !hasMetadataLine(wheelMeta, "Tag: py3-none-"+expected.platform) {
		return fmt.Errorf("wheel platform or installation scheme mismatch")
	}
	originJSON, err := read(di + "ptsip-origin.json")
	if err != nil {
		return err
	}
	var origin map[string]any
	if err := json.Unmarshal(originJSON, &origin); err != nil {
		return err
	}
	binaryDigest, ok := receipt["binary_sha256"].(string)
	if !ok {
		return fmt.Errorf("receipt binary digest missing")
	}
	for key, want := range map[string]string{
		"schema_version": "ptsip-independent-wheel-origin/v1", "source_sha": expected.sourceSHA,
		"engine_id": expected.engineID, "engine_version": expected.version,
		"host_target": expected.target, "entrypoint": expected.distribution + expected.suffix,
		"binary_sha256": binaryDigest,
	} {
		if err := requireString(origin, key, want); err != nil {
			return err
		}
	}
	if err := requireFalse(origin, "execution_authorized"); err != nil {
		return err
	}
	binaryData, err := read(script)
	if err != nil {
		return err
	}
	if err := requireString(receipt, "binary_sha256", hexSHA(binaryData)); err != nil {
		return err
	}
	if len(binaryData) < 64 {
		return fmt.Errorf("implausible native executable")
	}
	switch expected.platform {
	case "win_amd64":
		if !bytes.HasPrefix(binaryData, []byte("MZ")) {
			return fmt.Errorf("not a PE binary")
		}
	case "manylinux_2_39_x86_64":
		if !bytes.HasPrefix(binaryData, []byte{0x7f, 'E', 'L', 'F'}) {
			return fmt.Errorf("not an ELF binary")
		}
	case "macosx_11_0_arm64":
		if !bytes.HasPrefix(binaryData, []byte{0xcf, 0xfa, 0xed, 0xfe}) && !bytes.HasPrefix(binaryData, []byte{0xfe, 0xed, 0xfa, 0xcf}) {
			return fmt.Errorf("not a Mach-O binary")
		}
	}
	recordData, err := read(recordName)
	if err != nil {
		return err
	}
	rows, err := csv.NewReader(bytes.NewReader(recordData)).ReadAll()
	if err != nil {
		return fmt.Errorf("invalid RECORD CSV: %w", err)
	}
	if len(rows) != len(files) {
		return fmt.Errorf("RECORD coverage mismatch")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if len(row) != 3 {
			return fmt.Errorf("invalid RECORD row")
		}
		name := row[0]
		if files[name] == nil || seen[name] {
			return fmt.Errorf("missing or duplicate RECORD entry: %s", name)
		}
		seen[name] = true
		if name == recordName {
			if row[1] != "" || row[2] != "" {
				return fmt.Errorf("RECORD self hash must be blank")
			}
			continue
		}
		member, err := read(name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(member)
		wantHash := "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:])
		if row[1] != wantHash || row[2] != strconv.Itoa(len(member)) {
			return fmt.Errorf("RECORD member digest/size mismatch: %s", name)
		}
	}
	return nil
}

func containsString(value any, expected string) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func hexSHA(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hasMetadataLine(content []byte, line string) bool {
	for _, row := range strings.Split(string(content), "\n") {
		if strings.TrimSuffix(row, "\r") == line {
			return true
		}
	}
	return false
}

func requireString(obj map[string]any, key, want string) error {
	if got, ok := obj[key].(string); !ok || got != want {
		return fmt.Errorf("%s mismatch: got %v, want %s", key, obj[key], want)
	}
	return nil
}

func requireFalse(obj map[string]any, key string) error {
	if got, ok := obj[key].(bool); !ok || got {
		return fmt.Errorf("%s must explicitly be false", key)
	}
	return nil
}

// Probe an exact named native member after archive/receipt/RECORD verification.
// The complete PTSIP Wheel join, not this per-engine test, verifies pip install.
func probeNativeComponentBinary(t *testing.T, wheel string, expected wheelExpectation) error {
	t.Helper()
	archive, err := zip.OpenReader(wheel)
	if err != nil {
		return fmt.Errorf("open verified native archive: %w", err)
	}
	defer archive.Close()
	base := strings.ReplaceAll(expected.distribution, "-", "_") + "-" + expected.version
	script := base + ".data/scripts/" + expected.distribution + expected.suffix
	for _, member := range archive.File {
		if member.Name != script {
			continue
		}
		if member.UncompressedSize64 < 64 || member.UncompressedSize64 > 268435456 {
			return fmt.Errorf("invalid native member size")
		}
		reader, err := member.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(reader, 268435457))
		reader.Close()
		if err != nil || len(data) < 64 || len(data) > 268435456 {
			return fmt.Errorf("native member read failed: %v", err)
		}
		if hexSHA(data) != expectedBinaryHash(wheel, expected, script) {
			return fmt.Errorf("native executable changed after archive receipt verification")
		}
		binary := filepath.Join(t.TempDir(), expected.distribution+expected.suffix)
		if err := os.WriteFile(binary, data, 0700); err != nil {
			return err
		}
		cmd := exec.Command(binary, "--manifest")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("native component --manifest: %w (%s)", err, out)
		}
		var manifest map[string]any
		if err := json.Unmarshal(out, &manifest); err != nil {
			return fmt.Errorf("native component manifest JSON: %w", err)
		}
		for key, want := range map[string]string{
			"engine_id": expected.engineID,
			"implementation_version": expected.version,
		} {
			if err := requireString(manifest, key, want); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("missing native component executable")
}

// Native hash comparison reads the source-bound receipt rather than a
// separately installed pip distribution.
func expectedBinaryHash(wheel string, expected wheelExpectation, script string) string {
	_ = script
	data, err := os.ReadFile(wheel + ".receipt.json")
	if err != nil {
		return ""
	}
	var receipt map[string]any
	if json.Unmarshal(data, &receipt) != nil {
		return ""
	}
	if receipt["engine_id"] != expected.engineID ||
		receipt["source_sha"] != expected.sourceSHA {
		return ""
	}
	value, _ := receipt["binary_sha256"].(string)
	return value
}
