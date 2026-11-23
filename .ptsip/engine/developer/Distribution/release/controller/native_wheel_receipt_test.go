package releasecontroller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

func TestNativeWheelReceiptArchiveBinding(t *testing.T) {
	versions, err := versionsource.Parse([]byte(canonicalVersionFixture))
	if err != nil {
		t.Fatal(err)
	}
	distribution := "ptsip-" + versions.DistributionVersion()
	metadata := distribution + ".dist-info/"
	scripts := distribution + ".data/scripts/"
	profileFile := "storage/profile/project-profile.schema.json"
	profileBytes := []byte("{\"$id\":\"urn:fixture:profile\"}\n")
	profile := ProfileRegistration{Path: profileFile, Digest: Digest(profileBytes)}
	toolBytes := []byte("\x7fELFtool-fixture")
	source := strings.Repeat("a", 40)
	engineIDs := map[string]string{}

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
			platform := "linux-x64"
			displayProfile := profileBytes
			if test.name == "profile" {
				displayProfile = []byte("{}\n")
			}
			receipt := NativeWheelReceipt{
				Source: source, Platform: platform, Filename: distribution + "-py3-none-linux_x86_64.whl",
				Inventory: map[string]nativeWheelComponent{
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
				if id == "context" && test.name == "stale-origin" {
					originSource = strings.Repeat("b", 40)
				}
				if id == "context" && test.name == "native-format" {
					binary = []byte("MZ" + id + "-fixture")
				}
				engineIDs[id] = id + "-engine-id"
				members[scripts+entrypoint] = binary
				receipt.Inventory[id] = nativeWheelComponent{versions.Engines[id], Digest(binary)}
				origins[id] = map[string]any{
					"schema_version": "ptsip-independent-wheel-origin/v1", "source_sha": originSource,
					"engine_id": engineIDs[id], "engine_version": versions.Engines[id], "host_target": "x86_64-unknown-linux-gnu",
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
			if test.name == "duplicate-member" {
				members[scripts+"PTSIP"] = toolBytes
			}
			var record bytes.Buffer
			rows := csv.NewWriter(&record)
			for name, data := range members {
				hash := sha256.Sum256(data)
				if test.name == "record-digest" && name == scripts+"ptsip" {
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
			switch test.name {
			case "component-digest":
				receipt.Inventory["tool"] = nativeWheelComponent{versions.ToolVersion, strings.Repeat("0", 64)}
			case "artifact-digest":
				receipt.SHA256 = strings.Repeat("0", 64)
			case "platform":
				receipt.Platform = "windows-x64"
			}
			wheel := filepath.Join(t.TempDir(), receipt.Filename)
			if err := os.WriteFile(wheel, archive.Bytes(), 0644); err != nil {
				t.Fatal(err)
			}
			err = verifyReceiptArchive(wheel, receipt, versions, profile, engineIDs)
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
	root := t.TempDir()
	git := func(args ...string) string {
		output, err := Run(root, append([]string{"git"}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(output))
	}
	git("init")
	var contract Contract
	contract.Verification.Snapshot = "snapshot.json"
	contract.Verification.Graph = "graph.json"
	documents := map[string]string{
		versionsource.Path:             "[package]\nversion=\"1.0.0\"\n",
		contract.Verification.Snapshot: "{\"snapshot\":1}\n",
		contract.Verification.Graph:    "{\"graph\":1}\n",
	}
	for p, text := range documents {
		path := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git("-c", "core.autocrlf=false", "add", ".")
	git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "Add fixture")
	sha := git("rev-parse", "HEAD")
	proof := engineVerificationProof{
		Source: sha, Sync: sha, Snapshot: Digest([]byte(documents[contract.Verification.Snapshot])),
		Graph: Digest([]byte(documents[contract.Verification.Graph])),
	}
	receipt := NativeWheelReceipt{Source: sha, Snapshot: proof.Snapshot}
	proofPath := filepath.Join(t.TempDir(), "proof.json")
	verify := func(p engineVerificationProof) error {
		t.Helper()
		if err := WriteJSON(proofPath, p); err != nil {
			t.Fatal(err)
		}
		return verifyWheelReceiptEngineProof(root, proofPath, contract, receipt)
	}
	if head, err := cleanWheelReceiptHead(root); err != nil || head != sha {
		t.Fatalf("repository fixture is not clean: %s: %v", head, err)
	}
	if err := verify(proof); err != nil {
		t.Fatal(err)
	}
	proof.Graph = strings.Repeat("0", 64)
	if err := verify(proof); err == nil || !strings.Contains(err.Error(), "digest differs from committed graph.json") {
		t.Fatalf("stale graph evidence accepted: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "work-in-progress.txt"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := cleanWheelReceiptHead(root); err == nil || !strings.Contains(err.Error(), "requires a clean checkout") {
		t.Fatalf("dirty checkout accepted: %v", err)
	}
}
