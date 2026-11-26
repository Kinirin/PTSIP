package tool_test

import (
	"encoding/json"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
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
version = "ver.1.0"
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
"ptsip.policy.v4" = "SCH.1.00"
"urn:ptsip:contract:plan:public-structure" = "SCH.1.00"
RA_list = "SCH.1.00"
RAC_list = "SCH.1.00"
AEC_list = "SCH.1.00"
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
	sourceContract, err := os.ReadFile(filepath.Join(testrepo.Root(t), filepath.FromSlash(versionsContract)))
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
	previousSectioned := strings.Replace(currentVersionFixture, `version = "ver.1.0"`, `RA_list = "v.1"
RAC_list = "v.1"
AEC_list = "v.1"`, 1)
	previousSectioned = strings.Replace(previousSectioned, `"ptsip.policy.v4" = "SCH.1.00"
"urn:ptsip:contract:plan:public-structure" = "SCH.1.00"
RA_list = "SCH.1.00"
RAC_list = "SCH.1.00"
AEC_list = "SCH.1.00"`, `"ptsip.policy.v4" = "v4"`, 1)
	root, base, _ = versionTestRepoWithVersions(t, previousSectioned, currentVersionFixture)
	versions, err = releaseversion.VersionsAtCommit(root, base)
	if err != nil || versions["engine.validation"] != "0.3.4" || versions["profile"] != "pp.3.00" {
		t.Fatalf("former six-table Git revision not preserved: %v, %v", versions, err)
	}

	for _, test := range []struct{ name, source string }{
		{"legacy", oldVersionFixture},
		{"former six tables", previousSectioned},
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
	developerRoot := filepath.Join(testrepo.Root(t), ".ptsip/engine/developer")
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

func TestEngineVersionSuggestionsUseOnePublicationInterval(t *testing.T) {
	const base = "2.4.6"
	for _, tc := range []struct {
		name, current, want string
		triggers            []releaseversion.EngineVersionTrigger
	}{
		{"standalone responsibility deletion is patch", base, "2.4.7", []releaseversion.EngineVersionTrigger{releaseversion.StandaloneResponsibilityDeletion}},
		{"repeated patch", "2.4.7", "2.4.7", []releaseversion.EngineVersionTrigger{releaseversion.ContractOnlyChange}},
		{"patch promotes to minor from publication", "2.4.7", "2.5.0", []releaseversion.EngineVersionTrigger{releaseversion.NewEngineFunctionAddition}},
		{"transfer out is minor", base, "2.5.0", []releaseversion.EngineVersionTrigger{releaseversion.ResponsibilityTransferOut}},
		{"transfer in is minor", base, "2.5.0", []releaseversion.EngineVersionTrigger{releaseversion.ResponsibilityTransferIn}},
		{"minor promotes to major from publication", "2.5.0", "3.0.0", []releaseversion.EngineVersionTrigger{releaseversion.ResponsibilityCompatibilityBreak}},
		{"major remains fixed", "3.0.0", "3.0.0", []releaseversion.EngineVersionTrigger{releaseversion.ResponsibilityCompatibilityBreak, releaseversion.ExistingEngineFunctionChange}},
		{"combination uses highest class", base, "3.0.0", []releaseversion.EngineVersionTrigger{releaseversion.FunctionCompatibilityBreak, releaseversion.ResponsibilityCompatibilityBreak, releaseversion.ContractOnlyChange}},
		{"declaration without contract is unchanged", base, base, []releaseversion.EngineVersionTrigger{releaseversion.NewDomainWithoutRelatedContract, releaseversion.ResponsibilityScopeChangeWithoutCode}},
		{"entire engine removal freezes final version", "2.5.0", "2.5.0", []releaseversion.EngineVersionTrigger{releaseversion.FullEngineRemoval, releaseversion.ResponsibilityCompatibilityBreak}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := releaseversion.SuggestEngineVersion(base, tc.current, tc.triggers...)
			if err != nil || got != tc.want {
				t.Fatalf("got %s, %v; want %s", got, err, tc.want)
			}
		})
	}
	if _, err := releaseversion.SuggestEngineVersion("", base, releaseversion.ContractOnlyChange); err == nil {
		t.Fatal("suggested a version without a publication baseline")
	}
	if _, err := releaseversion.SuggestEngineVersion(base, "2.4.8", releaseversion.ContractOnlyChange); err == nil {
		t.Fatal("accepted a version outside the publication interval")
	}
	if _, err := releaseversion.SuggestEngineVersion(base, base, "undocumented reason"); err == nil {
		t.Fatal("accepted an unknown trigger")
	}
	if got, err := releaseversion.SuggestEngineVersion("3.0.0", "3.0.0", releaseversion.ContractOnlyChange); err != nil || got != "3.0.1" {
		t.Fatalf("confirmed publication did not reset the baseline: got %s, %v", got, err)
	}
}

func TestSchemaPublicationClassAndIndependentRBACClassification(t *testing.T) {
	if got, err := releaseversion.PublicationIntervalClass(releaseversion.VersionPatch, releaseversion.VersionMinor, releaseversion.VersionPatch); err != nil || got != releaseversion.VersionMinor {
		t.Fatalf("publication interval did not preserve the highest established class: %v, %v", got, err)
	}
	if got, err := releaseversion.PublicationIntervalClass(releaseversion.VersionMajor, releaseversion.VersionPatch, releaseversion.VersionMinor); err != nil || got != releaseversion.VersionMajor {
		t.Fatalf("publication interval did not preserve the fixed major class: %v, %v", got, err)
	}
	if got, err := releaseversion.ClassifyRBACSystemVersion(releaseversion.RBACListFieldAddition, releaseversion.RBACOrdinaryChange); err != nil || got != releaseversion.VersionMinor {
		t.Fatalf("ordinary RBAC changes did not yield independent minor classification: %v, %v", got, err)
	}
	if got, err := releaseversion.ClassifyRBACSystemVersion(releaseversion.RBACOrdinaryChange, releaseversion.RBACInferenceFunctionChange); err == nil || got != releaseversion.VersionUnchanged {
		t.Fatalf("RBAC major bypassed direct user decision: %v, %v", got, err)
	}
}
