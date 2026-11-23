package tool_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	releaseversion "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/version"
)

const (
	versionsContract = ".ptsip/contracts/release/Platform/Common/versions_different.json"
	versionsPreview  = ".ptsip/engine/developer/Distribution/release/version/versions_different.ini"
	versionsSource   = "src/VERSION"
)

const currentVersionFixture = `[package]
version = "0.3.8-b7"
[tool]
version = "T.0.3.8.b2"
[RBAC]
RA_list = "v.1"
RAC_list = "v.1"
AEC_list = "v.1"
[common_contract]
registry_revision = "v2"
[engines]
context = "0.2.2"
discovery = "0.3.3"
execution = "0.3.2"
projection = "0.2.2"
storage = "0.3.2"
validation = "0.3.4"
[schemas]
"ptsip.policy.v4" = "v4"
"ptsip.project-profile" = "pp.3.00"
`

const oldVersionFixture = `package_version = "0.3.8-b7"
tool_version = "T.0.3.8.b2"
profile_version = "pp.2.00"
[engines]
context = "0.2.2"
discovery = "0.3.3"
execution = "0.3.2"
projection = "0.2.2"
storage = "0.3.2"
validation = "0.3.4"
planning = "0.3.2"
`

func gitForVersionTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, b)
	}
	return strings.TrimSpace(string(b))
}
func fixtureWrite(t *testing.T, root, path, content string) {
	t.Helper()
	dest := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func versionTestRepo(t *testing.T) (string, string, string) {
	t.Helper()
	return versionTestRepoWithVersions(t, currentVersionFixture, currentVersionFixture)
}
func versionTestRepoWithVersions(t *testing.T, previous, current string) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	gitForVersionTest(t, root, "init", "-q")
	gitForVersionTest(t, root, "config", "user.email", "test@example.com")
	gitForVersionTest(t, root, "config", "user.name", "Test")
	sourceContract, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "..", "..", filepath.FromSlash(versionsContract)))
	if err != nil {
		t.Fatal(err)
	}
	fixtureWrite(t, root, versionsContract, string(sourceContract))
	fixtureWrite(t, root, versionsSource, previous)
	fixtureWrite(t, root, "src/ptsip/engines/validation/existing.rs", "pub fn old() {}\n")
	gitForVersionTest(t, root, "add", ".")
	gitForVersionTest(t, root, "commit", "-qm", "baseline")
	base := gitForVersionTest(t, root, "rev-parse", "HEAD")
	fixtureWrite(t, root, versionsSource, current)
	fixtureWrite(t, root, "src/ptsip/engines/validation/new.rs", "pub fn added_feature() {}\n")
	gitForVersionTest(t, root, "add", ".")
	gitForVersionTest(t, root, "commit", "-qm", "add declaration")
	head := gitForVersionTest(t, root, "rev-parse", "HEAD")
	return root, base, head
}
func TestVersionsDifferentPreviewDestination(t *testing.T) {
	if versionsPreview != ".ptsip/engine/developer/Distribution/release/version/versions_different.ini" {
		t.Fatalf("unexpected version preview destination: %s", versionsPreview)
	}
	root, base, head := versionTestRepo(t)
	if _, err := releaseversion.PreviewVersionsDifferent(root, base, head); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(versionsPreview))); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".ptsip", "engines", "release", "versions_different.ini")); !os.IsNotExist(err) {
		t.Fatalf("obsolete preview destination created: %v", err)
	}
}

