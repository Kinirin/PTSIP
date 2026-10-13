package releasecontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Asset struct {
	Name, Digest string
	URL          string `json:"browser_download_url"`
}
type Carrier struct {
	ID        int64
	Tag       string `json:"tag_name"`
	Draft     bool
	Published string `json:"published_at"`
	Assets    []Asset
}

// ReleaseAPI keeps creation and the returned numeric release identity together.
// Listing a newly created draft is not a publication readiness signal.
type ReleaseAPI func(method, endpoint string, payload any, result any) error

func GitHubReleaseAPI(method, endpoint string, payload any, result any) error {
	args := []string{"api", endpoint, "--method", method}
	var input []byte
	if payload != nil {
		var err error
		input, err = json.Marshal(payload)
		if err != nil {
			return err
		}
		args = append(args, "--input", "-")
	}
	cmd := exec.Command("gh", args...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("GitHub release %s failed (%v): %s", method, err, diagnostic.String())
	}
	return json.Unmarshal(data, result)
}

func ResolveCarrier(tag, source, title, body string, resumeID int64, api ReleaseAPI) (Carrier, error) {
	const base = "repos/Kinirin/PTSIP/releases"
	read := func(id int64) (Carrier, error) {
		var c Carrier
		if err := api("GET", fmt.Sprintf("%s/%d", base, id), nil, &c); err != nil {
			return c, err
		}
		if c.ID != id || c.Tag != tag {
			return Carrier{}, fmt.Errorf("release ID does not match the verified source tag")
		}
		return c, nil
	}
	if resumeID > 0 {
		return read(resumeID)
	}
	var matching *Carrier
	for page := 1; ; page++ {
		var rows []Carrier
		if err := api("GET", fmt.Sprintf("%s?per_page=100&page=%d", base, page), nil, &rows); err != nil {
			return Carrier{}, err
		}
		for _, c := range rows {
			if c.Tag == tag {
				if matching != nil {
					return Carrier{}, fmt.Errorf("duplicate source release carrier")
				}
				candidate := c
				matching = &candidate
			}
		}
		if len(rows) < 100 {
			break
		}
	}
	if matching != nil {
		return read(matching.ID)
	}
	var created Carrier
	payload := map[string]any{"tag_name": tag, "target_commitish": source, "name": title, "body": body, "draft": true, "prerelease": true, "make_latest": "false"}
	if err := api("POST", base, payload, &created); err != nil {
		return Carrier{}, err
	}
	if created.ID <= 0 || created.Tag != tag || !created.Draft {
		return Carrier{}, fmt.Errorf("release creation returned an invalid draft identity")
	}
	c, err := read(created.ID)
	if err != nil {
		return Carrier{}, fmt.Errorf("created draft %d could not be read: %w; resume with its verified release ID", created.ID, err)
	}
	return c, nil
}

func Carriers() ([]Carrier, error) {
	all := []Carrier{}
	for page := 1; ; page++ {
		var rows []Carrier
		if err := GitHub(fmt.Sprintf("repos/Kinirin/PTSIP/releases?per_page=100&page=%d", page), &rows); err != nil {
			return nil, err
		}
		all = append(all, rows...)
		if len(rows) < 100 {
			break
		}
	}
	return all, nil
}

// Registry assets on published source carriers persist composition history.
// Draft assets and the source being prepared are never a baseline authority.
func PublishedRegistry(root, ref, excludeSource string) (Registry, error) {
	var registry Registry
	if err := ReadJSON(root, ref, &registry); err != nil {
		return registry, err
	}
	if ref != ".github/checks/release/package-registry.json" {
		return registry, ValidateRegistry(registry)
	}
	rows, err := Carriers()
	if err != nil {
		return registry, err
	}
	composition, err := ReadComposition(root)
	if err != nil {
		return registry, err
	}
	registry.NumericBase = composition.PackageBase
	sort.Slice(rows, func(i, j int) bool { return rows[i].Published > rows[j].Published })
	for _, r := range rows {
		if r.Draft || !strings.HasPrefix(r.Tag, "ptsip-release-") || r.Tag == "ptsip-release-"+excludeSource {
			continue
		}
		for _, a := range r.Assets {
			if a.Name == "package-registry.json" {
				b, e := DownloadArtifact(a.URL, strings.TrimPrefix(a.Digest, "sha256:"))
				if e != nil {
					return registry, e
				}
				var remote Registry
				if e = json.Unmarshal(b, &remote); e != nil {
					return registry, e
				}
				if e = ValidateRegistry(remote); e != nil {
					return registry, e
				}
				// The current base is an explicitly controlled repository setting.
				remote.NumericBase = registry.NumericBase
				return remote, nil
			}
		}
	}
	return registry, ValidateRegistry(registry)
}

