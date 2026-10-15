package releasecontroller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func PublishContainer(root string) (map[string]any, error) {
	s, err := ContainerSelection(root)
	if err != nil {
		return nil, err
	}
	if err := ValidateContainerSelection(root, s, true); err != nil {
		return nil, err
	}
	c, err := ReadContainerContract(root)
	if err != nil {
		return nil, err
	}
	var current struct{ Object struct{ SHA string } }
	if err := GitHub("repos/Kinirin/PTSIP/git/ref/heads/"+c.Branch, &current); err != nil {
		return nil, err
	}
	if current.Object.SHA != s.SourceSHA {
		return nil, fmt.Errorf("remote branch changed before container publication")
	}
	run, err := FindVerification(s, GitHubReleaseAPI, func(run WorkflowRun) (Selection, error) {
		artifacts, e := RunArtifacts(run.ID)
		if e != nil {
			return Selection{}, e
		}
		for _, a := range artifacts {
			if a.Name == "verification-selection-"+s.SourceSHA {
				ref := fmt.Sprintf(".artifacts/release/container/evidence/%d/selection", run.ID)
				if e := DownloadActionArtifact(root, ref, a); e != nil {
					return Selection{}, e
				}
				var selected Selection
				e := ReadJSON(root, ref+"/selection.json", &selected)
				return selected, e
			}
		}
		return Selection{}, fmt.Errorf("container selection artifact absent")
	})
	if err != nil {
		return nil, err
	}
	artifacts, err := RunArtifacts(run)
	if err != nil {
		return nil, err
	}
	ref := fmt.Sprintf(".artifacts/release/container/evidence/%d/image", run)
	found := false
	for _, a := range artifacts {
		if a.Name == "verification-container-linux-x64-"+s.SourceSHA {
			if found {
				return nil, fmt.Errorf("duplicate container evidence")
			}
			found = true
			if err := DownloadActionArtifact(root, ref, a); err != nil {
				return nil, err
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("verified image archive absent")
	}
	var receipt ContainerReceipt
	if err := ReadJSON(root, ref+"/container-receipt.json", &receipt); err != nil {
		return nil, err
	}
	dir, err := SafePath(root, ref)
	if err != nil {
		return nil, err
	}
	if err := ValidateContainerReceipt(root, s, receipt, dir); err != nil {
		return nil, err
	}
	if _, err := Run(root, "docker", "load", "--input", filepath.Join(dir, receipt.Archive.Name)); err != nil {
		return nil, err
	}
	image, err := Run(root, "docker", "image", "inspect", receipt.ImageTag, "--format", "{{.Id}}")
	if err != nil || strings.TrimSpace(string(image)) != receipt.ImageID {
		return nil, fmt.Errorf("loaded image differs from tested image")
	}
	// Existing tags are immutable. Inspect package metadata with the workflow's
	// package-scoped token; do not infer absence from an arbitrary API failure.
	data, e := Run(root, "gh", "api", "users/Kinirin/packages/container/ptsip/versions?per_page=100")
	if e != nil && !strings.Contains(e.Error(), "HTTP 404") {
		return nil, e
	}
	if e == nil {
		var versions []struct {
			Metadata struct{ Container struct{ Tags []string } }
		}
		if err := json.Unmarshal(data, &versions); err != nil {
			return nil, err
		}
		for _, v := range versions {
			for _, old := range v.Metadata.Container.Tags {
				for _, tag := range c.Tags {
					if old == tag {
						if _, err := InspectContainerManifest(root, c.Image+":"+tag, receipt.ImageID); err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}
	for _, tag := range c.Tags {
		target := c.Image + ":" + tag
		if _, err := Run(root, "docker", "tag", receipt.ImageID, target); err != nil {
			return nil, err
		}
		if _, err := Run(root, "docker", "push", target); err != nil {
			return nil, err
		}
		if _, err := InspectContainerManifest(root, target, receipt.ImageID); err != nil {
			return nil, err
		}
	}
	bytes, err := Run(root, "docker", "image", "inspect", c.Image+":"+c.Tags[0], "--format", "{{json .RepoDigests}}")
	if err != nil {
		return nil, err
	}
	var digests []string
	if err := json.Unmarshal(bytes, &digests); err != nil {
		return nil, err
	}
	digest := ""
	for _, d := range digests {
		if strings.HasPrefix(d, c.Image+"@sha256:") {
			digest = d
		}
	}
	if digest == "" {
		return nil, fmt.Errorf("published container digest unavailable")
	}
	_, m, _, err := FrozenContainerManifest(root)
	if err != nil {
		return nil, err
	}
	result := map[string]any{"schema_version": "ptsip-temporary-container-publication/v1", "recipe_source_sha": s.SourceSHA, "application_source_sha": ContainerSource, "original_verification_run": c.OriginalRun, "container_verification_run": run, "image": c.Image, "tags": c.Tags, "image_id": receipt.ImageID, "digest_reference": digest, "package_identity": m.Snapshot.Version, "package_distribution_version": m.PackageDistribution.Version, "pypi_publication": false, "successor_info_implementation_cutover": false, "automated_policy_validation_available": false, "runtime_activation": false}
	note := ".artifacts/release/container/container-publication-" + s.SourceSHA + ".json"
	if err := WriteJSON(filepath.Join(root, filepath.FromSlash(note)), result); err != nil {
		return nil, err
	}
	var carrier Carrier
	if err := GitHub("repos/Kinirin/PTSIP/releases/tags/"+c.Release, &carrier); err != nil {
		return nil, err
	}
	name := filepath.Base(note)
	already := false
	for _, asset := range carrier.Assets {
		if asset.Name == name {
			b, e := os.ReadFile(filepath.Join(root, filepath.FromSlash(note)))
			if e != nil || asset.Digest != "sha256:"+Digest(b) {
				return nil, fmt.Errorf("existing container publication record differs")
			}
			already = true
		}
	}
	if !already {
		if _, err := Run(root, "gh", "release", "upload", c.Release, note, "--repo", "Kinirin/PTSIP"); err != nil {
			return nil, err
		}
	}
	if os.Getenv("GITHUB_STEP_SUMMARY") != "" {
		file, e := os.OpenFile(os.Getenv("GITHUB_STEP_SUMMARY"), os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(file, "Published temporary PTSIP %s container: `%s`\n\nApplication source: `%s`; recipe source: `%s`; verification run: %d.\n", m.PackageDistribution.Version, digest, ContainerSource, s.SourceSHA, run)
		file.Close()
	}
	return result, nil
}

func InspectContainerManifest(root, target, imageID string) (string, error) {
	b, err := Run(root, "docker", "manifest", "inspect", target)
	if err != nil {
		return "", err
	}
	var m struct {
		Config    struct{ Digest string }
		Manifests []struct {
			Digest   string
			Platform struct {
				OS           string
				Architecture string
			}
		}
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}
	if len(m.Manifests) != 0 {
		count := 0
		selected := ""
		for _, entry := range m.Manifests {
			if entry.Platform.OS == "linux" && entry.Platform.Architecture == "amd64" {
				count++
				selected = entry.Digest
			}
		}
		if count != 1 {
			return "", fmt.Errorf("one Linux amd64 image required in published index")
		}
		base := strings.SplitN(strings.SplitN(target, "@", 2)[0], ":", 2)[0]
		return InspectContainerManifest(root, base+"@"+selected, imageID)
	}
	if m.Config.Digest != imageID {
		return "", fmt.Errorf("container tag %s differs from the verified image; overwrite refused", target)
	}
	return m.Config.Digest, nil
}