func TestVersionsDifferentPreviewAndTamper(t *testing.T) {
	root, base, head := versionTestRepo(t)
	report, err := releaseversion.PreviewVersionsDifferent(root, base, head)
	if err != nil {
		t.Fatal(err)
	}
	if report.Legacy || len(report.Entries) != 1 || report.Entries[0].Component != "engine.validation" ||
		!report.Entries[0].DeclarationCandidate || report.Entries[0].Current != "0.3.4" {
		t.Fatalf("unexpected diff report: %#v", report)
	}
	if _, err = releaseversion.CheckVersionsDifferent(root, base, head); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(versionsPreview))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString("proposed_version=0.3.5\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err = releaseversion.CheckVersionsDifferent(root, base, head); err == nil {
		t.Fatal("modified preview was accepted")
	}
}
func TestVersionsDifferentPolicyOnlyNoBump(t *testing.T) {
	root, _, head := versionTestRepo(t)
	fixtureWrite(t, root, "developer/policy/unrelated.yaml", "approval: example\n")
	gitForVersionTest(t, root, "add", ".")
	gitForVersionTest(t, root, "commit", "-qm", "policy only")
	policyHead := gitForVersionTest(t, root, "rev-parse", "HEAD")
	report, err := releaseversion.AnalyzeVersionsDifferent(root, head, policyHead)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Entries) != 0 {
		t.Fatalf("policy-only commit changed versions: %#v", report)
	}
	if report.Entries == nil {
		t.Fatal("empty change set must use non-nil slice")
	}
	serialized, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serialized), `"Entries":[]`) {
		t.Fatalf("empty change list is not serialized as []: %s", serialized)
	}
}
func TestVersionsDifferentApplyRequiresExplicitSelection(t *testing.T) {
	root, base, head := versionTestRepo(t)
	report, err := releaseversion.PreviewVersionsDifferent(root, base, head)
	if err != nil {
		t.Fatal(err)
	}
	if err = releaseversion.ApplySelectedVersion(root, base, head, "engine.validation", "0.3.5", report.VersionDigest, false); err == nil {
		t.Fatal("unapproved version update accepted")
	}
	if err = releaseversion.ApplySelectedVersion(root, base, head, "engine.validation", "0.3.5", strings.Repeat("0", 64), true); err == nil {
		t.Fatal("wrong source digest accepted")
	}
	if err = releaseversion.ApplySelectedVersion(root, base, head, "engine.validation", "000.4.0", report.VersionDigest, true); err == nil {
		t.Fatal("invalid version accepted")
	}
	if err = releaseversion.ApplySelectedVersion(root, base, head, "engine.validation", "0.3.5", report.VersionDigest, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "src", "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "validation = \"0.3.5\"") {
		t.Fatalf("not updated: %s", data)
	}
	if _, err = releaseversion.CheckVersionsDifferent(root, base, head); err == nil {
		t.Fatal("stale preview remains valid after editing VERSION")
	}
}
func TestVersionsDifferentReadsHistoricalDeclarationsOnlyAtAnExactEarlierCommit(t *testing.T) {
	root, base, head := versionTestRepoWithVersions(t, oldVersionFixture, currentVersionFixture)
	versions, err := releaseversion.VersionsAtCommit(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if versions["profile"] != "pp.2.00" || versions["engine.planning"] != "0.3.2" {
		t.Fatalf("historical versions not preserved: %v", versions)
	}
	_, changes, err := releaseversion.DeclaredChanges(root, base, head)
	if err != nil || len(changes) != 1 || changes[0].Component != "profile" || changes[0].Version != "pp.3.00" {
		t.Fatalf("expected only the historical Profile transition: %v, %v", changes, err)
	}

	for _, test := range []struct{ name, source string }{
		{"legacy", oldVersionFixture},
		{"missing common contract", strings.Replace(currentVersionFixture, "[common_contract]\nregistry_revision = \"v2\"\n", "", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, base, head := versionTestRepoWithVersions(t, currentVersionFixture, test.source)
			if _, err := releaseversion.PreviewVersionsDifferent(root, base, head); err == nil {
				t.Fatal("preview accepted malformed current source")
			}
			if _, err := releaseversion.VersionsAtCommit(root, head); err == nil {
				t.Fatal("read accepted malformed current source")
			}
		})
	}
}

func TestVersionsDifferentEditsOnlyTheRegisteredProfileVersion(t *testing.T) {
	fixture := strings.Replace(currentVersionFixture, `"ptsip.project-profile" = "pp.3.00"`, `  'ptsip.project-profile' = 'pp.3.00' # current Profile`, 1)
	root, _, previous := versionTestRepoWithVersions(t, fixture, fixture)
	gitForVersionTest(t, root, "config", "core.autocrlf", "true")
	fixture = strings.ReplaceAll(fixture, "\n", "\r\n")
	fixtureWrite(t, root, versionsSource, fixture)
	const profilePath = "src/ptsip/schemas/storage/profile/project-profile.schema.json"
	fixtureWrite(t, root, profilePath, "{}\n")
	gitForVersionTest(t, root, "add", ".")
	gitForVersionTest(t, root, "commit", "-qm", "change current Profile schema")
	head := gitForVersionTest(t, root, "rev-parse", "HEAD")
	report, err := releaseversion.PreviewVersionsDifferent(root, previous, head)
	if err != nil || len(report.Entries) != 1 || report.Entries[0].Component != "profile" || report.Entries[0].Current != "pp.3.00" {
		t.Fatalf("Profile source boundary: %#v, %v", report, err)
	}
	if err := releaseversion.ApplySelectedVersion(root, previous, head, "profile", "pp.3.01", report.VersionDigest, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(versionsSource)))
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(fixture, "'pp.3.00'", `"pp.3.01"`, 1); string(data) != want {
		t.Fatalf("updated source did not preserve unrelated bytes:\n%s", data)
	}
}

func TestVersionsDifferentRejectsFakeHeadAndNoSourceChange(t *testing.T) {
	root, base, head := versionTestRepo(t)
	if _, err := releaseversion.AnalyzeVersionsDifferent(root, head, base); err == nil {
		t.Fatal("non-ancestor accepted")
	}
	if _, err := releaseversion.AnalyzeVersionsDifferent(root, base, strings.Repeat("a", 40)); err == nil {
		t.Fatal("unverified HEAD accepted")
	}
	if _, err := releaseversion.AnalyzeVersionsDifferent(root, head, head); err == nil {
		t.Fatal("identical commits accepted")
	}
}

func TestVersionsDifferentDeveloperMainEntrypoint(t *testing.T) {
	root, base, head := versionTestRepo(t)
	developerRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"-C", developerRoot, "run", ".", "--validator", "release-version-diff",
		"--operation", "preview", "--root", root, "--base", base, "--head", head}
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("developer release entrypoint failed: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "\"publication_authorized\": false") {
		t.Fatalf("release preview must not grant publication: %s", out)
	}
	if _, err = os.Stat(filepath.Join(root, filepath.FromSlash(versionsPreview))); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(developerRoot, "release", "cmd", "versions-different", "main.go")); !os.IsNotExist(err) {
		t.Fatalf("standalone version command still exists: %v", err)
	}
}
