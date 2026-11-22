package releasetest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLightweightSDistManifestTracksCurrentProfile(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "MANIFEST.in"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "include src/ptsip/schemas/config.json") ||
		!strings.Contains(string(b), "recursive-include src/ptsip/schemas/storage/profile *.schema.json") {
		t.Fatal("the canonical Profile catalog and schemas must be included in the sdist")
	}
	schemas := filepath.Join(root, "src", "ptsip", "schemas")
	config, err := os.ReadFile(filepath.Join(schemas, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Storage struct {
			Profile map[string]struct {
				Path      string `json:"path"`
				Digest    string `json:"schema_sha256"`
				Lifecycle string `json:"registered_lifecycle"`
			} `json:"profile"`
		} `json:"storage"`
	}
	if err = json.Unmarshal(config, &catalog); err != nil {
		t.Fatal(err)
	}
	registered := catalog.Storage.Profile["ptsip.project-profile"]
	if registered.Lifecycle != "ACTIVE" || !strings.HasPrefix(registered.Path, "storage/profile/") {
		t.Fatal("current Profile has no active central registration")
	}
	body, err := os.ReadFile(filepath.Join(schemas, filepath.FromSlash(registered.Path)))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != registered.Digest {
		t.Fatal("current Profile schema differs from its registered digest")
	}
}
