package releasetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
)

func profileFixture(t *testing.T) string {
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
	for _, path := range []string{
		profileContractPath,
		profileProjectionPath,
		profileSourcePath + "/index.yaml",
		"schemas/ptsip-profile-pp-1.01.schema.json",
		"schemas/ptsip-profile-pp-1.02.schema.json",
	} {
		copyFile(path)
	}
	base := filepath.Join(source, filepath.FromSlash(profileSourcePath))
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".ptsip.yaml") {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		copyFile(filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The retired developer policy/schema and its CLI are intentionally absent.
	if _, err := os.Stat(filepath.Join(target, "developer", "policy", "schemas", "root-family-policy.schema.json")); !os.IsNotExist(err) {
		t.Fatal("native Profile fixture must not contain retired schema")
	}
	return target
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

func TestNativeProfileVerifierNeedsNoRetiredDeveloperPolicy(t *testing.T) {
	root := profileFixture(t)
	if err := VerifyProjectProfiles(root, "pp.1.02"); err != nil {
		t.Fatalf("native verifier depended on retired developer-policy tooling: %v", err)
	}
}

func TestNativeProfileVerifierRejectsRegistryProjectionDrift(t *testing.T) {
	root := profileFixture(t)
	rewriteProfileFixture(t, root, profileProjectionPath, func(s string) string {
		return strings.Replace(s, "current: pp.1.02", "current: pp.9.99", 1)
	})
	if err := VerifyProjectProfiles(root, "pp.1.02"); err == nil {
		t.Fatal("embedded registry drift admitted")
	}
}

func TestNativeProfileVerifierRejectsReleaseVersionMismatch(t *testing.T) {
	if err := VerifyProjectProfiles(profileFixture(t), "pp.9.99"); err == nil {
		t.Fatal("wrong selected release version admitted")
	}
}

func TestNativeProfileVerifierRejectsCatalogModeDrift(t *testing.T) {
	root := profileFixture(t)
	rewriteProfileFixture(t, root, profileSourcePath+"/index.yaml", func(s string) string {
		return strings.Replace(s, "responsibility_mode: explicit", "responsibility_mode: hybrid", 1)
	})
	if err := VerifyProjectProfiles(root, "pp.1.02"); err == nil {
		t.Fatal("catalog vs profile responsibility mode drift admitted")
	}
}

func TestNativeProfileVerifierRejectsImmutableBaselineChange(t *testing.T) {
	root := profileFixture(t)
	rewriteProfileFixture(t, root, profileSourcePath+"/example.ptsip.yaml", func(s string) string {
		return strings.Replace(s, "revision: \"Rev.0001\"", "revision: \"Rev.0002\"", 1)
	})
	if err := VerifyProjectProfiles(root, "pp.1.02"); err == nil {
		t.Fatal("current Profile differed from immutable baseline")
	}
}

func TestNativeProfileVerifierRejectsUncatalogedResource(t *testing.T) {
	root := profileFixture(t)
	p := filepath.Join(root, filepath.FromSlash(profileSourcePath), "rogue.ptsip.yaml")
	if err := os.WriteFile(p, []byte("ptsip: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyProjectProfiles(root, "pp.1.02"); err == nil {
		t.Fatal("undeclared Profile distribution resource admitted")
	}
}

func TestNativeProfileVerifierRejectsMalformedProfileEvenWithMatchingBaseline(t *testing.T) {
	root := profileFixture(t)
	rewrite := func(s string) string {
		return strings.Replace(s, "revision: \"Rev.0001\"", "revision: \"invalid\"", 1)
	}
	rewriteProfileFixture(t, root, profileSourcePath+"/example.ptsip.yaml", rewrite)
	rewriteProfileFixture(t, root, profileSourcePath+"/history/pp.1.02/example.ptsip.yaml", rewrite)
	if err := VerifyProjectProfiles(root, "pp.1.02"); err == nil {
		t.Fatal("invalid Profile passed because it matched baseline")
	}
}

func TestNativeProfileExecutionNeverReentersRetiredDeveloperVerifier(t *testing.T) {
	root := repoRoot(t)
	runner, err := os.ReadFile(filepath.Join(root, ".ptsip/engine/developer/release/test/runner.go"))
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
