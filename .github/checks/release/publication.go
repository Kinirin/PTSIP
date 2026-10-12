package release

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

func PublishComponent(repo *machine.Repository, selection, id, output, repository, apiRoot string) (machine.Object, error) {
	if dirty, err := requireRun(repo, "git", "status", "--porcelain", "--untracked-files=no"); err != nil || dirty != "" {
		return nil, fmt.Errorf("publication requires a clean exact-source checkout")
	}
	s, err := readSelection(repo, selection)
	if err != nil {
		return nil, err
	}
	c, err := selectedComponent(s, id)
	if err != nil {
		return nil, err
	}
	if c.State != "BUILD" {
		return nil, fmt.Errorf("publication requires a selected changed component")
	}
	dir, err := repo.Path(output)
	if err != nil {
		return nil, err
	}
	receipts, err := VerifyReceipts(s, id, dir, true)
	if err != nil {
		return nil, err
	}
	client, err := newGitHubClient(repository, apiRoot)
	if err != nil {
		return nil, err
	}
	if err = verifyBuildEvidence(client, s, c, receipts, os.Getenv("GITHUB_RUN_ID")); err != nil {
		return nil, err
	}
	if _, err = ReconfirmMain(repo, s.SourceSHA); err != nil {
		return nil, err
	}
	if err = publishFiles(repo, repository, apiRoot, c.Tag, c.SourceSHA, c.ID+" "+c.Version, c.Version, dir, ""); err != nil {
		return nil, err
	}
	return machine.Object{"status": "PASS", "component": id, "tag": c.Tag, "source_sha": c.SourceSHA}, nil
}

