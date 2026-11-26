package releasecontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
)

// ObservePublishedPackageWheel reads a Package wheel after checking its live
// publication. The verified shipping manifest and exact GitHub source tag
// bind the wheel to the source revision. For PyPI, the public file hash must
// independently match the carrier wheel. This has no publication side effect.
func ObservePublishedPackageWheel(ctx context.Context, platform, reference, version, sourceSHA, name, digest string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(sourceSHA) ||
		!regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(digest) ||
		name != path.Base(name) || !strings.HasSuffix(name, ".whl") || strings.ContainsAny(name, "\\:") {
		return nil, fmt.Errorf("invalid published Package wheel coordinates")
	}
	tag := "ptsip-release-" + sourceSHA
	switch platform {
	case "github":
		if reference != tag {
			return nil, fmt.Errorf("published Package carrier reference differs")
		}
	case "pypi":
		if reference != "PTSIP/"+version {
			return nil, fmt.Errorf("published PyPI Package reference differs")
		}
	default:
		return nil, fmt.Errorf("unregistered Package publication platform")
	}
	const repository = "repos/Kinirin/PTSIP/"
	var carrier Carrier
	if err := GitHub(repository+"releases/tags/"+tag, &carrier); err != nil {
		return nil, err
	}
	if _, err := time.Parse(time.RFC3339, carrier.Published); err != nil || carrier.Draft || carrier.Tag != tag || carrier.ID <= 0 {
		return nil, fmt.Errorf("Package carrier publication not observed")
	}
	var ref struct {
		Object struct{ SHA, Type string }
	}
	if err := GitHub(repository+"git/ref/tags/"+tag, &ref); err != nil || ref.Object.SHA != sourceSHA || ref.Object.Type != "commit" {
		return nil, fmt.Errorf("published Package source tag does not identify the expected commit")
	}
	manifestAsset, err := uniquePublishedAsset(carrier, "shipping-manifest.json")
	if err != nil {
		return nil, err
	}
	raw, err := DownloadArtifact(manifestAsset.URL, strings.TrimPrefix(manifestAsset.Digest, "sha256:"))
	if err != nil {
		return nil, err
	}
	var m ShippingManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	d := m.PackageDistribution
	if m.Schema != "ptsip-shipping-manifest/v1" || m.Selection.SourceSHA != sourceSHA || m.Selection.Tag != tag ||
		m.Selection.Type != "PTSIP_PACKAGE" || m.VerificationRun <= 0 || d == nil ||
		d.ID != "package" || d.Kind != "package" || d.SourceSHA != sourceSHA || d.Version != version ||
		ValidateSnapshot(m.Snapshot) != nil || m.Snapshot.SourceSHA != sourceSHA ||
		!uniquePublishedDigest(d.Files, name, digest) || !uniquePublishedDigest(m.Assets, name, digest) {
		return nil, fmt.Errorf("published Package wheel is not bound to the expected shipping source")
	}
	wheel, err := uniquePublishedAsset(carrier, name)
	if err != nil || wheel.Digest != "sha256:"+digest {
		return nil, fmt.Errorf("published carrier wheel digest differs")
	}
	if platform == "pypi" {
		files, err := PyPIFiles("PTSIP", version)
		if err != nil {
			return nil, err
		}
		observed := []File{}
		for _, file := range files {
			observed = append(observed, File{Name: file.Name, SHA256: file.Digests.SHA256})
		}
		if !uniquePublishedDigest(observed, name, digest) {
			return nil, fmt.Errorf("Package wheel publication not observed on PyPI")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return DownloadArtifact(wheel.URL, digest)
}

func uniquePublishedAsset(carrier Carrier, name string) (Asset, error) {
	var found Asset
	for _, asset := range carrier.Assets {
		if asset.Name != name {
			continue
		}
		if found.Name != "" {
			return Asset{}, fmt.Errorf("duplicate published asset: %s", name)
		}
		found = asset
	}
	if !strings.HasPrefix(found.Digest, "sha256:") ||
		!regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(strings.TrimPrefix(found.Digest, "sha256:")) {
		return Asset{}, fmt.Errorf("published asset has no verified digest: %s", name)
	}
	return found, nil
}

func uniquePublishedDigest(files []File, name, digest string) bool {
	matches := 0
	for _, file := range files {
		if file.Name == name {
			if file.SHA256 != digest {
				return false
			}
			matches++
		}
	}
	return matches == 1
}
