package releasetest

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPyPIArchiveRemovesDescriptorsAndPreservesEveryMember(t *testing.T) {
	var source bytes.Buffer
	w := zip.NewWriter(&source)
	for _, name := range []string{"engine.data/scripts/native.exe", "engine.dist-info/RECORD", "engine.dist-info/ptsip-origin.json"} {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte("unchanged:" + name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	repaired, err := release.PyPIArchive(source.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	a, _ := zip.NewReader(bytes.NewReader(source.Bytes()), int64(source.Len()))
	b, err := zip.NewReader(bytes.NewReader(repaired), int64(len(repaired)))
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range b.File {
		if f.Flags&8 != 0 {
			t.Fatal("PyPI-forbidden descriptor remains")
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		actual, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		r, err = a.File[i].Open()
		if err != nil {
			t.Fatal(err)
		}
		expected, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name != a.File[i].Name || !bytes.Equal(actual, expected) || f.CRC32 != a.File[i].CRC32 {
			t.Fatal("archive repair changed member identity")
		}
	}
	again, err := release.PyPIArchive(repaired)
	if err != nil || !bytes.Equal(again, repaired) {
		t.Fatal("repair is not idempotent")
	}
}

func TestFreshEngineArchiveLinksNormalizedWheelToOriginalBinaryReceipt(t *testing.T) {
	root := t.TempDir()
	var source bytes.Buffer
	w := zip.NewWriter(&source)
	f, err := w.Create("engine.data/scripts/native")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("native-executable-unchanged")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 40)
	name := "ptsip_engine_context-0.2.1-py3-none-linux_x86_64.whl"
	original := filepath.Join(root, name)
	if err := os.WriteFile(original, source.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	receipt := map[string]any{"wheel_sha256": release.Digest(source.Bytes()), "source_sha": sha, "binary_sha256": release.Digest([]byte("native-executable-unchanged"))}
	if err := release.WriteJSON(original+".receipt.json", receipt); err != nil {
		t.Fatal(err)
	}
	c := release.PublishedComponent{ID: "context", Kind: "engine", Version: "0.2.1", SourceSHA: sha, Files: []release.File{{Name: name, Platform: "linux-x64", SHA256: release.Digest(source.Bytes())}, {Name: name + ".receipt.json", Platform: "linux-x64"}}}
	key := "context/linux-x64/" + name
	locations := map[string]string{key: original, key + ".receipt.json": original + ".receipt.json"}
	s := release.Selection{SourceSHA: sha, Tag: "ptsip-release-" + sha}
	if err := release.NormalizeEngineArchives(root, s, &c, locations); err != nil {
		t.Fatal(err)
	}
	if c.SourceSHA != sha || c.Files[0].SHA256 == release.Digest(source.Bytes()) || locations[key] == original {
		t.Fatal("fresh engine source or normalized archive identity differs")
	}
	b, err := os.ReadFile(locations[key+".receipt.json"])
	if err != nil {
		t.Fatal(err)
	}
	var actual map[string]any
	if err := json.Unmarshal(b, &actual); err != nil {
		t.Fatal(err)
	}
	if actual["binary_sha256"] != receipt["binary_sha256"] || actual["source_sha"] != sha || actual["wheel_sha256"] != c.Files[0].SHA256 {
		t.Fatal("normalization lost original native evidence")
	}
	if err := release.NormalizeEngineArchives(root, s, &c, locations); err != nil {
		t.Fatal("normalization is not repeatable", err)
	}
}