// A PASS string in a local receipt never supplies publication admission.
// The exact workflow source and each required build job must be observed at
// the GitHub API for the current run. This does not reinstate common CI.
func verifyBuildEvidence(client *githubClient, s Selection, c Component, receipts []Receipt, runID string) error {
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(runID) {
		return fmt.Errorf("publication requires an observed release workflow run; use run_release.ps1 -Action Dispatch")
	}
	var run struct {
		SHA   string `json:"head_sha"`
		Path  string `json:"path"`
		Event string `json:"event"`
	}
	if err := client.request("GET", "/repos/"+client.repository+"/actions/runs/"+runID, nil, &run); err != nil {
		return err
	}
	if run.SHA != s.SourceSHA || strings.Split(run.Path, "@")[0] != ".github/workflows/tooling-release.yml" || (run.Event != "workflow_dispatch" && run.Event != "release") {
		return fmt.Errorf("release evidence workflow or source mismatch")
	}
	var jobs struct {
		Jobs []struct {
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
		} `json:"jobs"`
	}
	if err := client.request("GET", "/repos/"+client.repository+"/actions/runs/"+runID+"/jobs?per_page=100", nil, &jobs); err != nil {
		return err
	}
	for _, receipt := range receipts {
		if receipt.RunID != runID {
			return fmt.Errorf("receipt belongs to another or unobserved workflow run")
		}
		name := "build-" + c.ID + "-" + receipt.Platform
		found := false
		for _, job := range jobs.Jobs {
			if job.Name == name && job.Status == "completed" && job.Conclusion == "success" {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("required exact-source build did not succeed: %s", name)
		}
	}
	return nil
}

func publishFiles(repo *machine.Repository, repository, apiRoot, tag, sha, title, version, dir, composition string) error {
	client, err := newGitHubClient(repository, apiRoot)
	if err != nil {
		return err
	}
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	err = client.request("GET", "/repos/"+repository+"/git/ref/tags/"+tag, nil, &ref)
	if err == nil && ref.Object.SHA != sha {
		return fmt.Errorf("existing tag points at another source; tag mutation is forbidden")
	}
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return err
		}
		if err = client.request("POST", "/repos/"+repository+"/git/refs", map[string]string{"ref": "refs/tags/" + tag, "sha": sha}, &ref); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	assets := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			assets = append(assets, filepath.Join(dir, e.Name()))
		}
	}
	if len(assets) == 0 {
		return fmt.Errorf("empty publication artifact set")
	}
	var existing struct {
		ID     int64 `json:"id"`
		Draft  bool  `json:"draft"`
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	err = client.request("GET", "/repos/"+repository+"/releases/tags/"+tag, nil, &existing)
	if err == nil {
		for _, p := range assets {
			a, e := digest(p)
			if e != nil {
				return e
			}
			found := false
			for _, old := range existing.Assets {
				if old.Name == a.Name && old.Digest == "sha256:"+a.SHA256 {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("existing release asset differs or is missing; immutable publication cannot be overwritten: %s", a.Name)
			}
		}
		if !existing.Draft {
			return nil
		}
		return RunCommand(repo, "gh", "release", "edit", tag, "--repo", repository, "--draft=false")
	}
	if !strings.Contains(err.Error(), "404") {
		return err
	}
	notes := fmt.Sprintf("# %s\n\nSource: `%s`\n\nIndependent component publication. Artifact hashes and platform/contract identities are supplied by the attached release receipts.\n", title, sha)
	if composition != "" {
		notes += "\nComposition SHA-256: `" + composition + "`\n\nThe six-character Package suffix is an opaque mapping identifier, without ordering or compatibility semantics.\n"
	}
	note := filepath.Join(dir, "publication-notes.md")
	if err = os.WriteFile(note, []byte(notes), 0644); err != nil {
		return err
	}
	defer os.Remove(note)
	args := []string{"release", "create", tag, "--repo", repository, "--verify-tag", "--title", title, "--notes-file", note, "--draft"}
	if regexp.MustCompile(`[ab]|rc`).MatchString(version) && composition == "" {
		args = append(args, "--prerelease")
	}
	args = append(args, assets...)
	if err = RunCommand(repo, "gh", args...); err != nil {
		return err
	}
	if err = client.request("GET", "/repos/"+repository+"/releases/tags/"+tag, nil, &existing); err != nil {
		return err
	}
	for _, p := range assets {
		a, e := digest(p)
		if e != nil {
			return e
		}
		ok := false
		for _, uploaded := range existing.Assets {
			ok = ok || (uploaded.Name == a.Name && uploaded.Digest == "sha256:"+a.SHA256)
		}
		if !ok {
			return fmt.Errorf("uploaded release asset digest was not verified: %s", a.Name)
		}
	}
	return RunCommand(repo, "gh", "release", "edit", tag, "--repo", repository, "--draft=false")
}

func downloadFile(url, path, hash string) error {
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("artifact download requires HTTPS")
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("artifact HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, io.LimitReader(resp.Body, 512*1024*1024))
	f.Close()
	if err != nil {
		return err
	}
	if hash != "" {
		a, e := digest(path)
		if e != nil {
			return e
		}
		if a.SHA256 != hash {
			return fmt.Errorf("downloaded artifact digest mismatch")
		}
	}
	return nil
}

func reuseComponent(repo *machine.Repository, s Selection, c Component, dir, repository, apiRoot string) error {
	if c.SourceSHA == s.SourceSHA && c.State != "UNCHANGED" {
		return fmt.Errorf("Package cannot silently build an unselected unpublished component: %s", c.ID)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if c.ID == "tool" {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get("https://pypi.org/pypi/PTSIP/" + c.Version + "/json")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("Tool package is not published on PyPI")
		}
		var p struct {
			Info struct{ Version string } `json:"info"`
			URLs []struct {
				Filename, URL string
				Digests       struct {
					SHA256 string `json:"sha256"`
				} `json:"digests"`
			} `json:"urls"`
		}
		if err = json.NewDecoder(resp.Body).Decode(&p); err != nil {
			return err
		}
		if p.Info.Version != c.Version || len(p.URLs) != 2 {
			return fmt.Errorf("published Tool identity or artifact set mismatch")
		}
		artifacts := []Artifact{}
		for _, u := range p.URLs {
			if filepath.Base(u.Filename) != u.Filename {
				return fmt.Errorf("unsafe PyPI filename")
			}
			path := filepath.Join(dir, u.Filename)
			if err = downloadFile(u.URL, path, u.Digests.SHA256); err != nil {
				return err
			}
			a, err := digest(path)
			if err != nil {
				return err
			}
			artifacts = append(artifacts, a)
		}
		return writeJSON(filepath.Join(dir, "tool-"+c.Version+"-any.receipt.json"), Receipt{Format: "ptsip-component-release-receipt/v1", Component: c.ID, Version: c.Version, Tag: c.Tag, SourceSHA: c.SourceSHA, Fingerprint: c.Fingerprint, Platform: "any", Verification: "PASS", Artifacts: artifacts})
	}
	return RunCommand(repo, "gh", "release", "download", c.Tag, "--repo", repository, "--dir", dir, "--pattern", c.ID+"-*")
}

func Assemble(repo *machine.Repository, selection, input, output, repository, apiRoot string, publish bool) (machine.Object, error) {
	s, err := readSelection(repo, selection)
	if err != nil {
		return nil, err
	}
	if !s.PackageSelected {
		return nil, fmt.Errorf("Package release is switched off")
	}
	in, err := repo.Path(input)
	if err != nil {
		return nil, err
	}
	out, err := repo.Path(output)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return nil, err
	}
	files := map[string]string{}
	receipts := []Receipt{}
	for _, c := range s.Components {
		dir := filepath.Join(in, c.ID)
		if _, err = os.Stat(dir); os.IsNotExist(err) {
			if err = reuseComponent(repo, s, c, dir, repository, apiRoot); err != nil {
				return nil, err
			}
		}
		r, err := VerifyReceipts(s, c.ID, dir, true)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, r...)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				files[c.ID+"/"+e.Name()] = filepath.Join(dir, e.Name())
			}
		}
	}
	manifest := machine.Object{"format": "ptsip-package-release/v1", "package_version": s.PackageVersion, "composition_sha256": s.CompositionSHA256, "versions": s.Versions, "source_sha": s.SourceSHA, "components": receipts, "installation_or_activation_performed": false}
	path := filepath.Join(out, "package-manifest.json")
	if err = writeJSON(path, manifest); err != nil {
		return nil, err
	}
	files["package-manifest.json"] = path
	zipPath := filepath.Join(out, "ptsip-package-"+s.PackageVersion+".zip")
	if err = zipFiles(repo, zipPath, files); err != nil {
		return nil, err
	}
	if publish {
		if dirty, err := requireRun(repo, "git", "status", "--porcelain", "--untracked-files=no"); err != nil || dirty != "" {
			return nil, fmt.Errorf("Package publication requires a clean exact-source checkout")
		}
		for _, component := range s.Components {
			if component.State == "BUILD" {
				client, e := newGitHubClient(repository, apiRoot)
				if e != nil {
					return nil, e
				}
				selected := []Receipt{}
				for _, r := range receipts {
					if r.Component == component.ID {
						selected = append(selected, r)
					}
				}
				if e = verifyBuildEvidence(client, s, component, selected, os.Getenv("GITHUB_RUN_ID")); e != nil {
					return nil, e
				}
			}
			client, e := newGitHubClient(repository, apiRoot)
			if e != nil {
				return nil, e
			}
			var record struct {
				Draft  bool                            `json:"draft"`
				Assets []struct{ Name, Digest string } `json:"assets"`
			}
			if e = client.request("GET", "/repos/"+repository+"/releases/tags/"+component.Tag, nil, &record); e != nil || record.Draft {
				return nil, fmt.Errorf("Package requires every component publication to complete: %s", component.ID)
			}
			if component.ID != "tool" {
				for _, r := range receipts {
					if r.Component != component.ID {
						continue
					}
					for _, a := range r.Artifacts {
						found := false
						for _, uploaded := range record.Assets {
							found = found || (uploaded.Name == a.Name && uploaded.Digest == "sha256:"+a.SHA256)
						}
						if !found {
							return nil, fmt.Errorf("Package component publication digest mismatch: %s", a.Name)
						}
					}
				}
			} else {
				response, e := http.Get("https://pypi.org/pypi/PTSIP/" + component.Version + "/json")
				if e != nil {
					return nil, e
				}
				var published struct {
					URLs []struct {
						Filename string
						Digests  struct {
							SHA256 string `json:"sha256"`
						} `json:"digests"`
					} `json:"urls"`
				}
				e = json.NewDecoder(io.LimitReader(response.Body, 2*1024*1024)).Decode(&published)
				response.Body.Close()
				if e != nil || response.StatusCode != 200 {
					return nil, fmt.Errorf("Package requires the completed PyPI Tool publication")
				}
				for _, r := range receipts {
					if r.Component == "tool" {
						for _, a := range r.Artifacts {
							found := false
							for _, u := range published.URLs {
								found = found || (u.Filename == a.Name && u.Digests.SHA256 == a.SHA256)
							}
							if !found {
								return nil, fmt.Errorf("Package Tool artifact does not match PyPI")
							}
						}
					}
				}
			}
		}
		if _, err = ReconfirmMain(repo, s.SourceSHA); err != nil {
			return nil, err
		}
		client, err := newGitHubClient(repository, apiRoot)
		if err != nil {
			return nil, err
		}
		var existing struct {
			Assets []struct {
				Name string
				URL  string `json:"browser_download_url"`
			} `json:"assets"`
		}
		tag := "package-v" + s.PackageVersion
		e := client.request("GET", "/repos/"+repository+"/releases/tags/"+tag, nil, &existing)
		if e == nil {
			for _, a := range existing.Assets {
				if a.Name == "package-manifest.json" {
					check := filepath.Join(out, "existing-package-manifest.json")
					if err = downloadFile(a.URL, check, ""); err != nil {
						return nil, err
					}
					var old struct {
						Composition string `json:"composition_sha256"`
					}
					data, err := os.ReadFile(check)
					os.Remove(check)
					if err != nil {
						return nil, err
					}
					if err = json.Unmarshal(data, &old); err != nil {
						return nil, err
					}
					if old.Composition != s.CompositionSHA256 {
						return nil, fmt.Errorf("six-character Package identifier collision")
					}
					return machine.Object{"status": "PASS", "result": "UNCHANGED", "package_version": s.PackageVersion}, nil
				}
			}
			return nil, fmt.Errorf("existing Package has no verified composition manifest")
		}
		if !strings.Contains(e.Error(), "404") {
			return nil, e
		}
		if err = publishFiles(repo, repository, apiRoot, tag, s.SourceSHA, "PTSIP Package "+s.PackageVersion, s.PackageVersion, out, s.CompositionSHA256); err != nil {
			return nil, err
		}
	}
	return machine.Object{"status": "PASS", "package_version": s.PackageVersion, "composition_sha256": s.CompositionSHA256, "artifact": zipPath, "installation_or_activation_performed": false}, nil
}
