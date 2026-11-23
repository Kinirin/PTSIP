package releasecontroller

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const canonicalVersionFixture = `[package]
version = "0.3.8-b7"
[tool]
version = "T.0.3.8.b2"
[RBAC]
RA_list = "v.1"
RAC_list = "v.1"
AEC_list = "v.1"
[common_contract]
registry_revision = "v2"
[schemas]
"ptsip.policy.v4" = "v4"
"ptsip.project-profile" = "pp.2.00"

[engines]
context = "0.2.2"
storage = "0.3.2"
validation = "0.3.4"
discovery = "0.3.3"
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
execution = "0.3.2"
projection = "0.2.2"

[tool.ptsip.release.published]
identity = "0.3.8.fa660a"
`

var versionsForFixture = map[string]string{
	"context": "0.2.2", "storage": "0.3.2", "validation": "0.3.4",
	"discovery": "0.3.3",
	"execution": "0.3.2", "projection": "0.2.2",
}

func writeVersionFixture(t *testing.T, source, project string) string {
	t.Helper()
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("src/VERSION", source)
	write("pyproject.toml", project)
	write("src/ptsip/common_contract/registry/manifest.json", `{"revision":"v2"}`)
	write(ProfileConfigPath, `{"Policy":{"ptsip.policy.v4":{"version":"v4"}},"storage":{"Responsibility-Authority":{"RA_list":{"version":"v.1"},"RAC_list":{"version":"v.1"},"AEC_list":{"version":"v.1"}},"profile":{"ptsip.project-profile":{
		"$id":"urn:ptsip:storage:contract:profile:official",
		"version":"pp.2.00",
		"path":"storage/profile/project-profile.schema.json",
		"schema_sha256":"53420b98df6e9acb379ef35a88d4280b064fd3e8e2184925b2b4d607d743c6ad",
		"registered_lifecycle":"ACTIVE"
	}}}}`)
	entries := []map[string]string{}
	selected, _ := ReadSourceVersions(root)
	for _, id := range officialEngineIDs {
		v := versionsForFixture[id]
		target := selected.Engines[id]
		if target == "" {
			target = v
		}
		dir, engine, cap, crate := "src/ptsip/engines/"+id+"/", "engine-"+id, "cap-"+id, "ptsip-engine-"+id
		contract := fmt.Sprintf(`{"engine_id":%q,"contract_id":%q,"version":%q}`, engine, cap, target)
		write(dir+"contract.json", contract)
		write(dir+"manifest.json", fmt.Sprintf(`{"engine_id":%q,"origin":"OFFICIAL","implementation_version":%q,"capabilities":[{"id":%q,"version":%q,"path":%q,"sha256":"%x"}]}`, engine, v, cap, target, dir+"contract.json", sha256.Sum256([]byte(contract))))
		write(dir+"execution-bindings.json", fmt.Sprintf(`{"bindings":[{"engine_id":%q,"capability_id":%q,"contract_version":%q,"engine_version":%q}]}`, engine, cap, target, v))
		write(dir+"Cargo.toml", fmt.Sprintf("[package]\nname = %q\nversion = %q\n", crate, v))
		write(dir+"Cargo.lock", fmt.Sprintf("version = 4\n[[package]]\nname = %q\nversion = %q\n[[package]]\nname = \"dependency\"\nversion = \"1.0.0\"\n", crate, v))
		write(dir+"version.yaml",
			"schema_version: ptsip-engine-version/v1\nengine: "+id+"\nversion: "+v+"\n")
		entries = append(entries, map[string]string{"kind": strings.ToUpper(id), "version": v,
			"engine_id": engine, "capability_id": cap, "contract_ref": dir + "contract.json", "manifest_ref": dir + "manifest.json",
			"execution_binding_ref": dir + "execution-bindings.json", "cargo_manifest_ref": dir + "Cargo.toml"})
	}
	index, err := json.MarshalIndent(map[string]any{"schema_version": "ptsip-official-engine-index/v1", "official_engine_entries": entries}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
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
	if !reflect.DeepEqual(before, afterCheck) {
		t.Fatal("check mode changed pyproject")
	}
	result, err := SyncPackageVersion(root, false)
	if err != nil || !result.Changed || result.Status != "SYNCED" || result.Version != "0.3.8-b7" ||
		result.ToolVersion != "T.0.3.8.b2" || result.ProfileVersion != "pp.2.00" ||
		result.Previous != "0.3.8b6" || len(result.CheckedFiles) != 41 {
		t.Fatalf("invalid full sync: %+v, %v", result, err)
	}
	if !reflect.DeepEqual(result.ChangedFiles, []string{"pyproject.toml"}) {
		t.Fatalf("unrelated projections changed: %+v", result.ChangedFiles)
	}
	after, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	for _, needle := range []string{
		`version = "0.3.8b7"`, `package_state = "b7"`,
		`tool_version = "T.0.3.8.b2"`, `profile_version = "pp.2.00"`,
		`context = "0.2.2"`, `[tool.ptsip.release.published]`, `identity = "0.3.8.fa660a"`,
	} {
		if !strings.Contains(string(after), needle) {
			t.Fatalf("lost field: %s", needle)
		}
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
		`version = "T.0.3.8.b2"`, `version = "T.0.3.9"`,
		`"ptsip.project-profile" = "pp.2.00"`, `"ptsip.project-profile" = "pp.2.01"`,
		`storage = "0.3.2"`, `storage = "0.3.3"`,
	).Replace(canonicalVersionFixture)
	root := writeVersionFixture(t, source, versionSyncFixture)
	registryPath := filepath.Join(root, filepath.FromSlash(ProfileConfigPath))
	registeredBefore, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	registered := strings.Replace(string(registeredBefore), "pp.2.00", "pp.2.01", 1)
	if err := os.WriteFile(registryPath, []byte(registered), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := SyncPackageVersion(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Engines["storage"] != "0.3.3" || result.Engines["context"] != "0.2.2" {
		t.Fatalf("unrelated engine implicitly changed: %+v", result.Engines)
	}
	expected := []string{"pyproject.toml", "src/ptsip/engines/index.json",
		"src/ptsip/engines/storage/manifest.json", "src/ptsip/engines/storage/execution-bindings.json",
		"src/ptsip/engines/storage/version.yaml", "src/ptsip/engines/storage/Cargo.toml", "src/ptsip/engines/storage/Cargo.lock"}
	if !reflect.DeepEqual(result.ChangedFiles, expected) {
		t.Fatalf("wrong projection targets: got %v, want %v", result.ChangedFiles, expected)
	}
	project, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	for _, needle := range []string{`tool_version = "T.0.3.9"`, `profile_version = "pp.2.01"`, `storage = "0.3.3"`, `context = "0.2.2"`} {
		if !strings.Contains(string(project), needle) {
			t.Fatalf("missing %s", needle)
		}
	}
	if result, err = SyncPackageVersion(root, true); err != nil || result.Changed {
		t.Fatalf("complete composition check failed: %+v %v", result, err)
	}
	registeredAfter, err := os.ReadFile(registryPath)
	if err != nil || string(registeredAfter) != registered {
		t.Fatalf("version sync changed the Profile registry: %v", err)
	}
}

func TestVersionSyncRejectsInvalidOrMissingSource(t *testing.T) {
	cases := []string{
		"0.3.8-b7\n",
		strings.Replace(canonicalVersionFixture, "0.3.8-b7", "0.3.8b7", 1),
		strings.Replace(canonicalVersionFixture, `version = "T.0.3.8.b2"`, `version = "0.3.8b2"`, 1),
		strings.Replace(canonicalVersionFixture, `"ptsip.project-profile" = "pp.2.00"`, "", 1),
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
	if _, err := SyncPackageVersion(root, false); err == nil {
		t.Fatal("missing canonical source accepted")
	}
}

func TestVersionSyncRejectsAmbiguousProjectFields(t *testing.T) {
	cases := map[string]string{
		"missing_project":   strings.Replace(versionSyncFixture, "[project]", "[other]", 1),
		"missing_version":   strings.Replace(versionSyncFixture, `version = "0.3.8b6"`, `v = "0.3.8b6"`, 1),
		"duplicate_version": strings.Replace(versionSyncFixture, `version = "0.3.8b6"`, `version = "0.3.8b6"`+"\n"+`version = "0.3.8b5"`, 1),
		"missing_state":     strings.Replace(versionSyncFixture, `package_state = "b6"`, "", 1),
		"missing_engine":    strings.Replace(versionSyncFixture, `storage = "0.3.2"`, "", 1),
	}
	for name, project := range cases {
		t.Run(name, func(t *testing.T) {
			root := writeVersionFixture(t, canonicalVersionFixture, project)
			if _, err := SyncPackageVersion(root, false); err == nil {
				t.Fatal("ambiguous projection accepted")
			}
		})
	}
}

func TestVersionSyncPreservesCRLFAndSupportsStableRelease(t *testing.T) {
	source := strings.Replace(canonicalVersionFixture, "0.3.8-b7", "0.3.9", 1)
	project := strings.ReplaceAll(versionSyncFixture, "\n", "\r\n")
	root := writeVersionFixture(t, source, project)
	if _, err := SyncPackageVersion(root, false); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if strings.Contains(strings.ReplaceAll(string(after), "\r\n", ""), "\n") {
		t.Fatal("CRLF rewritten")
	}
	for _, want := range []string{`version = "0.3.9"`, `package_base = "0.3.9"`, `package_state = ""`} {
		if !strings.Contains(string(after), want) {
			t.Fatalf("missing stable projection %s", want)
		}
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
	t.Run("yaml_crlf", TestVersionSyncYAMLCRLF)
	t.Run("ambiguous_or_different_active_profile", TestVersionSyncRejectsUnregisteredActiveProfile)
}

func TestVersionSyncYAMLCRLF(t *testing.T) {
	root := writeVersionFixture(t, canonicalVersionFixture, versionSyncFixture)
	var paths []string
	for _, id := range officialEngineIDs {
		paths = append(paths, "src/ptsip/engines/"+id+"/version.yaml")
	}
	before := map[string]string{}
	for _, ref := range paths {
		path := filepath.Join(root, filepath.FromSlash(ref))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		withCRLF := strings.ReplaceAll(string(data), "\n", "\r\n")
		if ref == "src/ptsip/engines/storage/version.yaml" {
			withCRLF = strings.Replace(withCRLF, "version: 0.3.2\r\n", "version: 0.3.2 # storage version\r\n", 1)
		}
		if err := os.WriteFile(path, []byte(withCRLF), 0644); err != nil {
			t.Fatal(err)
		}
		before[ref] = withCRLF
	}
	result, err := SyncPackageVersion(root, false)
	if err != nil || !reflect.DeepEqual(result.ChangedFiles, []string{"pyproject.toml"}) {
		t.Fatalf("CRLF fields must not cause false mismatch: %+v %v", result, err)
	}
	for _, ref := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref)))
		if err != nil || string(data) != before[ref] {
			t.Fatalf("unchanged CRLF projection was rewritten: %s: %v", ref, err)
		}
	}
	if result, err = SyncPackageVersion(root, true); err != nil || result.Changed || result.Status != "IN_SYNC" {
		t.Fatalf("CRLF check-version must succeed: %+v %v", result, err)
	}
	storageYAML := "src/ptsip/engines/storage/version.yaml"
	path := filepath.Join(root, filepath.FromSlash(storageYAML))
	stale := strings.Replace(before[storageYAML], "version: 0.3.2", "version: 0.3.1", 1)
	if err := os.WriteFile(path, []byte(stale), 0644); err != nil {
		t.Fatal(err)
	}
	result, err = SyncPackageVersion(root, false)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{storageYAML}
	if !reflect.DeepEqual(result.ChangedFiles, expected) {
		t.Fatalf("CRLF projection changed wrong paths: got %v, want %v", result.ChangedFiles, expected)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != before[storageYAML] {
		t.Fatalf("CRLF, comment or updated version was lost: %s: %v", storageYAML, err)
	}
	if result, err = SyncPackageVersion(root, true); err != nil || result.Changed {
		t.Fatalf("CRLF updates must be idempotent: %+v %v", result, err)
	}
}

func TestVersionSyncRejectsUnregisteredActiveProfile(t *testing.T) {
	source := strings.Replace(canonicalVersionFixture, `"ptsip.project-profile" = "pp.2.00"`, `"ptsip.project-profile" = "pp.2.01"`, 1)
	root := writeVersionFixture(t, source, versionSyncFixture)
	before, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SyncPackageVersion(root, false); err == nil || !strings.Contains(err.Error(), "Profile current projection differs") {
		t.Fatalf("unregistered active Profile must fail closed: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil || string(before) != string(after) {
		t.Fatalf("failed projection must not write other files: %v", err)
	}
}
