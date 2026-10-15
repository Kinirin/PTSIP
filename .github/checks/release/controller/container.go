package releasecontroller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const ContainerContractPath = ".github/checks/release/container/contract.json"
const ContainerSource = "916333f9aa0f9589272911818d8352fb0cae09e1"
const ContainerMode = "release-container"

type ContainerContract struct {
	Schema        string   `json:"schema_version"`
	Authorization string   `json:"authorization"`
	Branch        string   `json:"branch"`
	Image         string   `json:"image"`
	Tags          []string `json:"tags"`
	Platform      string   `json:"platform"`
	Base          string   `json:"base_image"`
	Release       string   `json:"source_release"`
	ManifestSHA   string   `json:"shipping_manifest_sha256"`
	OriginalRun   int64    `json:"original_verification_run"`
	Engines       []string `json:"mandatory_engines"`
	Canonical     bool     `json:"canonical_current_composition_owner"`
	Cutover       bool     `json:"successor_info_implementation_cutover"`
	PolicyPass    bool     `json:"automated_policy_validation_available"`
	PyPI          bool     `json:"pypi_publication"`
	Merge         bool     `json:"merge_to_main"`
	Activation    bool     `json:"runtime_activation"`
}

func ReadContainerContract(root string) (ContainerContract, error) {
	var c ContainerContract
	if err := ReadJSON(root, ContainerContractPath, &c); err != nil {
		return c, err
	}
	if c.Schema != "ptsip-temporary-container-delivery/v1" || c.Authorization != "PROJECT_OWNER_DIRECT_INSTRUCTION_2026_10_15_TEMPORARY_0_3_8B3_CONTAINER" || c.Branch != "dev/0.3.8b2" || c.Image != "ghcr.io/kinirin/ptsip" || c.Platform != "linux/amd64" || c.Release != "ptsip-release-"+ContainerSource || c.OriginalRun != 38677942975 || c.ManifestSHA != "72eb4b0a443294dcd54480c7458353675e4494599a9110bab9957611bddc2a86" || c.Canonical || c.Cutover || c.PolicyPass || c.PyPI || c.Merge || c.Activation || strings.Join(c.Tags, ",") != "0.3.8b3,0.3.8.b3,0.3.8.f87bfe" || strings.Join(c.Engines, ",") != "context,storage,validation,discovery,planning,execution,projection" || !regexp.MustCompile(`^ubuntu:24\.04@sha256:[0-9a-f]{64}$`).MatchString(c.Base) {
		return c, fmt.Errorf("temporary container delivery authorization differs")
	}
	return c, nil
}

// This selects the container recipe SHA. Its application artifacts retain their
// separate, already published source SHA. No current-version owner is created.
func ContainerSelection(root string) (Selection, error) {
	c, err := ReadContainerContract(root)
	if err != nil {
		return Selection{}, err
	}
	head, err := Run(root, "git", "rev-parse", "HEAD")
	if err != nil {
		return Selection{}, err
	}
	s := Selection{Schema: "ptsip-temporary-container-selection/v1", SourceSHA: strings.TrimSpace(string(head)), Type: "TEMPORARY_CONTAINER", Tag: c.Tags[0], Platforms: []string{"linux-x64"}, Components: []Component{{ID: "container", Kind: "container", Version: c.Tags[0]}}}
	b, err := json.Marshal(s)
	if err != nil {
		return s, err
	}
	s.SelectionSHA = Digest(b)
	return s, nil
}

func ValidateContainerSelection(root string, s Selection, clean bool) error {
	want, err := ContainerSelection(root)
	if err != nil {
		return err
	}
	a, _ := json.Marshal(s)
	b, _ := json.Marshal(want)
	if string(a) != string(b) {
		return fmt.Errorf("container selection differs from exact recipe source")
	}
	branch := os.Getenv("GITHUB_REF_NAME")
	if branch == "" {
		out, e := Run(root, "git", "symbolic-ref", "--short", "HEAD")
		if e != nil {
			return e
		}
		branch = strings.TrimSpace(string(out))
	}
	if branch != "dev/0.3.8b2" {
		return fmt.Errorf("temporary container delivery is restricted to dev/0.3.8b2")
	}
	if clean {
		out, e := Run(root, "git", "status", "--porcelain", "--untracked-files=no")
		if e != nil || len(out) != 0 {
			return fmt.Errorf("tracked recipe source changed")
		}
	}
	return nil
}

