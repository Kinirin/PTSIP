package profileverify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
)

const profileRoutePath = ".ptsip/contracts/release/Platform/Common/profile/index.json"

func profileFixture(t *testing.T) (string, string, release.ProfileRegistration) {
	t.Helper()
	source, target := repoRoot(t), t.TempDir()
	copyFile := func(path string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		to := filepath.Join(target, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	copyFile(profileRoutePath)
	copyFile(profileRegistryPath)
	registrations, current, err := release.ReadProfileRegistrations(target)
	if err != nil {
		t.Fatal(err)
	}
	var active release.ProfileRegistration
	for _, row := range registrations {
		copyFile(profileStorageRoot + row.Path)
		if row.Version == current {
			active = row
		}
	}
	return target, current, active
}

func rewriteProfileFixture(t *testing.T, root, path string, edit func(string) string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(path))
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	before, after := string(raw), edit(string(raw))
	if before == after {
		t.Fatalf("fixture mutation has no effect: %s", path)
	}
	if err := os.WriteFile(p, []byte(after), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNativeProfileVerifierAdmitsCurrentRepository(t *testing.T) {
	root := repoRoot(t)
	composition, err := release.ReadComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyProjectProfiles(root, composition.Profile); err != nil {
		t.Fatal(err)
	}
}

func TestNativeProfileVerifierNeedsNoHistoricalReleaseAssets(t *testing.T) {
	root, active, _ := profileFixture(t)
	if err := VerifyProjectProfiles(root, active); err != nil {
		t.Fatalf("native verifier required unregistered Profile assets: %v", err)
	}
	for _, name := range []string{"registry", "schemas", "developer", profileContractPath} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !os.IsNotExist(err) {
			t.Fatalf("fixture has a retired source %s", name)
		}
	}
}

func TestNativeProfileVerifierRejectsReleaseVersionMismatch(t *testing.T) {
	root, _, _ := profileFixture(t)
	if err := VerifyProjectProfiles(root, "pp.9.99"); err == nil || !strings.Contains(err.Error(), "PROFILE_RELEASE_VERSION_MISMATCH") {
		t.Fatalf("wrong selected release version: %v", err)
	}
}

func TestNativeProfileVerifierRejectsRegisteredSchemaDigestDrift(t *testing.T) {
	root, active, registration := profileFixture(t)
	rewriteProfileFixture(t, root, profileStorageRoot+registration.Path, func(s string) string {
		return s + "\n"
	})
	if err := VerifyProjectProfiles(root, active); err == nil || !strings.Contains(err.Error(), "PROFILE_SCHEMA_DIGEST_MISMATCH") {
		t.Fatalf("altered registered schema bytes: %v", err)
	}
}

func TestNativeProfileVerifierRejectsMalformedPinnedSchema(t *testing.T) {
	root, active, registration := profileFixture(t)
	var digest string
	rewriteProfileFixture(t, root, profileStorageRoot+registration.Path, func(s string) string {
		changed := strings.Replace(s, `"type": "object"`, `"type": "invalid-type"`, 1)
		digest = release.Digest([]byte(strings.ReplaceAll(changed, "\r\n", "\n")))
		return changed
	})
	rewriteProfileFixture(t, root, profileRegistryPath, func(s string) string {
		return strings.Replace(s, registration.Digest, digest, 1)
	})
	if err := VerifyProjectProfiles(root, active); err == nil || !strings.Contains(err.Error(), "PROFILE_SCHEMA_INVALID") {
		t.Fatalf("invalid schema with a matching digest: %v", err)
	}
}

func TestNativeProfileExecutionNeverReentersRetiredDeveloperVerifier(t *testing.T) {
	root := repoRoot(t)
	runner, err := os.ReadFile(filepath.Join(root, ".ptsip/engine/developer/Distribution/release/test/runner.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(runner), "VerifyProjectProfiles(root, c.Version)") ||
		strings.Contains(string(runner), `"developer/tests", "test", "./pp"`) {
		t.Fatal("Release Profile verification is not owned by the native release engine")
	}
	modes, err := os.ReadFile(filepath.Join(root, ".github/test_modes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(modes), "developer/tests/pp") {
		t.Fatal("retired developer Profile tests are still registered as active architecture verification")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestProfileVerifierUsesRegisteredSchemaStorage(t *testing.T) {
	root, active, _ := profileFixture(t)
	registrations, _, err := release.ReadProfileRegistrations(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, registration := range registrations {
		if registration.Version != active {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(profileStorageRoot+registration.Path))); err != nil {
				t.Fatal(err)
			}
			if err := VerifyProjectProfiles(root, active); err == nil || !strings.Contains(err.Error(), "PROFILE_SCHEMA_MISSING") {
				t.Fatalf("missing registered historical schema: %v", err)
			}
			return
		}
	}
	t.Fatal("fixture requires historical schema registrations")
}

func TestProfileVerifierRejectsAuthorityRouteDrift(t *testing.T) {
	for _, field := range []string{"publication_authorized", "consumer_contract_authority", "legacy_fallback"} {
		t.Run(field, func(t *testing.T) {
			root, active, _ := profileFixture(t)
			rewriteProfileFixture(t, root, profileRoutePath, func(s string) string {
				return strings.Replace(s, `"`+field+`": false`, `"`+field+`": true`, 1)
			})
			if err := VerifyProjectProfiles(root, active); err == nil || !strings.Contains(err.Error(), "PROFILE_VERIFICATION_ROUTING_INVALID") {
				t.Fatalf("forged route: %v", err)
			}
		})
	}
}
