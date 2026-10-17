package releasecontroller

import (
	"fmt"
	"os"
	"path/filepath"
)

type PackageReceipt struct {
	Schema     string   `json:"schema_version"`
	Source     string   `json:"source_sha"`
	Selection  string   `json:"selection_sha256"`
	Platform   string   `json:"platform"`
	PyPI       string   `json:"pypi_version"`
	Snapshot   Snapshot `json:"package"`
	Wheel      File     `json:"wheel"`
	SourceDist *File    `json:"source_distribution,omitempty"`
	Status     string   `json:"status"`
}

func BuildPackageReceipt(root string, s Selection, input, output, platform string) (PackageReceipt, error) {
	c, err := ReadComposition(root)
	if err != nil {
		return PackageReceipt{}, err
	}
	if c.PackagePyPI == "" {
		return PackageReceipt{}, fmt.Errorf("complete pip Package requires current composition v2")
	}
	components, locations, err := Gather(root, s, input)
	if err != nil {
		return PackageReceipt{}, err
	}
	components, err = EngineReuse(root, s, components, locations)
	if err != nil {
		return PackageReceipt{}, err
	}
	registry, err := PublishedRegistry(root, ".github/checks/release/package-registry.json", s.SourceSHA)
	if err != nil {
		return PackageReceipt{}, err
	}
	components, err = ReusePublishedComponents(root, s, registry, components, locations)
	if err != nil {
		return PackageReceipt{}, err
	}
	registry.NumericBase = c.PackageBase
	snapshot, err := RefreshPackage(&registry, s.SourceSHA, components)
	if err != nil {
		return PackageReceipt{}, err
	}
	if err := CompletePackage(snapshot, c.Engines); err != nil {
		return PackageReceipt{}, err
	}
	out, err := SafePath(root, output)
	if err != nil {
		return PackageReceipt{}, err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return PackageReceipt{}, err
	}
	var tool, profiles []byte
	native := PackageNativePayload{Platform: platform, Wheels: map[string][]byte{}}
	for _, component := range snapshot.Components {
		for _, file := range component.Files {
			if component.Kind == "tool" && filepath.Ext(file.Name) == ".whl" {
				tool, err = os.ReadFile(locations["tool/linux-x64/"+file.Name])
			}
			if component.Kind == "profiles" && filepath.Ext(file.Name) == ".zip" {
				profiles, err = os.ReadFile(locations["profiles/linux-x64/"+file.Name])
			}
			if component.Kind == "engine" && file.Platform == platform && filepath.Ext(file.Name) == ".whl" {
				var b []byte
				b, err = os.ReadFile(locations[component.ID+"/"+platform+"/"+file.Name])
				native.Wheels[component.ID] = b
			}
			if err != nil {
				return PackageReceipt{}, err
			}
		}
	}
	wheel, err := BundledPackageWheel(tool, profiles, snapshot, c.Engines, c.PackagePyPI, native)
	if err != nil {
		return PackageReceipt{}, err
	}
	name, err := BundledPackageWheelName(c.PackagePyPI, platform)
	if err != nil {
		return PackageReceipt{}, err
	}
	if err := os.WriteFile(filepath.Join(out, name), wheel, 0644); err != nil {
		return PackageReceipt{}, err
	}
	return PackageReceipt{Schema: "ptsip-pip-package-test-receipt/v2", Source: s.SourceSHA, Selection: s.SelectionSHA, Platform: platform, PyPI: c.PackagePyPI, Snapshot: snapshot, Wheel: File{Name: name, SHA256: Digest(wheel), Platform: platform}, Status: "AWAITING_INSTALL_VERIFICATION"}, nil
}
