package enginewheel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func githubAPI(endpoint, method string, payload any, out any) error {
	args := []string{"api", endpoint, "--method", method}
	cmd := exec.Command("gh", args...)
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		cmd.Args = append(cmd.Args, "--input", "-")
		cmd.Stdin = bytes.NewReader(body)
	}
	var errors bytes.Buffer
	cmd.Stderr = &errors
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("GitHub %s failed: %s", method, strings.TrimSpace(errors.String()))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func requireFreshMain(opts PublicationOptions) error {
	var branch struct{ Object struct{ SHA string } }
	if err := githubAPI("repos/"+opts.Repository+"/git/ref/heads/main", "GET", nil, &branch); err != nil {
		return err
	}
	if branch.Object.SHA != opts.SourceSHA {
		return fmt.Errorf("main advanced beyond the verified publication source")
	}
	return nil
}

type pypiFile struct {
	Filename string
	Digests  struct{ SHA256 string }
}

func checkPyPIFiles(bundle PublicationBundle, files []pypiFile) (map[string]string, error) {
	existing := map[string]string{}
	for _, file := range files {
		if !has(bundle.Wheels, file.Filename) || existing[file.Filename] != "" || bundle.Files[file.Filename] != file.Digests.SHA256 {
			return nil, fmt.Errorf("existing PyPI component version has a different artifact: %s", file.Filename)
		}
		existing[file.Filename] = file.Digests.SHA256
	}
	return existing, nil
}

