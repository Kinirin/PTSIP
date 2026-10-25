package releasecontroller

import (
	"fmt"
	"os"
	"path/filepath"
)

type PackageReceipt struct {
	Schema    string   `json:"schema_version"`
	Source    string   `json:"source_sha"`
	Selection string   `json:"selection_sha256"`
	Platform  string   `json:"platform"`
	PyPI      string   `json:"pypi_version"`
	Snapshot  Snapshot `json:"package"`
	Wheel     File     `json:"wheel"`
	SourceDist *File   `json:"source_distribution,omitempty"`
	Status    string   `json:"status"`
}

// sourceBoundVerificationSnapshot is never a published registry projection.
// Every component must be present and tied to the selected source SHA.
func sourceBoundVerificationSnapshot(s Selection, components []PublishedComponent, engines map[string]string) (Snapshot, error) {
	if s.Type != "PTSIP_PACKAGE" || s.SourceSHA == "" || s.PackageBase == "" ||
		len(s.Components) != len(components) {
		return Snapshot{}, fmt.Errorf("source-bound Package selection is incomplete")
	}
	selected := map[string]Component{}
	for _, c := range s.Components {
		if c.ID == "" || selected[c.ID].ID != "" {
			return Snapshot{}, fmt.Errorf("duplicate or empty source-bound component identity")
		}
		selected[c.ID] = c
	}
	seen := map[string]bool{}
	for _, c := range components {
		entry, ok := selected[c.ID]
		if !ok || seen[c.ID] || entry.Kind != c.Kind || entry.Version != c.Version ||
			c.SourceSHA != s.SourceSHA || len(c.Files) == 0 {
			return Snapshot{}, fmt.Errorf("source-bound component evidence differs: %s", c.ID)
		}
		seen[c.ID] = true
	}
	// Deliberately do not load the published registry. A test receipt is not
	// a license to reissue an immutable published version of this Package.
	registry := Registry{NumericBase: s.PackageBase}
	snapshot, err := RefreshPackage(&registry, s.SourceSHA, components)
	if err != nil {
		return Snapshot{}, err
	}
	if err := CompletePackage(snapshot, engines); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// BuildSourceVerificationPackageReceipt consumes only the exact-source receipts
// admitted by Gather. This is a non-publishing test Join: a changed artifact
// producer must be tested even when the public version has not been changed.
//
// Published artifact reuse and immutable version admission remain mandatory in
// PrepareShipping, the separate production/publishing path.
func BuildSourceVerificationPackageReceipt(root string, s Selection, input, output, platform string) (PackageReceipt, error) {
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
	// The test-only receipt must not resolve immutable published assets or
	// treat historical component versions as test admission authority.
	// Gather already authenticates every source/selection/host/file digest.
	snapshot, err := sourceBoundVerificationSnapshot(s, components, c.Engines)
	if err != nil {
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
				if err == nil {
					native.Wheels[component.ID] = b
				}
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