func EngineSourceChanges(root, origin, target, id string) (string, error) {
	var contract struct {
		Reuse struct {
			Boundaries []string `json:"source_boundaries"`
		} `json:"engine_reuse"`
	}
	if err := ReadJSON(root, ContractPath, &contract); err != nil {
		return "", err
	}
	args := []string{"git", "diff", "--name-only", origin, target, "--"}
	if len(contract.Reuse.Boundaries) == 0 {
		return "", fmt.Errorf("engine reuse source contract missing")
	}
	for _, ref := range contract.Reuse.Boundaries {
		ref = strings.ReplaceAll(ref, "{engine}", id)
		if _, err := SafePath(root, ref); err != nil {
			return "", err
		}
		args = append(args, ref)
	}
	delta, err := Run(root, args...)
	return strings.TrimSpace(string(delta)), err
}

func EngineReuse(root string, s Selection, components []PublishedComponent, locations map[string]string) ([]PublishedComponent, error) {
	carriers, err := Carriers()
	if err != nil {
		return nil, err
	}
	for i, c := range components {
		if c.Kind != "engine" {
			continue
		}
		for _, r := range carriers {
			if r.Draft || r.Tag != "engine-"+c.ID+"-v"+c.Version {
				continue
			}
			source, err := Run(root, "git", "rev-parse", r.Tag+"^{commit}")
			if err != nil {
				return nil, err
			}
			origin := strings.TrimSpace(string(source))
			delta, err := EngineSourceChanges(root, origin, s.SourceSHA, c.ID)
			if err != nil {
				return nil, err
			}
			if len(delta) != 0 {
				return nil, fmt.Errorf("published engine %s %s changed (%s); explicit new engine version required", c.ID, c.Version, delta)
			}
			reused := c
			reused.SourceSHA = origin
			reused.Files = []File{}
			for _, f := range c.Files {
				found := false
				for _, a := range r.Assets {
					if a.Name == f.Name {
						b, e := DownloadArtifact(a.URL, strings.TrimPrefix(a.Digest, "sha256:"))
						if e != nil {
							return nil, e
						}
						stage := filepath.Join(root, ".artifacts/release/reused", c.ID, f.Name)
						if e = os.MkdirAll(filepath.Dir(stage), 0755); e != nil {
							return nil, e
						}
						if e = os.WriteFile(stage, b, 0644); e != nil {
							return nil, e
						}
						f.SHA256 = Digest(b)
						f.URL = a.URL
						locations[c.ID+"/"+f.Platform+"/"+f.Name] = stage
						reused.Files = append(reused.Files, f)
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("original immutable engine artifact missing: %s", f.Name)
				}
			}
			components[i] = reused
			break
		}
	}
	return components, nil
}

type ShippingManifest struct {
	Schema          string    `json:"schema_version"`
	Selection       Selection `json:"selection"`
	VerificationRun int64     `json:"verification_run"`
	Snapshot        Snapshot  `json:"package"`
	Assets          []File    `json:"assets"`
}

func Shipping(root string, s Selection, output string) (ShippingManifest, error) {
	var m ShippingManifest
	if err := ReadJSON(root, output+"/shipping-manifest.json", &m); err != nil {
		return m, err
	}
	a, _ := json.Marshal(m.Selection)
	b, _ := json.Marshal(s)
	if m.Schema != "ptsip-shipping-manifest/v1" || string(a) != string(b) || m.VerificationRun <= 0 {
		return m, fmt.Errorf("shipping admission mismatch")
	}
	if err := ValidateSnapshot(m.Snapshot); err != nil {
		return m, err
	}
	out, err := SafePath(root, output)
	if err != nil {
		return m, err
	}
	names := map[string]bool{}
	for _, f := range m.Assets {
		if f.Name == "" || filepath.Base(f.Name) != f.Name || strings.ContainsAny(f.Name, `\\/:`) || names[f.Name] {
			return m, fmt.Errorf("invalid shipping asset")
		}
		names[f.Name] = true
		b, e := os.ReadFile(filepath.Join(out, f.Name))
		if e != nil || Digest(b) != f.SHA256 {
			return m, fmt.Errorf("shipping asset changed: %s", f.Name)
		}
	}
	for _, c := range m.Selection.Components {
		for _, component := range m.Snapshot.Components {
			if component.ID == c.ID {
				for _, f := range component.Files {
					if !names[f.Name] {
						return m, fmt.Errorf("selected artifact missing from shipping admission")
					}
				}
			}
		}
	}
	return m, nil
}

type PyPIFile struct {
	Name    string `json:"filename"`
	Digests struct {
		SHA256 string `json:"sha256"`
	}
}

func PyPIFiles(project, version string) ([]PyPIFile, error) {
	client := &http.Client{Timeout: time.Minute}
	response, err := client.Get("https://pypi.org/pypi/" + project + "/" + version + "/json")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == 404 {
		return nil, nil
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("PyPI version lookup failed: HTTP %d", response.StatusCode)
	}
	var result struct {
		Files []PyPIFile `json:"urls"`
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 8*1024*1024)).Decode(&result)
	return result.Files, err
}

