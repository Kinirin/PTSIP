package releasetest

import (
	"archive/zip"
	"bytes"
	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
	"io"
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
