package translation_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryLanguagesAreDataNotCode(t *testing.T) {
	root := t.TempDir()
	registerLanguages(t, root, "en", "de", "fr")
	raw, err := readmeCLI(t, root, "list-targets")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(strings.TrimSpace(raw), "\r\n", "\n") != "docs/translated/README.de.md\ndocs/translated/README.fr.md" {
		t.Fatalf("unexpected registry targets: %s", raw)
	}
	writeTranslationFixture(t, root, "README.md", "# Title\n\n\x60\x60\x60text\nregistry-fixture\n\x60\x60\x60")
	if _, err := readmeCLI(t, root, "apply"); err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"de", "fr"} {
		body, err := os.ReadFile(filepath.Join(root, "docs", "translated", "README."+lang+".md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `href="../../README.md">en</a> | `+lang) {
			t.Fatalf("registry language did not reach the rendered navigation: %s", body)
		}
	}
}

func TestUnsafeRegistryTargetFailsClosed(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "translated")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "schema_version: ptsip-readme-translation-registry/v1\n" +
		"canonical:\n  language: en\n  path: README.md\n" +
		"translations:\n" +
		"  - language: fr\n    path: ../../outside.md\n"
	if err := os.WriteFile(filepath.Join(dir, "index.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if output, err := readmeCLI(t, root, "list-targets"); err == nil || !strings.Contains(output, "unsafe registry path") {
		t.Fatalf("unsafe output location did not fail path validation: %s %v", output, err)
	}
}

func readmeCLI(t *testing.T, root, action string) (string, error) {
	t.Helper()
	module := filepath.Join(documentationRepository(t), ".ptsip", "engine", "developer", "Distribution", "release", "test")
	cmd := exec.Command("go", "-C", module, "run", "-mod=readonly", "./releasenote", "--root", root, "--action", action)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("readme-translate %s: %w: %s", action, err, output)
	}
	return string(output), nil
}

func writeTranslationFixture(t *testing.T, root, name, text string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func registerLanguages(t *testing.T, root, canonical string, langs ...string) {
	t.Helper()
	for _, schema := range []string{"readme-translation-path.schema.json", "readme-translation-registry.schema.json"} {
		copyDocumentationFile(t, root, ".ptsip/contracts/schemas/release/releasenote/"+schema)
	}
	text := "schema_version: ptsip-readme-translation-registry/v1\ncanonical:\n  language: " + canonical + "\n  path: README.md\ntranslations:\n"
	for _, lang := range langs {
		text += "  - language: " + lang + "\n    path: docs/translated/README." + lang + ".md\n"
	}
	writeTranslationFixture(t, root, "docs/translated/index.yaml", text)
}

func TestLinkEquivalenceUsesResolvedTarget(t *testing.T) {
	root := t.TempDir()
	registerLanguages(t, root, "en", "fr")
	source := "# Document\n\n## Reference\n\nSee [release](docs/releasenote/tool/0.3.8a5.md).\n"
	localized := "<p align=\"right\"><a href=\"../../README.md\">en</a> | fr</p>\n" +
		"# Document\n\n## Référence\n\nVoir [version](../../docs/releasenote/tool/0.3.8a5.md).\n"
	writeTranslationFixture(t, root, "README.md", source)
	writeTranslationFixture(t, root, "docs/translated/README.fr.md", localized)
	if _, err := readmeCLI(t, root, "verify"); err != nil {
		t.Fatal(err)
	}
	writeTranslationFixture(t, root, "docs/translated/README.fr.md", strings.ReplaceAll(localized, "../../docs/releasenote", "../../bad"))
	if output, err := readmeCLI(t, root, "verify"); err == nil || !strings.Contains(output, "missing canonical link target") {
		t.Fatalf("wrong target did not fail link verification: %s %v", output, err)
	}
}

func expectedMarker(block string) string {
	return fmt.Sprintf("<!-- ptsip-readme-i18n: %x -->", sha256.Sum256([]byte(block)))
}

func TestFencedBlocksAndMemory(t *testing.T) {
	root := t.TempDir()
	registerLanguages(t, root, "en", "fr")
	fence := "\x60\x60\x60text\n\nKEEP\n\x60\x60\x60"
	writeTranslationFixture(t, root, "README.md", "# Title\n\n## Section\n\n"+fence+"\n\nFooter")
	writeTranslationFixture(t, root, "docs/translated/README.fr.md", "# Title\n\n"+expectedMarker("## Section")+"\n\n## Élément\n\n"+expectedMarker("Footer")+"\n\nConclusion")
	if _, err := readmeCLI(t, root, "apply"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "docs/translated/README.fr.md"))
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	if strings.Count(result, "<!-- ptsip-readme-i18n: ") != 3 || !strings.Contains(result, fence) || !strings.Contains(result, "## Élément") {
		t.Fatalf("fence or exact source-block translation memory was not preserved: %s", result)
	}
}

func TestCodeTokensMustBePreserved(t *testing.T) {
	root := t.TempDir()
	registerLanguages(t, root, "en", "de")
	source := "# Title\n\n## Section\n\nUse \x60--check\x60."
	localized := "<p align=\"right\"><a href=\"../../README.md\">en</a> | de</p>" +
		"\n# Title\n\n## Abschnitt\n\nUse the option."
	writeTranslationFixture(t, root, "README.md", source)
	writeTranslationFixture(t, root, "docs/translated/README.de.md", localized)
	if output, err := readmeCLI(t, root, "verify"); err == nil || !strings.Contains(output, "missing inline code token") {
		t.Fatalf("omitted inline code did not fail token verification: %s %v", output, err)
	}
}
