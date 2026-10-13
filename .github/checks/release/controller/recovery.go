package releasecontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const EvidenceContractPath = ".github/checks/release/publication-evidence.json"

func SameReleaseRequest(a, b Selection) bool {
	x, _ := json.Marshal(struct {
		Type, Base string
		Components []Component
		Platforms  []string
	}{a.Type, a.PackageBase, a.Components, a.Platforms})
	y, _ := json.Marshal(struct {
		Type, Base string
		Components []Component
		Platforms  []string
	}{b.Type, b.PackageBase, b.Components, b.Platforms})
	return bytes.Equal(x, y)
}

func ReissueSourceChanges(root, original, current string) (string, error) {
	var contract struct {
		Schema   string `json:"schema_version"`
		Recovery struct {
			Boundaries []string `json:"source_boundaries"`
		} `json:"deleted_carrier_reissue"`
	}
	if err := ReadJSON(root, EvidenceContractPath, &contract); err != nil {
		return "", err
	}
	if contract.Schema != "ptsip-publication-evidence/v1" || len(contract.Recovery.Boundaries) == 0 {
		return "", fmt.Errorf("publication evidence contract unresolved")
	}
	args := []string{"git", "diff", "--name-only", original, current, "--"}
	for _, ref := range contract.Recovery.Boundaries {
		if _, err := SafePath(root, ref); err != nil {
			return "", err
		}
		args = append(args, ref)
	}
	delta, err := Run(root, args...)
	if err != nil || len(bytes.TrimSpace(delta)) != 0 {
		return strings.TrimSpace(string(delta)), err
	}
	// Only the observed-publication table may differ in the root declaration.
	canonical := func(source string) ([]byte, error) {
		data, err := Run(root, "git", "show", source+":pyproject.toml")
		if err != nil {
			return nil, err
		}
		var project map[string]any
		if err := toml.Unmarshal(data, &project); err != nil {
			return nil, err
		}
		if tool, ok := project["tool"].(map[string]any); ok {
			if ptsip, ok := tool["ptsip"].(map[string]any); ok {
				if release, ok := ptsip["release"].(map[string]any); ok {
					delete(release, "published")
				}
			}
		}
		return json.Marshal(project)
	}
	a, err := canonical(original)
	if err != nil {
		return "", err
	}
	b, err := canonical(current)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(a, b) {
		return "pyproject.toml", nil
	}
	return "", nil
}

func OriginalPublicationPassed(run int64, source string) error {
	var r WorkflowRun
	if err := GitHub(fmt.Sprintf("repos/Kinirin/PTSIP/actions/runs/%d", run), &r); err != nil {
		return err
	}
	if r.Head != source || r.Path != ".github/workflows/tooling-release.yml" || r.Status != "completed" {
		return fmt.Errorf("original publication run source differs")
	}
	var result struct {
		Jobs []struct{ Name, Status, Conclusion string }
	}
	if err := GitHub(fmt.Sprintf("repos/Kinirin/PTSIP/actions/runs/%d/jobs?per_page=100", run), &result); err != nil {
		return err
	}
	for _, job := range result.Jobs {
		if job.Name == "github-publication" && job.Status == "completed" && job.Conclusion == "success" {
			return nil
		}
	}
	return fmt.Errorf("original GitHub publication was not observed")
}

func OriginalVerification(root string, s Selection, run int64) error {
	dir, err := os.MkdirTemp("", "ptsip-original-admission-")
	if err != nil {
		return err
	}
	// Git creates a detached, read-only-by-convention checkout for original evidence.
	if _, err := Run(root, "git", "-c", "core.autocrlf=false", "worktree", "add", "--detach", dir, s.SourceSHA); err != nil {
		os.Remove(dir)
		return err
	}
	defer Run(root, "git", "worktree", "remove", dir)
	return AdmitVerification(dir, s, run)
}

func AdmitShipping(root string, m ShippingManifest) error {
	policyData, err := Run(root, "go", "-C", "developer/automation", "run", "-tags", "grammar_subset,grammar_subset_python", "./cmd/ptsip-dev", "policy-resolver", "get", "MPD-CTRL-0001", "--section", "release_orchestration")
	if err != nil {
		return err
	}
	var policy struct {
		Record map[string]any `json:"record"`
	}
	if json.Unmarshal(policyData, &policy) != nil || policy.Record["evidence_resolution_contract_ref"] != EvidenceContractPath {
		return fmt.Errorf("current publication evidence authority unresolved")
	}
	tracked, err := Run(root, "git", "status", "--porcelain", "--untracked-files=no")
	if err != nil || len(tracked) != 0 {
		return fmt.Errorf("tracked publisher source changed before publication")
	}
	if m.ReissueRun == 0 {
		return AdmitVerification(root, m.Selection, m.VerificationRun)
	}
	head, err := Run(root, "git", "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != m.PublicationSource {
		return fmt.Errorf("reissue publisher source differs")
	}
	current, err := Select(root, selectionTargets(m.Selection), selectionOptional(m.Selection))
	if err != nil || !SameReleaseRequest(current, m.Selection) {
		return fmt.Errorf("reissue selection differs from current canonical registration")
	}
	delta, err := ReissueSourceChanges(root, m.Selection.SourceSHA, m.PublicationSource)
	if err != nil {
		return err
	}
	if delta != "" {
		return fmt.Errorf("reissue artifact producer changed: %s", delta)
	}
	if err := OriginalPublicationPassed(m.ReissueRun, m.Selection.SourceSHA); err != nil {
		return err
	}
	artifacts, err := RunArtifacts(m.ReissueRun)
	if err != nil {
		return err
	}
	valid := false
	for _, a := range artifacts {
		valid = valid || (!a.Expired && a.ID == m.OriginalArtifactID && a.Digest == m.OriginalArtifactDigest && a.Name == "admitted-publication-"+m.Selection.SourceSHA)
	}
	if !valid {
		return fmt.Errorf("original admitted artifact provenance differs")
	}
	return OriginalVerification(root, m.Selection, m.VerificationRun)
}

