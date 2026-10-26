package releasecontroller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const versionSyncFixture = `[project]
name = "PTSIP"
version = "0.3.8b6"

[tool.ptsip.release]
schema_version = "ptsip-current-composition/v2"
package_base = "0.3.8"
package_revision = 0
package_state = "b6"
tool_version = "T.0.3.8.b2"

[tool.ptsip.release.engines]
context = "0.2.2"
validation = "0.3.4"

[tool.ptsip.release.published]
identity = "0.3.8.fa660a"
`

func writeVersionFixture(t *testing.T, source, project string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "VERSION"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(project), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSyncPackageVersionFromSource(t *testing.T) {
	root := writeVersionFixture(t, "0.3.8-b7\n", versionSyncFixture)
	original, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SyncPackageVersion(root, true); err == nil || !strings.Contains(err.Error(), "run sync-version") {
		t.Fatalf("check-mode should reject stale Package version: %v", err)
	}
	unchanged, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil || string(unchanged) != string(original) {
		t.Fatalf("check-mode must not mutate pyproject: %v", err)
	}
	result, err := SyncPackageVersion(root, false)
	if err != nil || result.Status != "SYNCED" || !result.Changed || result.Previous != "0.3.8b6" || result.Version != "0.3.8-b7" {
		t.Fatalf("invalid sync result: %+v, %v", result, err)
	}
	updated, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`version = "0.3.8-b7"`, `package_base = "0.3.8"`, `package_state = "b7"`, `package_revision = 0`, `tool_version = "T.0.3.8.b2"`, `context = "0.2.2"`, `[tool.ptsip.release.published]`, `identity = "0.3.8.fa660a"`} {
		if !strings.Contains(string(updated), want) {
			t.Fatalf("missing projection or preserved field: %s", want)
		}
	}
	result, err = SyncPackageVersion(root, true)
	if err != nil || result.Status != "IN_SYNC" || result.Changed {
		t.Fatalf("check after sync failed: %+v, %v", result, err)
	}
	result, err = SyncPackageVersion(root, false)
	if err != nil || result.Changed {
		t.Fatalf("second sync must be idempotent: %+v, %v", result, err)
	}
}

func TestVersionSyncRejectsInvalidOrMissingSource(t *testing.T) {
	for _, source := range []string{"0.3.8b7\n", "0.3.8-b7\nextra\n", "0.3.8--b7\n", "0.3.8-b0\n", "0.3.8-b7 \n", "0.3.8-b7\n\n"} {
		t.Run(strings.ReplaceAll(source, "\n", "_"), func(t *testing.T) {
			root := writeVersionFixture(t, source, versionSyncFixture)
			if _, err := SyncPackageVersion(root, false); err == nil {
				t.Fatalf("invalid source accepted: %q", source)
			}
		})
	}
	root := writeVersionFixture(t, "0.3.8-b7\n", versionSyncFixture)
	if err := os.Remove(filepath.Join(root, "src", "VERSION")); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncPackageVersion(root, false); err == nil {
		t.Fatal("missing canonical source must fail closed")
	}
}

func TestVersionSyncRejectsAmbiguousProjectFields(t *testing.T) {
	for name, project := range map[string]string{
		"missing_project": strings.Replace(versionSyncFixture, "[project]", "[other]", 1),
		"missing_version": strings.Replace(versionSyncFixture, `version = "0.3.8b6"`, "v = \"0.3.8b6\"", 1),
		"duplicate_version": strings.Replace(versionSyncFixture, `version = "0.3.8b6"`, `version = "0.3.8b6"`+"\n"+`version = "0.3.8b5"`, 1),
		"duplicate_table": versionSyncFixture + "\n[project]\n",
		"missing_state": strings.Replace(versionSyncFixture, `package_state = "b6"`, "", 1),
	} {
		t.Run(name, func(t *testing.T) {
			root := writeVersionFixture(t, "0.3.8-b7\n", project)
			if _, err := SyncPackageVersion(root, false); err == nil {
				t.Fatal("invalid or ambiguous projection accepted")
			}
		})
	}
}

func TestVersionSyncPreservesCRLFAndSupportsStableRelease(t *testing.T) {
	project := strings.ReplaceAll(versionSyncFixture, "\n", "\r\n")
	root := writeVersionFixture(t, "0.3.9\n", project)
	if _, err := SyncPackageVersion(root, false); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ReplaceAll(string(after), "\r\n", ""), "\n") {
		t.Fatal("projection rewrote unrelated CRLF lines")
	}
	for _, want := range []string{`version = "0.3.9"`, `package_base = "0.3.9"`, `package_state = ""`} {
		if !strings.Contains(string(after), want) {
			t.Fatalf("stable version projection missing: %s", want)
		}
	}
}
