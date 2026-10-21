package releasecontroller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ConsumePackageReceipt(root string, s Selection, snapshot Snapshot, input, output, pypiVersion string) (*PublishedComponent, error) {
	dir, err := SafePath(root, input)
	if err != nil {
		return nil, err
	}
	rows := map[string]PackageReceipt{}
	var payload []byte
	var file File
	err = filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.Name() != "package-receipt.json" || e.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var r PackageReceipt
		if err := json.Unmarshal(b, &r); err != nil {
			return err
		}
		if r.Schema != "ptsip-pip-package-test-receipt/v1" || r.Source != s.SourceSHA || r.Selection != s.SelectionSHA || r.Status != "PASS" || r.PyPI != pypiVersion || r.Snapshot.Digest != snapshot.Digest {
			return fmt.Errorf("complete pip Package receipt differs from original component admission")
		}
		if _, ok := rows[r.Platform]; ok {
			return fmt.Errorf("duplicate Package host receipt")
		}
		rows[r.Platform] = r
		if filepath.Base(r.Wheel.Name) != r.Wheel.Name {
			return fmt.Errorf("invalid Package wheel name")
		}
		wheel, err := os.ReadFile(filepath.Join(filepath.Dir(p), r.Wheel.Name))
		if err != nil || Digest(wheel) != r.Wheel.SHA256 {
			return fmt.Errorf("verified Package wheel changed")
		}
		if payload != nil && Digest(payload) != r.Wheel.SHA256 {
			return fmt.Errorf("Package metadata wheel differs between required hosts")
		}
		payload, file = wheel, r.Wheel
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, host := range s.Platforms {
		if _, ok := rows[host]; !ok {
			return nil, fmt.Errorf("complete Package installation verification missing for %s", host)
		}
	}
	if len(rows) != len(s.Platforms) {
		return nil, fmt.Errorf("Package host receipt outside selected platforms")
	}
	out, err := SafePath(root, output)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(out, file.Name), payload, 0644); err != nil {
		return nil, err
	}
	stage := filepath.Join(out, "pypi/package")
	if err := os.MkdirAll(stage, 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(stage, file.Name), payload, 0644); err != nil {
		return nil, err
	}
	file.URL = "https://github.com/Kinirin/PTSIP/releases/download/" + s.Tag + "/" + file.Name
	return &PublishedComponent{ID: "package", Kind: "package", Version: pypiVersion, SourceSHA: s.SourceSHA, Files: []File{file}}, nil
}
