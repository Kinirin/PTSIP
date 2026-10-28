package releasenote_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	documentation "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/releasenote"
	version "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/version"
	policy "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
)

func TestFormattingContractsThroughInstalledNativeValidation(t *testing.T) {
	root, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	validate, err := policy.InstalledValidator()
	if err != nil {
		t.Skipf("installed PTSIP VALIDATION unavailable: %v", err)
	}
	catalog, err := documentation.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.CheckCurrent(validate); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme, err := documentation.ParseReadme(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	readme.Sections[0], readme.Sections[1] = readme.Sections[1], readme.Sections[0]
	if err := catalog.Validate(validate, documentation.BasicID, readme); err == nil {
		t.Fatal("native engine accepted unordered README sections")
	}
	for _, item := range []struct{ kind, component, previous, current string }{
		{"engine", "engine.validation", "0.3.4", "0.3.5"},
		{"profile", "profile", "pp.2.00", "pp.2.01"},
		{"tool", "tool", "T.0.3.8.b2", "T.0.3.8.b3"},
		{"package", "package", "0.3.8-b7", "0.3.8-b8"},
	} {
		note := documentation.Note{Schema: "ptsip-release-note/v1", Language: "en", Kind: item.kind,
			Change: version.DeclaredVersionChange{Component: item.component, Previous: item.previous, Version: item.current, Files: []string{"src/VERSION"}},
			Base:   strings.Repeat("a", 40), Head: strings.Repeat("b", 40), Digest: strings.Repeat("c", 64), Summary: "Example declared version change.", Details: "Explicit technical details.", Documentation: []documentation.Link{{Title: "Original technical document", Path: "README.md"}}}
		if err := catalog.CheckNote(note, validate); err != nil {
			t.Fatalf("%s format rejected: %v", item.kind, err)
		}
		note.Language = "ko"
		if err := catalog.CheckNote(note, validate); err == nil {
			t.Fatalf("%s translated archive accepted", item.kind)
		}
	}
}
