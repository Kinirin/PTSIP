package releasecontroller

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type WorkflowRun struct {
	ID                       int64
	Head                     string `json:"head_sha"`
	Path, Status, Conclusion string
}

// FindVerification chooses the newest successful run whose exact selection was
// emitted by the registered verification workflow. It never promotes a nearby SHA.
func FindVerification(s Selection, api ReleaseAPI, selectionFor func(WorkflowRun) (Selection, error)) (int64, error) {
	for page := 1; page <= 3; page++ {
		var result struct {
			Runs []WorkflowRun `json:"workflow_runs"`
		}
		endpoint := fmt.Sprintf("repos/Kinirin/PTSIP/actions/workflows/tooling-test.yml/runs?head_sha=%s&status=success&per_page=100&page=%d", s.SourceSHA, page)
		if err := api("GET", endpoint, nil, &result); err != nil {
			return 0, err
		}
		for _, run := range result.Runs {
			if run.Head != s.SourceSHA || run.Path != ".github/workflows/tooling-test.yml" || run.Status != "completed" || run.Conclusion != "success" {
				continue
			}
			candidate, err := selectionFor(run)
			if err != nil {
				continue
			}
			a, _ := json.Marshal(s)
			b, _ := json.Marshal(candidate)
			if bytes.Equal(a, b) {
				return run.ID, nil
			}
		}
		if len(result.Runs) < 100 {
			break
		}
	}
	return 0, fmt.Errorf("no successful exact-source tooling-test run for this selection; dispatch tooling-test with the same targets and optional platforms")
}

type ActionArtifact struct {
	ID           int64
	Name, Digest string
	Expired      bool
}

func RunArtifacts(run int64) ([]ActionArtifact, error) {
	all := []ActionArtifact{}
	for page := 1; ; page++ {
		var result struct{ Artifacts []ActionArtifact }
		if err := GitHub(fmt.Sprintf("repos/Kinirin/PTSIP/actions/runs/%d/artifacts?per_page=100&page=%d", run, page), &result); err != nil {
			return nil, err
		}
		all = append(all, result.Artifacts...)
		if len(result.Artifacts) < 100 {
			return all, nil
		}
	}
}

// The API archive digest authenticates the entire download before extracting it.
func DownloadActionArtifact(root, destination string, a ActionArtifact) error {
	if a.Expired || a.ID <= 0 || !strings.HasPrefix(a.Digest, "sha256:") {
		return fmt.Errorf("original Actions artifact unavailable")
	}
	b, err := Run(root, "gh", "api", fmt.Sprintf("repos/Kinirin/PTSIP/actions/artifacts/%d/zip", a.ID))
	if err != nil {
		return err
	}
	if Digest(b) != strings.TrimPrefix(a.Digest, "sha256:") {
		return fmt.Errorf("Actions archive digest differs: %s", a.Name)
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return err
	}
	dir, err := SafePath(root, destination)
	if err != nil {
		return err
	}
	var total uint64
	for _, f := range z.File {
		ref := strings.TrimSuffix(f.Name, "/")
		if ref == "" {
			continue
		}
		p, err := SafePath(dir, ref)
		if err != nil || f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe Actions artifact entry")
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(p, 0755); err != nil {
				return err
			}
			continue
		}
		total += f.UncompressedSize64
		if f.UncompressedSize64 > 128*1024*1024 || total > 512*1024*1024 {
			return fmt.Errorf("Actions artifact exceeds extraction limit")
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(r, 128*1024*1024+1))
		r.Close()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func PreparePublication(root, targets, optional, output string) (ShippingManifest, error) {
	s, err := Select(root, targets, optional)
	if err != nil {
		return ShippingManifest{}, err
	}
	if err := AdmitBeta2(root, s); err != nil {
		return ShippingManifest{}, err
	}
	var selectionDir string
	run, err := FindVerification(s, GitHubReleaseAPI, func(run WorkflowRun) (Selection, error) {
		artifacts, err := RunArtifacts(run.ID)
		if err != nil {
			return Selection{}, err
		}
		for _, a := range artifacts {
			if a.Name == "verification-selection-"+s.SourceSHA {
				selectionDir = ".artifacts/release/evidence/" + strconv.FormatInt(run.ID, 10) + "/selection"
				if err := DownloadActionArtifact(root, selectionDir, a); err != nil {
					return Selection{}, err
				}
				var candidate Selection
				err := ReadJSON(root, selectionDir+"/selection.json", &candidate)
				return candidate, err
			}
		}
		return Selection{}, fmt.Errorf("selection artifact missing")
	})
	if err != nil {
		return ShippingManifest{}, err
	}
	if err := AdmitVerification(root, s, run); err != nil {
		return ShippingManifest{}, err
	}
	artifacts, err := RunArtifacts(run)
	if err != nil {
		return ShippingManifest{}, err
	}
	for _, a := range artifacts {
		if strings.HasPrefix(a.Name, "verification-") && a.Name != "verification-selection-"+s.SourceSHA && strings.HasSuffix(a.Name, s.SourceSHA) {
			if err := DownloadActionArtifact(root, ".artifacts/release/evidence/"+strconv.FormatInt(run, 10)+"/verified/"+a.Name, a); err != nil {
				return ShippingManifest{}, err
			}
		}
	}
	if _, err := PrepareShipping(root, s, run, ".artifacts/release/evidence/"+strconv.FormatInt(run, 10)+"/verified", output); err != nil {
		return ShippingManifest{}, err
	}
	if err := WriteJSON(filepath.Join(root, ".artifacts/release/selection.json"), s); err != nil {
		return ShippingManifest{}, err
	}
	return Shipping(root, s, output)
}
