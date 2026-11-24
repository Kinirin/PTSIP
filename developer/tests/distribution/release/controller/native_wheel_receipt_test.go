package releasecontroller_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	. "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

type fixtureWheelComponent struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type fixtureWheelReceipt struct {
	Source    string                           `json:"source_sha"`
	Snapshot  string                           `json:"verified_snapshot_sha256"`
	Platform  string                           `json:"platform"`
	Filename  string                           `json:"filename"`
	Inventory map[string]fixtureWheelComponent `json:"components"`
	SHA256    string                           `json:"wheel_sha256"`
}

type fixtureEngineProof struct {
	Source   string `json:"source_sha"`
	Sync     string `json:"sync_sha"`
	Snapshot string `json:"snapshot_sha256"`
	Graph    string `json:"dependency_graph_sha256"`
}

type nativeWheelFixture struct {
	root         string
	versions     VersionSource
	profileFile  string
	profileBytes []byte
	engineIDs    map[string]string
	proof        fixtureEngineProof
}

func newNativeWheelFixture(t *testing.T) nativeWheelFixture {
	t.Helper()
	source := testrepo.Root(t)
	fixture := nativeWheelFixture{root: t.TempDir(), engineIDs: map[string]string{}}
	versions, err := ReadSourceVersions(source)
	if err != nil {
		t.Fatal(err)
	}
	fixture.versions = versions
	contract, err := ReadContract(source)
	if err != nil {
		t.Fatal(err)
	}
	profiles, _, err := ReadProfileRegistrations(source)
	if err != nil {
		t.Fatal(err)
	}
	fixture.profileFile = profiles[versionsource.ProfileKey].Path
	profilePath := "src/ptsip/schemas/" + fixture.profileFile
	paths := []string{
		".ptsip/contracts/release/index.json", boundary.Path, boundary.CompositionPath, boundary.CompositionSchema,
		versionsource.Path, "pyproject.toml", ProfileConfigPath, profilePath,
		"src/ptsip/common_contract/registry/manifest.json",
		contract.Sources.EngineRegistry, contract.Verification.Snapshot, contract.Verification.Graph,
	}
	var index struct {
		Entries []struct {
			Kind string `json:"kind"`
			ID   string `json:"engine_id"`
		} `json:"official_engine_entries"`
	}
	if err := ReadJSON(source, contract.Sources.EngineRegistry, &index); err != nil {
		t.Fatal(err)
	}
	for _, row := range index.Entries {
		name := strings.ToLower(row.Kind)
		fixture.engineIDs[name] = row.ID
		prefix := "src/ptsip/engines/" + name + "/"
		for _, file := range []string{"manifest.json", "contract.json", "execution-bindings.json", "version.yaml", "Cargo.toml", "Cargo.lock"} {
			paths = append(paths, prefix+file)
		}
	}
	for _, p := range paths {
		raw, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(fixture.root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0644); err != nil {
			t.Fatal(err)
		}
		switch p {
		case profilePath:
			fixture.profileBytes = raw
		case contract.Verification.Snapshot:
			fixture.proof.Snapshot = Digest(raw)
		case contract.Verification.Graph:
			fixture.proof.Graph = Digest(raw)
		}
	}
	git := func(args ...string) string {
		t.Helper()
		output, err := Run(fixture.root, append([]string{"git"}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(output))
	}
	git("init")
	git("-c", "core.autocrlf=false", "add", ".")
	git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "Add fixture")
	fixture.proof.Source = git("rev-parse", "HEAD")
	fixture.proof.Sync = fixture.proof.Source
	if status := git("status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("repository fixture is not clean: %s", status)
	}
	return fixture
}

