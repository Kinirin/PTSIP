package releasecontroller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const canonicalVersionFixture = `package_version = "0.3.8-b7"
tool_version = "T.0.3.8.b2"
profile_version = "pp.2.00"

[engines]
context = "0.2.2"
storage = "0.3.2"
validation = "0.3.4"
discovery = "0.3.3"
planning = "0.3.2"
execution = "0.3.2"
projection = "0.2.2"
`

const versionSyncFixture = `[project]
name = "PTSIP"
version = "0.3.8b6"

[tool.ptsip.release]
schema_version = "ptsip-current-composition/v2"
package_base = "0.3.8"
package_revision = 0
package_state = "b6"
tool_version = "T.0.3.8.b2"
profile_version = "pp.2.00"

[tool.ptsip.release.engines]
context = "0.2.2"
storage = "0.3.2"
validation = "0.3.4"
discovery = "0.3.3"
planning = "0.3.2"
execution = "0.3.2"
projection = "0.2.2"

[tool.ptsip.release.published]
identity = "0.3.8.fa660a"
`

var versionsForFixture = map[string]string{
	"context": "0.2.2", "storage": "0.3.2", "validation": "0.3.4",
	"discovery": "0.3.3", "planning": "0.3.2",
	"execution": "0.3.2", "projection": "0.2.2",
}

func writeVersionFixture(t *testing.T, source, project string) string {
	t.Helper()
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(absolute, []byte(contents), 0644); err != nil { t.Fatal(err) }
	}
	write("src/VERSION", source)
	write("pyproject.toml", project)
	write("src/ptsip/specdata/project-profile-contracts.yaml",
		"schema_version: ptsip-project-profile-contract-registry/v1\ncurrent: pp.2.00\ncontracts:\n  - version: pp.1.02\n")
	entries := []map[string]string{}
	for _, id := range officialEngineIDs {
		v := versionsForFixture[id]
		write("src/ptsip/engines/"+id+"/version.yaml",
			"schema_version: ptsip-engine-version/v1\nengine: "+id+"\nversion: "+v+"\n")
		entries = append(entries, map[string]string{"kind": strings.ToUpper(id), "version": v})
	}
	index, err := json.MarshalIndent(map[string]any{"schema_version": "ptsip-official-engine-index/v1", "official_engine_entries": entries}, "", "  ")
	if err != nil { t.Fatal(err) }
	write("src/ptsip/engines/index.json", string(index)+"\n")
	return root
}

func TestSyncPackageVersionFromSource(t *testing.T) {
	root := writeVersionFixture(t, canonicalVersionFixture, versionSyncFixture)
	before, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if _, err := SyncPackageVersion(root, true); err == nil || !strings.Contains(err.Error(), "run sync-version") {
		t.Fatalf("stale version must fail read-only check: %v", err)
	}
	afterCheck, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if !reflect.DeepEqual(before, afterCheck) { t.Fatal("check mode changed pyproject") }
	result, err := SyncPackageVersion(root, false)
	if err != nil || !result.Changed || result.Status != "SYNCED" || result.Version != "0.3.8-b7" ||
		result.ToolVersion != "T.0.3.8.b2" || result.ProfileVersion != "pp.2.00" ||
		result.Previous != "0.3.8b6" || len(result.CheckedFiles) != 10 {
		t.Fatalf("invalid full sync: %+v, %v", result, err)
	}
	if !reflect.DeepEqual(result.ChangedFiles, []string{"pyproject.toml"}) {
		t.Fatalf("unrelated projections changed: %+v", result.ChangedFiles)
	}
	after, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	for _, needle := range []string{
		`version = "0.3.8-b7"`, `package_state = "b7"`,
		`tool_version = "T.0.3.8.b2"`, `profile_version = "pp.2.00"`,
		`context = "0.2.2"`, `[tool.ptsip.release.published]`, `identity = "0.3.8.fa660a"`,
	} {
		if !strings.Contains(string(after), needle) { t.Fatalf("lost field: %s", needle) }
	}
	if result, err = SyncPackageVersion(root, true); err != nil || result.Status != "IN_SYNC" || result.Changed {
		t.Fatalf("check after sync failed: %+v %v", result, err)
	}
	if result, err = SyncPackageVersion(root, false); err != nil || result.Changed {
		t.Fatalf("repeat sync changed files: %+v %v", result, err)
	}
}

