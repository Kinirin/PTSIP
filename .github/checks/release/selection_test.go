package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

func vector() map[string]string {
	return map[string]string{"tool": "0.3.8a5", "profile": "pp.1.02", "context": "0.2", "storage": "0.3", "validation": "0.3", "discovery": "0.3", "planning": "0.3", "execution": "0.3", "projection": "0.2"}
}
func TestPackageVersionMapsEveryIndependentVersionWithoutNumericBump(t *testing.T) {
	v := vector()
	first, hash, err := PackageIdentity("0.3.8", v)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, "0.3.8.") || len(hash) != 64 || first != "0.3.8."+hash[:6] {
		t.Fatal(first, hash)
	}
	for id := range v {
		next := vector()
		next[id] += "a1"
		version, newHash, err := PackageIdentity("0.3.8", next)
		if err != nil || version == first || newHash == hash || !strings.HasPrefix(version, "0.3.8.") {
			t.Fatalf("%s change was coupled or omitted: %s %v", id, version, err)
		}
	}
	again, _, _ := PackageIdentity("0.3.8", vector())
	if again != first {
		t.Fatal("same composition must retain identity")
	}
	manual, _, _ := PackageIdentity("1.0.0", vector())
	if !strings.HasPrefix(manual, "1.0.0.") {
		t.Fatal("explicit base was not consumed")
	}
	delete(v, "context")
	if _, _, err := PackageIdentity("0.3.8", v); err == nil {
		t.Fatal("incomplete composition admitted")
	}
}
func TestPublicEngineVersionAndCargoProjectionAreSeparate(t *testing.T) {
	for public, want := range map[string]string{"0.2": "0.2.0", "0.2a1": "0.2.0-alpha.1", "2.5b12": "2.5.0-beta.12", "1.4rc3": "1.4.0-rc.3"} {
		got, err := CargoVersion(public)
		if err != nil || got != want {
			t.Fatal(public, got, err)
		}
	}
	for _, bad := range []string{"0.2.0", "0.2a", "0.2a0", "01.2", "0.2.alpha1", "0.2.1a1"} {
		if _, err := CargoVersion(bad); err == nil {
			t.Fatalf("unapproved public version accepted: %s", bad)
		}
	}
}
func TestSelectionDoesNotInferOtherEngineOrToolReleases(t *testing.T) {
	allowed := map[string]bool{"context": true, "tool": true, "profile": true, "package": true}
	s, err := selectionSet("context", allowed)
	if err != nil || len(s) != 1 || !s["context"] || s["tool"] {
		t.Fatal(s, err)
	}
	for _, bad := range []string{"context,context", "contracts", "Context", "context,missing"} {
		if _, err := selectionSet(bad, allowed); err == nil {
			t.Fatal("ambiguous target accepted", bad)
		}
	}
}
func TestArtifactVerificationRejectsTamperingAndMissingPlatforms(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "context-0.2-windows-x64.zip")
	if err := os.WriteFile(p, []byte("verified build"), 0600); err != nil {
		t.Fatal(err)
	}
	a, _ := digest(p)
	c := Component{ID: "context", Version: "0.2", Tag: "engine-context-v0.2", SourceSHA: strings.Repeat("a", 40), Fingerprint: strings.Repeat("b", 64), Platforms: []string{"windows-x64", "linux-x64"}}
	s := Selection{SourceSHA: c.SourceSHA, Components: []Component{c}}
	r := Receipt{Format: "ptsip-component-release-receipt/v1", Component: c.ID, Version: c.Version, Tag: c.Tag, SourceSHA: c.SourceSHA, Fingerprint: c.Fingerprint, Platform: "windows-x64", Verification: "PASS", Artifacts: []Artifact{a}}
	if err := writeJSON(filepath.Join(dir, "context-0.2-windows-x64.receipt.json"), r); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReceipts(s, "context", dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReceipts(s, "context", dir, true); err == nil {
		t.Fatal("partial platform release admitted")
	}
	if err := os.WriteFile(p, []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReceipts(s, "context", dir, false); err == nil {
		t.Fatal("artifact tampering admitted")
	}
	os.WriteFile(p, []byte("verified build"), 0600)
	os.WriteFile(filepath.Join(dir, "unverified.exe"), []byte("extra"), 0600)
	if _, err := VerifyReceipts(s, "context", dir, false); err == nil {
		t.Fatal("unverified payload admitted")
	}
}
func TestToolFingerprintExcludesIndependentEngineAndProfileData(t *testing.T) {
	dir := t.TempDir()
	repo := &machine.Repository{Root: dir}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init")
	git("config", "user.email", "release@example.invalid")
	git("config", "user.name", "release test")
	put := func(path, text string) {
		t.Helper()
		p := filepath.Join(dir, path)
		os.MkdirAll(filepath.Dir(p), 0700)
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	put("src/ptsip/cli.py", "tool implementation\n")
	put("src/ptsip/engines/context/manifest.json", "engine v1\n")
	put("src/ptsip/profiles/index.yaml", "profile v1\n")
	git("add", ".")
	git("commit", "-m", "baseline")
	base := git("rev-parse", "HEAD")
	c := Component{ID: "tool", Roots: []string{"src/ptsip"}}
	first, err := componentFingerprint(repo, base, c)
	if err != nil {
		t.Fatal(err)
	}
	put("src/ptsip/engines/context/manifest.json", "engine v2\n")
	put("src/ptsip/profiles/index.yaml", "profile v2\n")
	git("add", ".")
	git("commit", "-m", "independent versions")
	second, _ := componentFingerprint(repo, "HEAD", c)
	if second != first {
		t.Fatal("independent engine/Profile data forced a Tool release")
	}
	put("src/ptsip/cli.py", "changed tool implementation\n")
	git("add", ".")
	git("commit", "-m", "tool change")
	third, _ := componentFingerprint(repo, "HEAD", c)
	if third == first {
		t.Fatal("Tool source mutation was not detected")
	}
}