func (f nativeWheelFixture) check(t *testing.T, scenario string) error {
	t.Helper()
	versions := f.versions
	distribution := "ptsip-" + versions.DistributionVersion()
	metadata := distribution + ".dist-info/"
	scripts := distribution + ".data/scripts/"
	profileFile, profileBytes := f.profileFile, f.profileBytes
	toolBytes := []byte("\x7fELFtool-fixture")
	source := f.proof.Source
	platform := "linux-x64"
	displayProfile := profileBytes
	if scenario == "profile" {
		displayProfile = []byte("{}\n")
	}
	receipt := fixtureWheelReceipt{
		Source: source, Snapshot: f.proof.Snapshot, Platform: platform, Filename: distribution + "-py3-none-linux_x86_64.whl",
		Inventory: map[string]fixtureWheelComponent{
			"tool":     {versions.ToolVersion, Digest(toolBytes)},
			"profiles": {versions.ProfileVersion, Digest(displayProfile)},
		},
	}
	members := map[string][]byte{
		metadata + "METADATA":          []byte("Name: PTSIP\nVersion: " + versions.DistributionVersion() + "\n\n"),
		metadata + "WHEEL":             []byte("Root-Is-Purelib: false\nTag: py3-none-linux_x86_64\n"),
		"ptsip/schemas/" + profileFile: displayProfile,
		scripts + "ptsip":              toolBytes,
	}
	origins := map[string]map[string]any{}
	for _, id := range versionsource.EngineIDs() {
		entrypoint := "ptsip-engine-" + id
		binary := []byte("\x7fELF" + id + "-fixture")
		originSource := source
		if id == "context" && scenario == "stale-origin" {
			originSource = strings.Repeat("b", 40)
		}
		if id == "context" && scenario == "native-format" {
			binary = []byte("MZ" + id + "-fixture")
		}
		members[scripts+entrypoint] = binary
		receipt.Inventory[id] = fixtureWheelComponent{versions.Engines[id], Digest(binary)}
		origins[id] = map[string]any{
			"schema_version": "ptsip-independent-wheel-origin/v1", "source_sha": originSource,
			"engine_id": f.engineIDs[id], "engine_version": versions.Engines[id], "host_target": "x86_64-unknown-linux-gnu",
			"entrypoint": entrypoint, "binary_sha256": Digest(binary), "execution_authorized": false,
		}
	}
	bundle, err := json.Marshal(map[string]any{
		"schema_version": "ptsip-bundled-engine-origins/v1", "package_version": versions.DistributionVersion(),
		"platform": platform, "execution_authorized": false, "engines": origins,
	})
	if err != nil {
		t.Fatal(err)
	}
	members[metadata+"ptsip-engines.json"] = bundle
	if scenario == "duplicate-member" {
		members[scripts+"PTSIP"] = toolBytes
	}
	var record bytes.Buffer
	rows := csv.NewWriter(&record)
	for name, data := range members {
		hash := sha256.Sum256(data)
		if scenario == "record-digest" && name == scripts+"ptsip" {
			hash = sha256.Sum256([]byte("stale"))
		}
		if err := rows.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(hash[:]), strconv.Itoa(len(data))}); err != nil {
			t.Fatal(err)
		}
	}
	if err := rows.Write([]string{metadata + "RECORD", "", ""}); err != nil {
		t.Fatal(err)
	}
	rows.Flush()
	if err := rows.Error(); err != nil {
		t.Fatal(err)
	}
	members[metadata+"RECORD"] = record.Bytes()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, data := range members {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	receipt.SHA256 = Digest(archive.Bytes())
	switch scenario {
	case "component-digest":
		receipt.Inventory["tool"] = fixtureWheelComponent{versions.ToolVersion, strings.Repeat("0", 64)}
	case "artifact-digest":
		receipt.SHA256 = strings.Repeat("0", 64)
	case "platform":
		receipt.Platform = "windows-x64"
	}
	temp := t.TempDir()
	wheel := filepath.Join(temp, receipt.Filename)
	if err := os.WriteFile(wheel, archive.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(temp, "native-wheel-receipt.json")
	proofPath := filepath.Join(temp, "engine-proof.json")
	if err := WriteJSON(receiptPath, receipt); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSON(proofPath, f.proof); err != nil {
		t.Fatal(err)
	}
	_, err = VerifyNativeWheelReceipt(f.root, wheel, receiptPath, proofPath)
	return err
}

func TestNativeWheelReceiptArchiveBinding(t *testing.T) {
	fixture := newNativeWheelFixture(t)
	for _, test := range []struct{ name, expected string }{
		{"exact", ""},
		{"component-digest", "version or member digest differs for tool"},
		{"artifact-digest", "artifact SHA-256 differs"},
		{"platform", "filename tag differs"},
		{"profile", "Profile schema differs"},
		{"stale-origin", "engine origin differs for context"},
		{"native-format", "binary does not match platform"},
		{"record-digest", "native wheel RECORD SHA-256 differs"},
		{"duplicate-member", "invalid or ambiguous native wheel member"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := fixture.check(t, test.name)
			if test.expected == "" && err != nil {
				t.Fatal(err)
			}
			if test.expected != "" && (err == nil || !strings.Contains(err.Error(), test.expected)) {
				t.Fatalf("expected %q, got %v", test.expected, err)
			}
		})
	}
}

func TestNativeWheelReceiptEngineProofUsesCommittedBytes(t *testing.T) {
	fixture := newNativeWheelFixture(t)
	if err := fixture.check(t, "exact"); err != nil {
		t.Fatal(err)
	}
	graph := fixture.proof.Graph
	fixture.proof.Graph = strings.Repeat("0", 64)
	if err := fixture.check(t, "exact"); err == nil || !strings.Contains(err.Error(), "digest differs from committed src/ptsip/engines/contracts/dependency-graph.json") {
		t.Fatalf("stale graph evidence accepted: %v", err)
	}
	fixture.proof.Graph = graph
	if err := os.WriteFile(filepath.Join(fixture.root, "work-in-progress.txt"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := fixture.check(t, "exact"); err == nil || !strings.Contains(err.Error(), "requires a clean checkout") {
		t.Fatalf("dirty checkout accepted: %v", err)
	}
}