func pypiFiles(bundle PublicationBundle) (map[string]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://pypi.org/pypi/" + bundle.Distribution + "/" + bundle.Version + "/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return map[string]string{}, nil
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("PyPI version read returned HTTP %d", resp.StatusCode)
	}
	var release struct {
		URLs []pypiFile `json:"urls"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return nil, err
	}
	return checkPyPIFiles(bundle, release.URLs)
}

func PreparePublication(root string, opts PublicationOptions, token string) (PublicationBundle, error) {
	root, err := filepath.Abs(root)
	if err != nil || opts.Repository != "Kinirin/PTSIP" {
		return PublicationBundle{}, fmt.Errorf("publication requires the admitted Kinirin/PTSIP repository")
	}
	if err := AdmitPublication(root); err != nil {
		return PublicationBundle{}, err
	}
	gate := opts.GateOptions
	gate.Kind = opts.VerifiedEngines
	if _, err := Gate(root, gate, token); err != nil {
		return PublicationBundle{}, err
	}
	var run struct{ Status, Conclusion string }
	if err := githubAPI("repos/"+opts.Repository+"/actions/runs/"+strconv.FormatInt(opts.RunID, 10), "GET", nil, &run); err != nil || run.Status != "completed" || run.Conclusion != "success" {
		return PublicationBundle{}, fmt.Errorf("publication requires a completed successful verification run: %v", err)
	}
	if err := requireFreshMain(opts); err != nil {
		return PublicationBundle{}, err
	}
	bundle, err := validatePublicationBundle(root, opts)
	if err != nil {
		return PublicationBundle{}, err
	}
	existing, err := pypiFiles(bundle)
	if err != nil {
		return PublicationBundle{}, err
	}
	for _, name := range bundle.Wheels {
		if existing[name] != "" {
			continue
		}
		bundle.PyPIPending = true
		pypiDir := filepath.Join(bundle.Directory, "pypi")
		if err := os.MkdirAll(pypiDir, 0755); err != nil {
			return PublicationBundle{}, err
		}
		data, err := os.ReadFile(filepath.Join(bundle.Directory, name))
		if err != nil {
			return PublicationBundle{}, err
		}
		if err := os.WriteFile(filepath.Join(pypiDir, name), data, 0644); err != nil {
			return PublicationBundle{}, err
		}
	}
	manifest, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return PublicationBundle{}, err
	}
	if err := os.WriteFile(filepath.Join(bundle.Directory, "publication-receipt.json"), append(manifest, '\n'), 0644); err != nil {
		return PublicationBundle{}, err
	}
	return bundle, nil
}

func PublishGitHub(root string, opts PublicationOptions, token string) (PublicationBundle, error) {
	bundle, err := PreparePublication(root, opts, token)
	if err != nil {
		return bundle, err
	}
	if err := requireFreshMain(opts); err != nil {
		return bundle, err
	}
	var ref struct{ Object struct{ SHA, Type string } }
	err = githubAPI("repos/"+opts.Repository+"/git/ref/tags/"+bundle.Tag, "GET", nil, &ref)
	if err == nil && (ref.Object.SHA != bundle.SourceSHA || ref.Object.Type != "commit") {
		return bundle, fmt.Errorf("existing engine tag cannot be changed")
	}
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return bundle, err
		}
		if err := githubAPI("repos/"+opts.Repository+"/git/refs", "POST", map[string]string{"ref": "refs/tags/" + bundle.Tag, "sha": bundle.SourceSHA}, nil); err != nil {
			return bundle, err
		}
	}
	manifest, err := os.ReadFile(filepath.Join(bundle.Directory, "publication-receipt.json"))
	if err != nil {
		return bundle, err
	}
	wanted := map[string]string{"publication-receipt.json": digest(manifest)}
	for name, sha := range bundle.Files {
		wanted[name] = sha
	}
	type release struct {
		Draft  bool
		Assets []struct{ Name, Digest string }
	}
	var current release
	endpoint := "repos/" + opts.Repository + "/releases/tags/" + bundle.Tag
	err = githubAPI(endpoint, "GET", nil, &current)
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return bundle, err
		}
		notes := fmt.Sprintf("# PTSIP Engine %s %s\n\nIndependent native engine publication. Tool 0.3.8a5 is preserved.\n\nSource: `%s`\nVerification: https://github.com/%s/actions/runs/%d\n\nPlatforms: Windows x64, Linux x64 (glibc 2.39+), macOS arm64.\n\nInstall: `python -m pip install %s==%s`\n\nPublished availability does not authorize runtime activation.\n", bundle.Kind, bundle.Version, bundle.SourceSHA, opts.Repository, bundle.VerificationRunID, bundle.Distribution, bundle.Version)
		note := filepath.Join(bundle.Directory, "publication-notes.md")
		if err := os.WriteFile(note, []byte(notes), 0644); err != nil {
			return bundle, err
		}
		args := []string{"release", "create", bundle.Tag, "--repo", opts.Repository, "--verify-tag", "--draft", "--latest=false", "--title", "PTSIP Engine " + bundle.Kind + " " + bundle.Version, "--notes-file", note}
		if data, err := exec.Command("gh", args...).CombinedOutput(); err != nil {
			return bundle, fmt.Errorf("draft creation failed: %s", data)
		}
		if err := githubAPI(endpoint, "GET", nil, &current); err != nil {
			return bundle, err
		}
	}
	seen := map[string]bool{}
	for _, asset := range current.Assets {
		if wanted[asset.Name] == "" || asset.Digest != "sha256:"+wanted[asset.Name] {
			return bundle, fmt.Errorf("existing release asset differs: %s", asset.Name)
		}
		seen[asset.Name] = true
	}
	for name := range wanted {
		if seen[name] {
			continue
		}
		if !current.Draft {
			return bundle, fmt.Errorf("published release asset set is immutable")
		}
		if data, err := exec.Command("gh", "release", "upload", bundle.Tag, filepath.Join(bundle.Directory, name), "--repo", opts.Repository).CombinedOutput(); err != nil {
			return bundle, fmt.Errorf("asset upload failed: %s", data)
		}
	}
	if err := githubAPI(endpoint, "GET", nil, &current); err != nil {
		return bundle, err
	}
	if len(current.Assets) != len(wanted) {
		return bundle, fmt.Errorf("release asset coverage is incomplete")
	}
	for _, asset := range current.Assets {
		if asset.Digest != "sha256:"+wanted[asset.Name] {
			return bundle, fmt.Errorf("uploaded release digest mismatch: %s", asset.Name)
		}
	}
	if current.Draft {
		if data, err := exec.Command("gh", "release", "edit", bundle.Tag, "--repo", opts.Repository, "--draft=false", "--latest=false").CombinedOutput(); err != nil {
			return bundle, fmt.Errorf("release publication failed: %s", data)
		}
	}
	return bundle, nil
}

func VerifyPublishedPyPI(root string, opts PublicationOptions, token string) (PublicationBundle, error) {
	bundle, err := PreparePublication(root, opts, token)
	if err != nil {
		return bundle, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		files, err := pypiFiles(bundle)
		if err != nil {
			return bundle, err
		}
		if len(files) == len(bundle.Wheels) {
			return bundle, nil
		}
		if attempt < 3 {
			time.Sleep(5 * time.Second)
		}
	}
	return bundle, fmt.Errorf("PyPI does not expose all three verified wheel digests")
}
