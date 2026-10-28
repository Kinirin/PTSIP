package releasecontroller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func ConsumePackageReceipt(root string, s Selection, snapshot Snapshot, input, output, pypiVersion string) (*PublishedComponent, error) {
	dir, err := SafePath(root, input)
	if err != nil {
		return nil, err
	}
	rows := map[string]PackageReceipt{}
	payloads := map[string][]byte{}
	files := []File{}
	err = filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked Package verification artifact")
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
		if r.Schema != "ptsip-pip-package-test-receipt/v2" || r.Source != s.SourceSHA || r.Selection != s.SelectionSHA || r.Status != "PASS" || r.PyPI != pypiVersion || r.Snapshot.Digest != snapshot.Digest {
			return fmt.Errorf("complete pip Package receipt differs from original component admission")
		}
		if err := ValidateSnapshot(r.Snapshot); err != nil {
			return err
		}
		if _, ok := rows[r.Platform]; ok {
			return fmt.Errorf("duplicate Package host receipt")
		}
		rows[r.Platform] = r
		name, err := BundledPackageWheelName(pypiVersion, r.Platform)
		if err != nil || r.Wheel.Name != name || r.Wheel.Platform != r.Platform {
			return fmt.Errorf("Package native wheel host differs")
		}
		admitted := []File{r.Wheel}
		if r.SourceDist != nil {
			if r.Platform != "linux-x64" || r.SourceDist.Name != "ptsip-"+pypiVersion+".tar.gz" || r.SourceDist.Platform != "any" {
				return fmt.Errorf("Package source receipt differs")
			}
			admitted = append(admitted, *r.SourceDist)
		}
		for _, file := range admitted {
			if filepath.Base(file.Name) != file.Name {
				return fmt.Errorf("invalid Package distribution name")
			}
			data, err := os.ReadFile(filepath.Join(filepath.Dir(p), file.Name))
			if err != nil || Digest(data) != file.SHA256 {
				return fmt.Errorf("verified Package distribution changed")
			}
			if _, exists := payloads[file.Name]; exists {
				return fmt.Errorf("duplicate Package distribution")
			}
			payloads[file.Name] = data
			file.URL = "https://github.com/Kinirin/PTSIP/releases/download/" + s.Tag + "/" + file.Name
			files = append(files, file)
		}
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
	if len(rows) != len(s.Platforms) || len(files) != len(s.Platforms)+1 {
		return nil, fmt.Errorf("Package requires selected host wheels and one verified source distribution")
	}
	out, err := SafePath(root, output)
	if err != nil {
		return nil, err
	}
	stage := filepath.Join(out, "pypi/package")
	if err := os.MkdirAll(stage, 0755); err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(out, file.Name), payloads[file.Name], 0644); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(stage, file.Name), payloads[file.Name], 0644); err != nil {
			return nil, err
		}
	}
	return &PublishedComponent{ID: "package", Kind: "package", Version: pypiVersion, SourceSHA: s.SourceSHA, Files: files}, nil
}