func selectionTargets(s Selection) string {
	ids := []string{}
	for _, c := range s.Components {
		if c.Kind == "engine" {
			ids = append(ids, "engine:"+c.ID)
		} else {
			ids = append(ids, c.ID)
		}
	}
	return strings.Join(ids, ",")
}
func selectionOptional(s Selection) string {
	platforms := []string{}
	for _, p := range s.Platforms {
		if p != "windows-x64" && p != "linux-x64" {
			platforms = append(platforms, p)
		}
	}
	return strings.Join(platforms, ",")
}

func RecoverDeletedPublication(root string, current Selection, output string) (ShippingManifest, bool, error) {
	var recent struct {
		Runs []WorkflowRun `json:"workflow_runs"`
	}
	if err := GitHub("repos/Kinirin/PTSIP/actions/workflows/tooling-release.yml/runs?status=completed&per_page=30", &recent); err != nil {
		return ShippingManifest{}, false, err
	}
	carriers, err := Carriers()
	if err != nil {
		return ShippingManifest{}, false, err
	}
	for _, run := range recent.Runs {
		if run.Path != ".github/workflows/tooling-release.yml" {
			continue
		}
		exists := false
		for _, r := range carriers {
			exists = exists || (!r.Draft && r.Tag == "ptsip-release-"+run.Head)
		}
		if exists {
			continue
		}
		if err := OriginalPublicationPassed(run.ID, run.Head); err != nil {
			continue
		}
		delta, err := ReissueSourceChanges(root, run.Head, current.SourceSHA)
		if err != nil {
			return ShippingManifest{}, false, err
		}
		if delta != "" {
			continue
		}
		artifacts, err := RunArtifacts(run.ID)
		if err != nil {
			return ShippingManifest{}, false, err
		}
		for _, a := range artifacts {
			if a.Expired || a.Name != "admitted-publication-"+run.Head {
				continue
			}
			dir := ".artifacts/release/recovery/" + strconv.FormatInt(run.ID, 10)
			if err := DownloadActionArtifact(root, dir, a); err != nil {
				return ShippingManifest{}, false, err
			}
			var m ShippingManifest
			if err := ReadJSON(root, dir+"/shipping-manifest.json", &m); err != nil {
				return m, false, err
			}
			if !SameReleaseRequest(current, m.Selection) || m.Selection.SourceSHA != run.Head {
				continue
			}
			if _, err := Shipping(root, m.Selection, dir); err != nil {
				return m, false, err
			}
			m.PublicationSource, m.ReissueRun, m.OriginalArtifactID, m.OriginalArtifactDigest = current.SourceSHA, run.ID, a.ID, a.Digest
			if err := AdmitShipping(root, m); err != nil {
				return m, false, err
			}
			from, err := SafePath(root, dir)
			if err != nil {
				return m, false, err
			}
			to, err := SafePath(root, output)
			if err != nil {
				return m, false, err
			}
			if err := os.CopyFS(to, os.DirFS(from)); err != nil {
				return m, false, err
			}
			if err := WritePresentation(root, &m, output); err != nil {
				return m, false, err
			}
			if err := WriteJSON(filepath.Join(root, ".artifacts/release/selection.json"), m.Selection); err != nil {
				return m, false, err
			}
			return m, true, nil
		}
	}
	return ShippingManifest{}, false, nil
}

func WritePresentation(root string, m *ShippingManifest, output string) error {
	_, body, err := ReleasePresentation(root, m.Selection, m.Snapshot)
	if err != nil {
		return err
	}
	for _, name := range []string{"release-body.md", m.Snapshot.Version + ".md"} {
		if err := os.WriteFile(filepath.Join(root, output, name), []byte(body), 0644); err != nil {
			return err
		}
		for i := range m.Assets {
			if m.Assets[i].Name == name {
				m.Assets[i].SHA256 = Digest([]byte(body))
			}
		}
	}
	return WriteJSON(filepath.Join(root, output, "shipping-manifest.json"), m)
}
