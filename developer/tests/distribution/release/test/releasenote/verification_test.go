package translation_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIndependentVerifierDetectsFreshnessWithoutMutation(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs/translated"), 0755); err != nil {
		t.Fatal(err)
	}
	source := "# Title\n\n## Detail\n\nUse `--check`."
	registerLanguages(t, root, "en", "fr")
	path := "docs/translated/README.fr.md"
	document := "<p align=\"right\"><a href=\"../../README.md\">en</a> | fr</p>\n" + source
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, filepath.FromSlash(path))
	if err := os.WriteFile(dest, []byte(document), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readmeCLI(t, root, "verify"); err != nil {
		t.Fatal(err)
	}
	if _, err := readmeCLI(t, root, "verify-strict"); err == nil {
		t.Fatal("strict verifier accepted unmarked legacy projection")
	}
	marker := expectedMarker("## Detail") + "\n\n## Detail\n\n"
	marker2 := expectedMarker("Use `--check`.") + "\n\nUse `--check`."
	documented := "<p align=\"right\"><a href=\"../../README.md\">en</a> | fr</p>\n# Title\n\n" + marker + marker2
	if err := os.WriteFile(dest, []byte(documented), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readmeCLI(t, root, "verify-strict"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(source+"\n\nExtra"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readmeCLI(t, root, "verify-strict"); err == nil {
		t.Fatal("outdated projection accepted")
	}
	actual, err := os.ReadFile(dest)
	if err != nil || string(actual) != documented {
		t.Fatal("verifier mutated document", err)
	}
}

func TestVerifierRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture requires platform privileges")
	}
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "other.md")
	if err := os.MkdirAll(filepath.Join(root, "docs/translated"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "docs/translated/README.fr.md")); err != nil {
		t.Fatal(err)
	}
	registerLanguages(t, root, "en", "fr")
	writeTranslationFixture(t, root, "README.md", "# Title")
	if output, err := readmeCLI(t, root, "verify"); err == nil || !strings.Contains(output, "symlink target forbidden") {
		t.Fatalf("symlink verification: %s %v", output, err)
	}
}

func TestApplyPreparesAllLanguagesBeforeMutation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell fixture")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	mock := "#!/bin/sh\nfor a in \"$@\"; do if [ \"$a\" = fr ]; then exit 13; fi; done\ncat\n"
	if err := os.WriteFile(filepath.Join(bin, "argos-translate"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	source := "# Title\n\n## Section\n\nHello, world."
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs/translated"), 0755); err != nil {
		t.Fatal(err)
	}
	de := filepath.Join(root, "docs/translated/README.de.md")
	if err := os.WriteFile(de, []byte("PREVIOUS DE"), 0644); err != nil {
		t.Fatal(err)
	}
	registerLanguages(t, root, "en", "de", "fr")
	_, err := readmeCLI(t, root, "apply")
	if err == nil {
		t.Fatal("failed Argos conversion did not stop apply")
	}
	text, readErr := os.ReadFile(de)
	if readErr != nil || string(text) != "PREVIOUS DE" {
		t.Fatalf("partial mutation: %q %v", text, readErr)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/translated/README.fr.md")); !os.IsNotExist(err) {
		t.Fatal("failed target was written")
	}
	if !strings.Contains(err.Error(), "fr") {
		t.Fatal("failed language not in diagnostic", err)
	}
}

func TestApplyChangesOnlyRegisteredTargetsAndIsIdempotent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell fixture")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "argos-translate"), []byte("#!/bin/sh\ncat\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.MkdirAll(filepath.Join(root, "docs/translated"), 0755); err != nil {
		t.Fatal(err)
	}
	source := "# Title\n\n## Note\n\nUse `--flag` and [guide](docs/guide.md)."
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(root, "docs/translated/README.other.md")
	if err := os.WriteFile(unrelated, []byte("KEEP"), 0644); err != nil {
		t.Fatal(err)
	}
	registerLanguages(t, root, "en", "de", "fr")
	if _, err := readmeCLI(t, root, "apply"); err != nil {
		t.Fatal(err)
	}
	first := map[string]string{}
	for _, language := range []string{"de", "fr"} {
		filename := filepath.Join(root, "docs", "translated", "README."+language+".md")
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		first[filename] = string(data)
	}
	if _, err := readmeCLI(t, root, "verify-strict"); err != nil {
		t.Fatal(err)
	}
	if _, err := readmeCLI(t, root, "apply"); err != nil {
		t.Fatal(err)
	}
	for filename, previous := range first {
		actual, err := os.ReadFile(filename)
		if err != nil || string(actual) != previous {
			t.Fatalf("not idempotent %s: %v", filename, err)
		}
	}
	data, err := os.ReadFile(unrelated)
	if err != nil || string(data) != "KEEP" {
		t.Fatal("unregistered target was changed", err)
	}
	data, err = os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil || string(data) != source {
		t.Fatal("canonical README was changed", err)
	}
}