func TestFullCanonicalSourceProjectsIndependentVersions(t *testing.T) {
	source := strings.NewReplacer(
		`tool_version = "T.0.3.8.b2"`, `tool_version = "T.0.3.9"`,
		`profile_version = "pp.2.00"`, `profile_version = "pp.2.01"`,
		`storage = "0.3.2"`, `storage = "0.3.3"`,
	).Replace(canonicalVersionFixture)
	root := writeVersionFixture(t, source, versionSyncFixture)
	result, err := SyncPackageVersion(root, false)
	if err != nil { t.Fatal(err) }
	if result.Engines["storage"] != "0.3.3" || result.Engines["context"] != "0.2.2" {
		t.Fatalf("unrelated engine implicitly changed: %+v", result.Engines)
	}
	expected := []string{"pyproject.toml", "src/ptsip/engines/index.json", "src/ptsip/specdata/project-profile-contracts.yaml", "src/ptsip/engines/storage/version.yaml"}
	if !reflect.DeepEqual(result.ChangedFiles, expected) {
		t.Fatalf("wrong projection targets: got %v, want %v", result.ChangedFiles, expected)
	}
	project, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	for _, needle := range []string{`tool_version = "T.0.3.9"`, `profile_version = "pp.2.01"`, `storage = "0.3.3"`, `context = "0.2.2"`} {
		if !strings.Contains(string(project), needle) { t.Fatalf("missing %s", needle) }
	}
	if result, err = SyncPackageVersion(root, true); err != nil || result.Changed {
		t.Fatalf("complete composition check failed: %+v %v", result, err)
	}
}

func TestVersionSyncRejectsInvalidOrMissingSource(t *testing.T) {
	cases := []string{
		"0.3.8-b7\n",
		strings.Replace(canonicalVersionFixture, "0.3.8-b7", "0.3.8b7", 1),
		strings.Replace(canonicalVersionFixture, `tool_version = "T.0.3.8.b2"`, `tool_version = "0.3.8b2"`, 1),
		strings.Replace(canonicalVersionFixture, `profile_version = "pp.2.00"`, "", 1),
		strings.Replace(canonicalVersionFixture, `context = "0.2.2"`, "", 1),
		canonicalVersionFixture + "unknown = true\n",
		canonicalVersionFixture + "\n[not-approved]\nx = 1\n",
	}
	for i, source := range cases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			root := writeVersionFixture(t, source, versionSyncFixture)
			if _, err := SyncPackageVersion(root, false); err == nil {
				t.Fatalf("invalid version source accepted: %q", source)
			}
		})
	}
	root := writeVersionFixture(t, canonicalVersionFixture, versionSyncFixture)
	os.Remove(filepath.Join(root, "src", "VERSION"))
	if _, err := SyncPackageVersion(root, false); err == nil { t.Fatal("missing canonical source accepted") }
}

func TestVersionSyncRejectsAmbiguousProjectFields(t *testing.T) {
	cases := map[string]string{
		"missing_project": strings.Replace(versionSyncFixture, "[project]", "[other]", 1),
		"missing_version": strings.Replace(versionSyncFixture, `version = "0.3.8b6"`, `v = "0.3.8b6"`, 1),
		"duplicate_version": strings.Replace(versionSyncFixture, `version = "0.3.8b6"`, `version = "0.3.8b6"`+"\n"+`version = "0.3.8b5"`, 1),
		"missing_state": strings.Replace(versionSyncFixture, `package_state = "b6"`, "", 1),
		"missing_engine": strings.Replace(versionSyncFixture, `storage = "0.3.2"`, "", 1),
	}
	for name, project := range cases {
		t.Run(name, func(t *testing.T) {
			root := writeVersionFixture(t, canonicalVersionFixture, project)
			if _, err := SyncPackageVersion(root, false); err == nil { t.Fatal("ambiguous projection accepted") }
		})
	}
}

func TestVersionSyncPreservesCRLFAndSupportsStableRelease(t *testing.T) {
	source := strings.Replace(canonicalVersionFixture, "0.3.8-b7", "0.3.9", 1)
	project := strings.ReplaceAll(versionSyncFixture, "\n", "\r\n")
	root := writeVersionFixture(t, source, project)
	if _, err := SyncPackageVersion(root, false); err != nil { t.Fatal(err) }
	after, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if strings.Contains(strings.ReplaceAll(string(after), "\r\n", ""), "\n") { t.Fatal("CRLF rewritten") }
	for _, want := range []string{`version = "0.3.9"`, `package_base = "0.3.9"`, `package_state = ""`} {
		if !strings.Contains(string(after), want) { t.Fatalf("missing stable projection %s", want) }
	}
}

func TestPublicationEvidenceAuthorityRequiresCanonicalVersionProjection(t *testing.T) {
	TestSyncPackageVersionFromSource(t)
}
func TestPublicationEvidenceAuthorityVersionProjectionSuite(t *testing.T) {
	t.Run("full", TestSyncPackageVersionFromSource)
	t.Run("components", TestFullCanonicalSourceProjectsIndependentVersions)
	t.Run("invalid_source", TestVersionSyncRejectsInvalidOrMissingSource)
	t.Run("invalid_target", TestVersionSyncRejectsAmbiguousProjectFields)
	t.Run("crlf", TestVersionSyncPreservesCRLFAndSupportsStableRelease)
}
