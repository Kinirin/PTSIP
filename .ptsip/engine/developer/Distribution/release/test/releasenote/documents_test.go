package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	documentation "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/releasenote"
)

func documentationRepository(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func copyDocumentationFile(t *testing.T, root, path string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(documentationRepository(t), filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func documentationGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, raw)
	}
	return strings.TrimSpace(string(raw))
}

func documentationVersions(changed bool) string {
	tool, profile, pkg, validation := "T.1.0.0.b1", "pp.1.00", "1.0.0", "1.0.0"
	if changed {
		tool, profile, pkg, validation = "T.1.0.0.b2", "pp.1.01", "1.0.1", "1.1.0"
	}
	text := fmt.Sprintf("[package]\npackage_version = %q\n[tool]\ntool_version = %q\n[profile]\nprofile_version = %q\n[engines]\n", pkg, tool, profile)
	for _, engine := range []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"} {
		version := "1.0.0"
		if engine == "validation" {
			version = validation
		}
		text += fmt.Sprintf("%s = %q\n", engine, version)
	}
	return text
}

func documentationFixture(t *testing.T, changed bool) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	paths := []string{"README.md", "docs/translated/index.yaml", documentation.WorkflowPath,
		".ptsip/contracts/release/versions_different.json", ".ptsip/contracts/schemas/release/versions_different.schema.json",
		"docs/validation-schema-references.md", "src/ptsip/profiles/index.yaml", "src/ptsip/engines/validation/contract.json", "src/ptsip/engines/contracts/installation-plan.json"}
	for _, schema := range []string{"Basic_Formatting_readme", "engine_Formatting_readme", "profile_Formatting_readme", "tool_Formatting_readme", "readme-translation-path", "readme-translation-registry"} {
		paths = append(paths, documentation.SchemaRoot+schema+".schema.json")
	}
	for _, path := range paths {
		copyDocumentationFile(t, root, path)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs/releasenote"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src/VERSION"), []byte(documentationVersions(false)), 0644); err != nil {
		t.Fatal(err)
	}
	documentationGit(t, root, "init", "-q")
	documentationGit(t, root, "config", "user.name", "Documentation test")
	documentationGit(t, root, "config", "user.email", "documentation@example.invalid")
	documentationGit(t, root, "add", ".")
	documentationGit(t, root, "commit", "-qm", "baseline")
	base := documentationGit(t, root, "rev-parse", "HEAD")
	if changed {
		if err := os.WriteFile(filepath.Join(root, "src/VERSION"), []byte(documentationVersions(true)), 0644); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := os.WriteFile(filepath.Join(root, "src/ptsip/changed.py"), []byte("def candidate(): pass\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	documentationGit(t, root, "add", ".")
	documentationGit(t, root, "commit", "-qm", "candidate")
	return root, base, documentationGit(t, root, "rev-parse", "HEAD")
}

func TestDeclaredChangesGenerateFourClassifiedEnglishArchives(t *testing.T) {
	root, base, head := documentationFixture(t, true)
	report, code := documentation.Run(root, "generate", base, head, "", documentationValidator)
	if code != 0 {
		t.Fatalf("generation failed: %v", report)
	}
	for _, path := range []string{
		"docs/releasenote/engine/validation/validation-engine-1.1.0.md",
		"docs/releasenote/project-profile/pp.1.01.md",
		"docs/releasenote/tool/T.1.0.0-b2.md",
		"docs/releasenote/PTSIP_Package/1.0.1.md",
	} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(raw, []byte(`"language":"en"`)) || !bytes.Contains(raw, []byte("## Technical details\n")) || !bytes.Contains(raw, []byte("## Documentation\n")) {
			t.Fatalf("missing original-language formatting: %s", path)
		}
	}
	catalog, err := documentation.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.CheckCurrent(documentationValidator); err != nil {
		t.Fatal(err)
	}
	count, err := catalog.CheckArchive(documentationValidator)
	if err != nil || count != 4 {
		t.Fatalf("archive classifications: count=%d err=%v", count, err)
	}
	report, code = documentation.Run(root, "generate", base, head, "", documentationValidator)
	if code != 0 || len(report.(map[string]any)["output_paths"].([]string)) != 0 {
		t.Fatalf("generation is not idempotent: %v", report)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/releasenote/engine/README.md")); !os.IsNotExist(err) {
		t.Fatal("withdrawn engine index generated")
	}
	if _, err := os.Stat(filepath.Join(root, "docs/releasenote/translated")); !os.IsNotExist(err) {
		t.Fatal("translations archived")
	}
}

func TestSourceChangesWithoutDeclaredVersionChangesDoNotWriteDocuments(t *testing.T) {
	root, base, head := documentationFixture(t, false)
	before, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	report, code := documentation.Run(root, "generate", base, head, "", documentationValidator)
	if code != 0 || report.(map[string]any)["status"] != "NO_VERSION_CHANGES" {
		t.Fatalf("source syntax inferred a version change: %v", report)
	}
	after, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("README changed without a declared version change")
	}
}

func TestArchiveConflictAndConcurrentReadmeEditPreserveEveryExistingFile(t *testing.T) {
	root, base, head := documentationFixture(t, true)
	catalog, err := documentation.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	edits, _, err := catalog.Prepare(base, head, documentation.Input{}, documentationValidator)
	if err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(root, "README.md")
	original, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(root, "docs/releasenote/tool/T.1.0.0-b2.md")
	if err := os.MkdirAll(filepath.Dir(conflict), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conflict, []byte("Existing original"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Apply(edits); err == nil {
		t.Fatal("archive overwritten")
	}
	current, _ := os.ReadFile(readmePath)
	if !bytes.Equal(original, current) {
		t.Fatal("README changed before archive conflict was resolved")
	}
	if err := os.Remove(conflict); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmePath, []byte("Concurrent manual edit"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Apply(edits); err == nil {
		t.Fatal("concurrent README edit overwritten")
	}
	current, _ = os.ReadFile(readmePath)
	if string(current) != "Concurrent manual edit" {
		t.Fatal("concurrent content lost")
	}
}

func TestFormattingAndRegistryContractsRejectWrongOrderAndLanguage(t *testing.T) {
	root := documentationRepository(t)
	catalog, err := documentation.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := documentation.ParseReadme(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	doc.Sections[0], doc.Sections[1] = doc.Sections[1], doc.Sections[0]
	if err := catalog.Validate(documentationValidator, documentation.BasicID, doc); err == nil {
		t.Fatal("unordered README sections accepted")
	}
	registry := map[string]any{"schema_version": "ptsip-readme-translation-registry/v1", "canonical": map[string]any{"language": "it", "path": "README.md"}, "translations": []any{}}
	if err := documentation.ValidateRegistry(root, registry, documentationValidator); err == nil {
		t.Fatal("non-English canonical README accepted")
	}
	registry["canonical"] = map[string]any{"language": "en", "path": "README.md"}
	if err := documentation.ValidateRegistry(root, registry, documentationValidator); err != nil {
		t.Fatalf("English documentation must support an explicitly empty translation set: %v", err)
	}
	registry["translations"] = []any{map[string]any{"language": "ko", "path": "docs/translated/README.ja.md"}}
	if err := documentation.ValidateRegistry(root, registry, documentationValidator); err == nil {
		t.Fatal("language/path mismatch accepted")
	}
	raw, _ = json.Marshal(registry)
	if bytes.Contains(raw, []byte("docs/releasenote")) {
		t.Fatal("translation contract unexpectedly uses archive paths")
	}
}
