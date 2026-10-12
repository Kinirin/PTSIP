package enginewheel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPyPIReuseRequiresExactPublishedFileDigests(t *testing.T) {
	bundle := PublicationBundle{Wheels: []string{"windows.whl", "linux.whl", "macos.whl"}, Files: map[string]string{"windows.whl": strings.Repeat("a", 64), "linux.whl": strings.Repeat("b", 64), "macos.whl": strings.Repeat("c", 64)}}
	for _, name := range []string{"empty", "partial", "complete", "changed-content", "unexpected-platform", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			files := []pypiFile{}
			for _, wheel := range bundle.Wheels {
				file := pypiFile{Filename: wheel}
				file.Digests.SHA256 = bundle.Files[wheel]
				files = append(files, file)
			}
			invalid := false
			switch name {
			case "empty":
				files = nil
			case "partial":
				files = files[:1]
			case "changed-content":
				files[0].Digests.SHA256 = strings.Repeat("d", 64)
				invalid = true
			case "unexpected-platform":
				files[0].Filename = "other.whl"
				invalid = true
			case "duplicate":
				files = append(files, files[0])
				invalid = true
			}
			result, err := checkPyPIFiles(bundle, files)
			if invalid && err == nil {
				t.Fatal("unrelated immutable PyPI file was accepted")
			}
			if !invalid && (err != nil || len(result) != len(files)) {
				t.Fatalf("valid partial or complete publication rejected: %v", err)
			}
		})
	}
}

func TestPublicationRejectsMissingPlatformsAndUnregisteredComponents(t *testing.T) {
	for _, name := range []string{"tool", "missing-platforms", "source", "run", "escape"} {
		t.Run(name, func(t *testing.T) {
			root, _ := fixture(t, "x86_64-pc-windows-msvc")
			input := filepath.Join(root, "input")
			if err := os.Mkdir(input, 0755); err != nil {
				t.Fatal(err)
			}
			opts := PublicationOptions{GateOptions: GateOptions{Kind: "context", SourceSHA: strings.Repeat("a", 40), RunID: 1}, Input: "input", Output: "output"}
			switch name {
			case "tool":
				opts.Kind = "tool"
			case "source":
				opts.SourceSHA = "not-a-sha"
			case "run":
				opts.RunID = 0
			case "escape":
				opts.Input = "../outside"
			}
			if _, err := validatePublicationBundle(root, opts); err == nil {
				t.Fatal("unverified publication bundle was admitted")
			}
			if _, err := os.Stat(filepath.Join(root, "output")); !os.IsNotExist(err) {
				t.Fatal("rejected publication wrote output")
			}
		})
	}
}

func TestLinuxABIRejectsOtherExecutableFormats(t *testing.T) {
	for _, data := range [][]byte{[]byte("MZ"), {0xcf, 0xfa, 0xed, 0xfe}, []byte("not an ELF binary")} {
		if err := linuxABI(data); err == nil {
			t.Fatal("non-Linux artifact accepted as manylinux")
		}
	}
}