// Existing immutable files are removed only from the upload staging directory
// after a matching public digest. A differing same-name file aborts publication.
func PyPIStage(root string, s Selection, output string, verify bool) error {
	m, err := Shipping(root, s, output)
	if err != nil {
		return err
	}
	for _, c := range m.Snapshot.Components {
		selected := false
		for _, v := range s.Components {
			selected = selected || v.ID == c.ID
		}
		if !selected || c.Kind == "profiles" {
			continue
		}
		project := "PTSIP"
		if c.Kind == "engine" {
			project = "ptsip-engine-" + c.ID
		}
		existing, e := PyPIFiles(project, c.Version)
		if e != nil {
			return e
		}
		for _, f := range c.Files {
			if !strings.HasSuffix(f.Name, ".whl") && !strings.HasSuffix(f.Name, ".tar.gz") {
				continue
			}
			match := false
			for _, old := range existing {
				if old.Name == f.Name {
					if old.Digests.SHA256 != f.SHA256 {
						return fmt.Errorf("immutable PyPI file differs: %s; explicit new component version required", f.Name)
					}
					match = true
				}
			}
			if verify && !match {
				return fmt.Errorf("PyPI publication not observed: %s", f.Name)
			}
			if !verify && match {
				p, e := SafePath(root, output+"/pypi/"+c.ID+"/"+f.Name)
				if e != nil {
					return e
				}
				if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
					return e
				}
			}
		}
	}
	return nil
}

func PublisherEnvironment(root string, c Component) (string, error) {
	var routing struct {
		Schema string `json:"schema_version"`
		Tool   struct{ Project, Environment string }
		Engine struct {
			Project     string `json:"project_pattern"`
			Environment string `json:"environment_pattern"`
		}
	}
	if err := ReadJSON(root, ".github/checks/release/pypi-publishers.json", &routing); err != nil {
		return "", err
	}
	if routing.Schema != "ptsip-pypi-publisher-routing/v1" {
		return "", fmt.Errorf("publisher routing contract unsupported")
	}
	if c.Kind == "tool" {
		return routing.Tool.Environment, nil
	}
	if c.Kind == "engine" && c.ID != "" {
		return strings.ReplaceAll(routing.Engine.Environment, "{engine}", c.ID), nil
	}
	return "", fmt.Errorf("component has no PyPI publisher route")
}

func RecordPackage(root string, s Selection, output string) error {
	m, err := Shipping(root, s, output)
	if err != nil {
		return err
	}
	var carrier Carrier
	if err = GitHub("repos/Kinirin/PTSIP/releases/tags/"+s.Tag, &carrier); err != nil {
		return err
	}
	if carrier.Draft {
		return fmt.Errorf("Package carrier is not published")
	}
	var registryBytes, noteBytes []byte
	for _, a := range carrier.Assets {
		if a.Name == "package-registry.json" || a.Name == m.Snapshot.Version+".md" {
			b, e := DownloadArtifact(a.URL, strings.TrimPrefix(a.Digest, "sha256:"))
			if e != nil {
				return e
			}
			if a.Name == "package-registry.json" {
				registryBytes = b
			} else {
				noteBytes = b
			}
		}
	}
	var registry Registry
	if len(noteBytes) == 0 || json.Unmarshal(registryBytes, &registry) != nil {
		return fmt.Errorf("published Package record missing")
	}
	if err = ValidateRegistry(registry); err != nil {
		return err
	}
	selected, err := ResolvePackage(registry, m.Snapshot.Version)
	if err != nil || selected.Digest != m.Snapshot.Digest {
		return fmt.Errorf("published Package record differs")
	}
	// Only these two derived records are written; policy and component versions
	// remain owned by their source contracts.
	if err = WriteJSON(filepath.Join(root, ".github/checks/release/package-registry.json"), registry); err != nil {
		return err
	}
	if err = RecordPublishedDeclaration(root, m.Snapshot); err != nil {
		return err
	}
	noteRef := "docs/releasenote/PTSIP_Package/" + m.Snapshot.Version + ".md"
	p, err := SafePath(root, noteRef)
	if err != nil {
		return err
	}
	if err = os.WriteFile(p, noteBytes, 0644); err != nil {
		return err
	}
	if _, err = Run(root, "git", "add", "--", ".github/checks/release/package-registry.json", noteRef, "pyproject.toml"); err != nil {
		return err
	}
	delta, err := Run(root, "git", "diff", "--cached", "--name-only")
	if err != nil {
		return err
	}
	if len(delta) == 0 {
		return nil
	}
	if _, err = Run(root, "git", "-c", "user.name=github-actions[bot]", "-c", "user.email=41898282+github-actions[bot]@users.noreply.github.com", "commit", "-m", "docs(release): record Package "+m.Snapshot.Version); err != nil {
		return err
	}
	_, err = Run(root, "git", "push", "origin", "HEAD:main")
	return err
}