func ValidateContainerManifest(c ContainerContract, m ShippingManifest) error {
	if m.Schema != "ptsip-shipping-manifest/v1" || m.Selection.SourceSHA != ContainerSource || m.Snapshot.SourceSHA != ContainerSource || m.VerificationRun != c.OriginalRun || m.Snapshot.Version != "0.3.8.f87bfe" || m.PackageDistribution == nil || m.PackageDistribution.Version != c.Tags[0] || m.PackageDistribution.SourceSHA != ContainerSource || len(m.PackageDistribution.Files) != 1 {
		return fmt.Errorf("frozen container source manifest differs")
	}
	if err := ValidateSnapshot(m.Snapshot); err != nil {
		return err
	}
	expected := map[string]string{}
	seen := map[string]bool{}
	for _, id := range c.Engines {
		expected[id] = ""
	}
	for _, component := range m.Snapshot.Components {
		if seen[component.ID] {
			return fmt.Errorf("duplicate frozen component")
		}
		seen[component.ID] = true
		if component.Kind == "engine" {
			if _, ok := expected[component.ID]; !ok {
				return fmt.Errorf("unapproved frozen engine")
			}
			expected[component.ID] = component.Version
		}
	}
	if len(m.Snapshot.Components) != 9 || !seen["tool"] || !seen["profiles"] {
		return fmt.Errorf("frozen Package must contain Tool, Profiles and seven engines")
	}
	for _, v := range expected {
		if v == "" {
			return fmt.Errorf("mandatory frozen engine missing")
		}
	}
	return CompletePackage(m.Snapshot, expected)
}

func FrozenContainerManifest(root string) (ContainerContract, ShippingManifest, []byte, error) {
	c, err := ReadContainerContract(root)
	if err != nil {
		return c, ShippingManifest{}, nil, err
	}
	var carrier Carrier
	if err := GitHub("repos/Kinirin/PTSIP/releases/tags/"+c.Release, &carrier); err != nil {
		return c, ShippingManifest{}, nil, err
	}
	if carrier.Draft || carrier.Tag != c.Release {
		return c, ShippingManifest{}, nil, fmt.Errorf("original source Release is not published")
	}
	url := ""
	for _, asset := range carrier.Assets {
		if asset.Name == "shipping-manifest.json" && asset.Digest == "sha256:"+c.ManifestSHA {
			if url != "" {
				return c, ShippingManifest{}, nil, fmt.Errorf("duplicate original manifest")
			}
			url = asset.URL
		}
	}
	if url == "" {
		return c, ShippingManifest{}, nil, fmt.Errorf("original manifest digest not observed")
	}
	b, err := DownloadArtifact(url, c.ManifestSHA)
	if err != nil {
		return c, ShippingManifest{}, nil, err
	}
	var m ShippingManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return c, m, nil, err
	}
	if err := ValidateContainerManifest(c, m); err != nil {
		return c, m, nil, err
	}
	var original WorkflowRun
	if err := GitHub(fmt.Sprintf("repos/Kinirin/PTSIP/actions/runs/%d", c.OriginalRun), &original); err != nil {
		return c, m, nil, err
	}
	if original.Head != ContainerSource || original.Path != ".github/workflows/tooling-test.yml" || original.Status != "completed" || original.Conclusion != "success" {
		return c, m, nil, fmt.Errorf("original seven-engine verification unavailable")
	}
	return c, m, b, nil
}

