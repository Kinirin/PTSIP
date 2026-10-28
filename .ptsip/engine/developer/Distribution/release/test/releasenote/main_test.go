package main

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestRegistryLanguagesAreDataNotCode(t *testing.T) {
    root := t.TempDir()
    dir := filepath.Join(root, "docs", "translated")
    if err := os.MkdirAll(dir, 0755); err != nil { t.Fatal(err) }
    registry := "schema_version: ptsip-readme-translation-registry/v1\n"+
        "canonical:\n  language: it\n  path: README.md\n"+
        "translations:\n"+
        "  - language: de\n    path: docs/translated/README.de.md\n"+
        "  - language: fr\n    path: docs/translated/README.fr.md\n"
    if err := os.WriteFile(filepath.Join(dir, "index.yaml"), []byte(registry), 0644); err != nil { t.Fatal(err) }
    r, err := loadRegistry(root, "docs/translated/index.yaml")
    if err != nil { t.Fatal(err) }
    if r.Canonical.Language != "it" || len(r.Translations) != 2 ||
        r.Translations[0].Language != "de" || r.Translations[1].Language != "fr" {
        t.Fatalf("unexpected registry: %+v", r)
    }
}

func TestUnsafeRegistryTargetFailsClosed(t *testing.T) {
    root := t.TempDir()
    dir := filepath.Join(root, "docs", "translated")
    if err := os.MkdirAll(dir, 0755); err != nil { t.Fatal(err) }
    content := "schema_version: ptsip-readme-translation-registry/v1\n"+
        "canonical:\n  language: it\n  path: README.md\n"+
        "translations:\n"+
        "  - language: fr\n    path: ../../outside.md\n"
    if err := os.WriteFile(filepath.Join(dir, "index.yaml"), []byte(content), 0644); err != nil { t.Fatal(err) }
    if _, err := loadRegistry(root, "docs/translated/index.yaml"); err == nil {
        t.Fatal("unsafe output location was accepted")
    }
}

func TestLinkEquivalenceUsesResolvedTarget(t *testing.T) {
    source := "# Document\n\n## Reference\n\nSee [release](docs/releasenote/tool/0.3.8a5.md).\n"
    localized := "<p align=\"right\"><a href=\"../../README.md\">it</a> | fr</p>\n"+
        "# Document\n\n## Référence\n\nVoir [version](../../docs/releasenote/tool/0.3.8a5.md).\n"
    r := translationRegistry{Canonical: languageEntry{Language:"it",Path:"README.md"}}
    target := languageEntry{Language:"fr",Path:"docs/translated/README.fr.md"}
    if err := validate(source, localized, r, target); err != nil { t.Fatal(err) }
    localized = strings.ReplaceAll(localized, "../../docs/releasenote", "../../bad")
    if err := validate(source, localized, r, target); err == nil {
        t.Fatal("wrong target was accepted")
    }
}

func TestFencedBlocksAndMemory(t *testing.T) {
    source := "# Title\n\n## Section\n\n"+"\x60\x60\x60text\n\nKEEP\n\x60\x60\x60"+"\n\nFooter"
    blocks := blockSplit(source)
    if len(blocks) != 4 { t.Fatalf("blocks=%d", len(blocks)) }
    if len(fenced(source)) != 1 || !strings.Contains(fenced(source)[0], "\n\nKEEP") {
        t.Fatal("fence content was not preserved")
    }
    sha := hashBlock("## Section")
    old := "<!-- ptsip-readme-i18n: "+sha+" -->\n\n## Élément"
    if got := oldMemory(old)[sha]; got != "## Élément" {
        t.Fatalf("memory lookup: %q", got)
    }
}

func TestCodeTokensMustBePreserved(t *testing.T) {
    source := "# Title\n\n## Section\n\nUse \x60--check\x60."
    localized := "<p align=\"right\"><a href=\"../../README.md\">it</a> | de</p>"+
        "\n# Title\n\n## Abschnitt\n\nUse the option."
    r := translationRegistry{Canonical: languageEntry{Language:"it",Path:"README.md"}}
    target := languageEntry{Language:"de",Path:"docs/translated/README.de.md"}
    if err := validate(source, localized, r, target); err == nil {
        t.Fatal("missing inline code accepted")
    }
}
