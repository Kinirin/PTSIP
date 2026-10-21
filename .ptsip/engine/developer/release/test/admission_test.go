package releasetest

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	p, e := filepath.Abs("../../../..")
	if e != nil {
		t.Fatal(e)
	}
	return p
}

func TestRegisteredSelectionRejectsAliasesAndTamper(t *testing.T) {
	root := repoRoot(t)
	for _, target := range []string{"profile,profiles", "context,engine:context", "unknown"} {
		if _, err := release.Select(root, target, ""); err == nil {
			t.Fatalf("ambiguous selection accepted: %s", target)
		}
	}
	if _, err := release.Select(root, "context", "macos-arm64,macos-arm64"); err == nil {
		t.Fatal("duplicate host accepted")
	}
	selected, err := release.Select(root, "context,projection", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := release.ValidateSelection(root, selected); err != nil {
		t.Fatal(err)
	}
	selected.Components[0].Version = "99.0.0"
	if err := release.ValidateSelection(root, selected); err == nil {
		t.Fatal("modified registered version admitted")
	}
}

func TestCanonicalMachineOutputExcludesSuccessfulProcessDiagnostics(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "main.go")
	fixture := "package main\nimport (\"fmt\"; \"os\")\nfunc main(){fmt.Fprintln(os.Stderr,\"dependency download diagnostic\");fmt.Print(`{\"record\":{\"status\":\"PASS\"}}`)}\n"
	if err := os.WriteFile(source, []byte(fixture), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := release.Run(root, "go", "run", source)
	if err != nil || string(data) != `{"record":{"status":"PASS"}}` {
		t.Fatalf("canonical JSON contaminated by stderr: %s, %v", data, err)
	}
}

func archive(t *testing.T, names []string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, name := range names {
		f, e := w.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.Write([]byte("catalog-resource")); e != nil {
			t.Fatal(e)
		}
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}

func TestProfileDistributionRejectsHistoryCodeAndTraversal(t *testing.T) {
	base := []string{"src/ptsip/profiles/index.yaml", "src/ptsip/profiles/example.ptsip.yaml", "registry/project-profile-contracts.yaml"}
	if _, e := release.ProfileResources(archive(t, base)); e != nil {
		t.Fatal(e)
	}
	for _, ref := range []string{"src/ptsip/profiles/history/old.ptsip.yaml", "src/ptsip/profiles/catalog.py", "src/ptsip/profiles/../escape.ptsip.yaml"} {
		if _, e := release.ProfileResources(archive(t, append(append([]string{}, base...), ref))); e == nil {
			t.Fatalf("forbidden Profile resource admitted: %s", ref)
		}
	}
}

func TestToolDistinguishesProductTargetNamespaceFromRustBuildOutput(t *testing.T) {
	if forbiddenToolMember("ptsip/migration/analysis/target/semantics.py") {
		t.Fatal("existing product API namespace rejected as Cargo output")
	}
	for _, name := range []string{"ptsip/engines/context/target/release/binary", "ptsip/profiles/history/pp.1.01/example.ptsip.yaml", "ptsip/engines/context/binary.exe"} {
		if !forbiddenToolMember(name) {
			t.Fatalf("forbidden wheel member accepted: %s", name)
		}
	}
}

func TestToolMetadataHeadersAcceptLFAndCRLFWithoutWeakeningVersionMatch(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		data := []byte(strings.Join([]string{"Metadata-Version: 2.4", "Name: PTSIP", "Version: 0.3.8a6", "", "Description"}, newline))
		if !toolMetadataMatches(data, "0.3.8a6") || toolMetadataMatches(data, "0.3.8a5") {
			t.Fatalf("metadata version handling differs for %q", newline)
		}
	}
}

func TestToolBuildStagePreservesPinnedGitLicenseBytes(t *testing.T) {
	root := repoRoot(t)
	head, err := release.Run(root, "git", "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.TrimSpace(string(head))
	stage, cleanup, err := toolSource(root, source)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	expected, err := release.Run(root, "git", "show", source+":License-Authority/LICENSE.md")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(stage, "License-Authority/LICENSE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("derived Tool build stage changed canonical Git license bytes")
	}
}

func TestPackageTamperAndIncompleteInstallationFailBeforeAdapter(t *testing.T) {
	r := release.Registry{NumericBase: "0.3.8"}
	c := release.PublishedComponent{ID: "context", Kind: "engine", Version: "0.2.0", SourceSHA: strings.Repeat("a", 40), Files: []release.File{{Name: "engine.whl", SHA256: strings.Repeat("b", 64), URL: "https://example.invalid/engine.whl", Platform: "linux-x64"}}}
	s, e := release.RefreshPackage(&r, strings.Repeat("a", 40), []release.PublishedComponent{c})
	if e != nil {
		t.Fatal(e)
	}
	if e := release.InstallPackage(s, "linux-x64", "must-never-execute", map[string]string{"context": "0.2.0", "storage": "0.3.0"}); e == nil {
		t.Fatal("partial bootstrap installed")
	}
	r.Snapshots[0].Components[0].Files[0].SHA256 = strings.Repeat("c", 64)
	if _, e := release.ResolvePackage(r, "0.3.8"); e == nil {
		t.Fatal("tampered published registry resolved")
	}
}

func TestReceiptJoinRejectsMissingHostWrongSelectionAndCorruptFile(t *testing.T) {
	root := t.TempDir()
	s := release.Selection{SourceSHA: strings.Repeat("a", 40), SelectionSHA: strings.Repeat("b", 64), Tag: "tag", Components: []release.Component{{ID: "context", Kind: "engine", Version: "0.2.0"}}, Platforms: []string{"windows-x64", "linux-x64"}}
	for _, host := range s.Platforms {
		dir := filepath.Join(root, "verified", host)
		if e := os.MkdirAll(dir, 0755); e != nil {
			t.Fatal(e)
		}
		data := []byte(host)
		name := host + ".whl"
		if e := os.WriteFile(filepath.Join(dir, name), data, 0644); e != nil {
			t.Fatal(e)
		}
		r := Receipt{Schema: "ptsip-release-test-receipt/v1", SourceSHA: s.SourceSHA, SelectionSHA: s.SelectionSHA, Component: "context", Kind: "engine", Version: "0.2.0", Platform: host, Status: "PASS", Files: []release.File{{Name: name, SHA256: release.Digest(data), Platform: host}}}
		if e := release.WriteJSON(filepath.Join(dir, "test-receipt.json"), r); e != nil {
			t.Fatal(e)
		}
	}
	if _, _, e := release.Gather(root, s, "verified"); e != nil {
		t.Fatal(e)
	}
	s.SelectionSHA = strings.Repeat("c", 64)
	if _, _, e := release.Gather(root, s, "verified"); e == nil {
		t.Fatal("wrong selection receipt joined")
	}
	s.SelectionSHA = strings.Repeat("b", 64)
	path := filepath.Join(root, "verified/linux-x64/linux-x64.whl")
	if e := os.WriteFile(path, []byte("corrupt"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, _, e := release.Gather(root, s, "verified"); e == nil {
		t.Fatal("corrupt artifact joined")
	}
	if e := os.RemoveAll(filepath.Join(root, "verified/linux-x64")); e != nil {
		t.Fatal(e)
	}
	if _, _, e := release.Gather(root, s, "verified"); e == nil {
		t.Fatal("one-host result admitted")
	}
}