func ContainerWheels(c ContainerContract, m ShippingManifest) ([]File, error) {
	if err := ValidateContainerManifest(c, m); err != nil {
		return nil, err
	}
	files := append([]File{}, m.PackageDistribution.Files...)
	for _, component := range m.Snapshot.Components {
		if component.Kind != "engine" {
			continue
		}
		count := 0
		for _, f := range component.Files {
			if f.Platform == "linux-x64" && strings.HasSuffix(f.Name, ".whl") {
				files = append(files, f)
				count++
			}
		}
		if count != 1 {
			return nil, fmt.Errorf("one Linux native wheel required for %s", component.ID)
		}
	}
	if len(files) != 8 {
		return nil, fmt.Errorf("container requires Package wheel and seven native wheels")
	}
	admitted := map[string]string{}
	for _, f := range m.Assets {
		admitted[f.Name] = f.SHA256
	}
	for _, f := range files {
		if filepath.Base(f.Name) != f.Name || strings.ContainsAny(f.Name, `\/:`) || admitted[f.Name] != f.SHA256 || !strings.HasPrefix(f.URL, "https://github.com/Kinirin/PTSIP/releases/download/"+c.Release+"/") {
			return nil, fmt.Errorf("frozen container wheel is outside admitted Release")
		}
	}
	return files, nil
}

func PrepareContainer(root, output string) (ShippingManifest, error) {
	c, m, manifest, err := FrozenContainerManifest(root)
	if err != nil {
		return m, err
	}
	files, err := ContainerWheels(c, m)
	if err != nil {
		return m, err
	}
	out, err := SafePath(root, output)
	if err != nil {
		return m, err
	}
	if err := os.MkdirAll(filepath.Join(out, "wheels"), 0755); err != nil {
		return m, err
	}
	for _, f := range files {
		target := filepath.Join(out, "wheels", f.Name)
		data, e := os.ReadFile(target)
		if e != nil || Digest(data) != f.SHA256 {
			data, e = DownloadArtifact(f.URL, f.SHA256)
			if e != nil {
				return m, e
			}
			if e = os.WriteFile(target, data, 0644); e != nil {
				return m, e
			}
		}
	}
	if err := os.WriteFile(filepath.Join(out, "shipping-manifest.json"), manifest, 0644); err != nil {
		return m, err
	}
	text, err := os.ReadFile(filepath.Join(root, ".github/checks/release/container/Dockerfile"))
	if err != nil {
		return m, err
	}
	if err := os.WriteFile(filepath.Join(out, "Dockerfile"), []byte(strings.ReplaceAll(string(text), "\r\n", "\n")), 0644); err != nil {
		return m, err
	}
	return m, nil
}

type ContainerReceipt struct {
	Schema         string         `json:"schema_version"`
	Source         string         `json:"recipe_source_sha"`
	OriginalSource string         `json:"application_source_sha"`
	Selection      string         `json:"selection_sha256"`
	ImageID        string         `json:"image_id"`
	ImageTag       string         `json:"tested_image_tag"`
	Archive        File           `json:"archive"`
	ManifestSHA    string         `json:"shipping_manifest_sha256"`
	Status         string         `json:"status"`
	Runtime        map[string]any `json:"runtime_verification"`
}

func ValidateContainerReceipt(root string, s Selection, r ContainerReceipt, dir string) error {
	c, err := ReadContainerContract(root)
	if err != nil {
		return err
	}
	if r.Schema != "ptsip-temporary-container-test/v1" || r.Source != s.SourceSHA || r.OriginalSource != ContainerSource || r.Selection != s.SelectionSHA || r.ManifestSHA != c.ManifestSHA || r.Status != "PASS" || !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(r.ImageID) || r.ImageTag != "ptsip-container-test:"+s.SourceSHA || r.Archive.Name != "image.tar.gz" || r.Runtime["engine_count"] != float64(7) || r.Runtime["all_installed"] != true || r.Runtime["execution_authorized"] != false {
		return fmt.Errorf("container verification receipt differs")
	}
	b, err := os.ReadFile(filepath.Join(dir, r.Archive.Name))
	if err != nil || Digest(b) != r.Archive.SHA256 {
		return fmt.Errorf("verified container archive digest differs")
	}
	return nil
}
