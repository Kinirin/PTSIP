package releasecontroller

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PyPI rejects streaming ZIP data descriptors. Copy the already checked raw
// compressed entries with their known CRC/sizes into explicit local headers.
// Member bytes, RECORD hashes, engine binaries and original source stay intact.
func PyPIArchive(data []byte) ([]byte, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	changed := false
	seen := map[string]bool{}
	for _, f := range z.File {
		if seen[f.Name] || f.Mode()&os.ModeSymlink != 0 || f.UncompressedSize64 > 128*1024*1024 {
			return nil, fmt.Errorf("unsafe PyPI ZIP entry")
		}
		seen[f.Name] = true
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(io.Discard, r)
		r.Close()
		if err != nil {
			return nil, err
		}
		changed = changed || f.Flags&8 != 0
	}
	if !changed {
		return data, nil
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, f := range z.File {
		h := f.FileHeader
		h.Flags &^= 8
		raw, err := f.OpenRaw()
		if err != nil {
			return nil, err
		}
		member, err := w.CreateRaw(&h)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(member, raw); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func NormalizeEngineArchives(root string, s Selection, c *PublishedComponent, locations map[string]string) error {
	for i := range c.Files {
		f := &c.Files[i]
		if !strings.HasSuffix(f.Name, ".whl") {
			continue
		}
		key := c.ID + "/" + f.Platform + "/" + f.Name
		original, err := os.ReadFile(locations[key])
		if err != nil {
			return err
		}
		normalized, err := PyPIArchive(original)
		if err != nil {
			return err
		}
		if bytes.Equal(original, normalized) {
			continue
		}
		originalSHA := Digest(original)
		stage := filepath.Join(root, ".artifacts/release/normalized", c.ID, f.Name)
		if err := os.MkdirAll(filepath.Dir(stage), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(stage, normalized, 0644); err != nil {
			return err
		}
		locations[key] = stage
		f.SHA256 = Digest(normalized)
		f.URL = "https://github.com/Kinirin/PTSIP/releases/download/" + s.Tag + "/" + f.Name
		found := false
		for j := range c.Files {
			receipt := &c.Files[j]
			if receipt.Name != f.Name+".receipt.json" {
				continue
			}
			receiptKey := c.ID + "/" + receipt.Platform + "/" + receipt.Name
			b, err := os.ReadFile(locations[receiptKey])
			if err != nil {
				return err
			}
			var r map[string]any
			if err := json.Unmarshal(b, &r); err != nil {
				return err
			}
			if r["wheel_sha256"] != originalSHA || r["source_sha"] != c.SourceSHA {
				return fmt.Errorf("original native receipt differs before ZIP normalization")
			}
			r["wheel_sha256"] = f.SHA256
			r["archive_normalization"] = map[string]any{"original_wheel_sha256": originalSHA, "operation": "REMOVE_ZIP_DATA_DESCRIPTORS_ONLY", "member_contents_and_binary": "UNCHANGED"}
			target := stage + ".receipt.json"
			if err := WriteJSON(target, r); err != nil {
				return err
			}
			b, err = os.ReadFile(target)
			if err != nil {
				return err
			}
			locations[receiptKey] = target
			receipt.SHA256 = Digest(b)
			receipt.URL = f.URL + ".receipt.json"
			found = true
		}
		if !found {
			return fmt.Errorf("original native receipt missing for ZIP normalization")
		}
	}
	return nil
}
