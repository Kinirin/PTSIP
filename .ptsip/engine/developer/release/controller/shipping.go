package releasecontroller

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func GitHub(endpoint string, out any) error {
	data, err := exec.Command("gh", "api", endpoint).Output()
	if err != nil {
		return fmt.Errorf("GitHub evidence read failed: %w", err)
	}
	return json.Unmarshal(data, out)
}

func AdmitVerification(root string, s Selection, run int64) error {
	if err := ValidateSelection(root, s); err != nil {
		return err
	}
	if err := AdmitCTRLPublicationPolicy(root); err != nil {
		return err
	}
	head, err := Run(root, "git", "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != s.SourceSHA {
		return fmt.Errorf("checkout source does not match selection")
	}
	tracked, err := Run(root, "git", "status", "--porcelain", "--untracked-files=no")
	if err != nil || len(tracked) != 0 {
		return fmt.Errorf("tracked source changed before publication")
	}
	if run <= 0 {
		return fmt.Errorf("original verification run required")
	}
	var current struct {
		Head                     string `json:"head_sha"`
		Path, Status, Conclusion string
	}
	if err := GitHub("repos/Kinirin/PTSIP/actions/runs/"+strconv.FormatInt(run, 10), &current); err != nil {
		return err
	}
	if current.Head != s.SourceSHA || current.Path != ".github/workflows/tooling-test.yml" || current.Status != "completed" || current.Conclusion != "success" {
		return fmt.Errorf("exact-source tooling-test admission missing")
	}
	type job struct{ Name, Status, Conclusion string }
	var jobs struct {
		Jobs []job
	}
	for page := 1; ; page++ {
		var batch struct{ Jobs []job }
		if err := GitHub(fmt.Sprintf("repos/Kinirin/PTSIP/actions/runs/%d/jobs?per_page=100&page=%d", run, page), &batch); err != nil {
			return err
		}
		jobs.Jobs = append(jobs.Jobs, batch.Jobs...)
		if len(batch.Jobs) < 100 {
			break
		}
	}
	for _, c := range s.Components {
		for _, platform := range s.Platforms {
			name := "Registered verification / " + c.ID + " / " + platform
			count := 0
			for _, job := range jobs.Jobs {
				if job.Name == name && job.Status == "completed" && job.Conclusion == "success" {
					count++
				}
			}
			if count != 1 {
				return fmt.Errorf("required component/host verification missing: %s", name)
			}
		}
	}
	if s.Type == "PTSIP_PACKAGE" {
        waived,waiverErr:=LightweightPackageWaiver(root,s)
        if waiverErr!=nil{return waiverErr}
        for _, platform := range s.Platforms {
            name := "Registered join / package / " + platform
            if waived {name = "Lightweight install / package / " + platform}
			count := 0
			for _, job := range jobs.Jobs {
				if job.Name == name && job.Status == "completed" && job.Conclusion == "success" {
					count++
				}
			}
			if count != 1 {
				return fmt.Errorf("required original package join missing: %s", name)
			}
		}
	}
	return nil
}

func Gather(root string, s Selection, input string) ([]PublishedComponent, map[string]string, error) {
	dir, err := SafePath(root, input)
	if err != nil {
		return nil, nil, err
	}
	type receipt struct {
		Schema                                     string `json:"schema_version"`
		SourceSHA                                  string `json:"source_sha"`
		SelectionSHA                               string `json:"selection_sha256"`
		Component, Kind, Version, Platform, Status string
		Files                                      []File
	}
	rows := map[string]receipt{}
	locations := map[string]string{}
	err = filepath.WalkDir(dir, func(p string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked shipping artifact")
		}
		if entry.Name() != "test-receipt.json" || entry.IsDir() {
			return nil
		}
		data, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		var r receipt
		if e := json.Unmarshal(data, &r); e != nil {
			return e
		}
		if r.Schema != "ptsip-release-test-receipt/v1" || r.SourceSHA != s.SourceSHA || r.SelectionSHA != s.SelectionSHA || r.Status != "PASS" {
			return fmt.Errorf("receipt source or selection mismatch")
		}
		allowed := false
		for _, c := range s.Components {
			for _, platform := range s.Platforms {
				allowed = allowed || (c.ID == r.Component && c.Kind == r.Kind && c.Version == r.Version && platform == r.Platform)
			}
		}
		if !allowed || len(r.Files) == 0 {
			return fmt.Errorf("receipt outside selected component/host set")
		}
		key := r.Component + "/" + r.Platform
		if _, ok := rows[key]; ok {
			return fmt.Errorf("duplicate component/host receipt")
		}
		rows[key] = r
		for _, file := range r.Files {
			if filepath.Base(file.Name) != file.Name {
				return fmt.Errorf("invalid artifact filename")
			}
			asset := filepath.Join(filepath.Dir(p), file.Name)
			b, e := os.ReadFile(asset)
			if e != nil || Digest(b) != file.SHA256 {
				return fmt.Errorf("artifact digest mismatch: %s", file.Name)
			}
			locations[key+"/"+file.Name] = asset
		}
		locations[key+"/receipt"] = p
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	components := []PublishedComponent{}
	for _, c := range s.Components {
		published := PublishedComponent{ID: c.ID, Kind: c.Kind, Version: c.Version, SourceSHA: s.SourceSHA, Files: []File{}}
		for _, platform := range s.Platforms {
			key := c.ID + "/" + platform
			r, ok := rows[key]
			if !ok || r.Version != c.Version || r.Kind != c.Kind {
				return nil, nil, fmt.Errorf("missing receipt for %s", key)
			}
			if c.Kind != "engine" && platform != "linux-x64" {
				continue
			}
			for _, file := range r.Files {
				file.Platform = platform
				if c.Kind != "engine" {
					file.Platform = "any"
				}
				file.URL = "https://github.com/Kinirin/PTSIP/releases/download/" + s.Tag + "/" + file.Name
				published.Files = append(published.Files, file)
			}
		}
		components = append(components, published)
	}
	return components, locations, nil
}

func PrepareShipping(root string, s Selection, run int64, input, output string) (Snapshot, error) {
	if err := AdmitVerification(root, s, run); err != nil {
		return Snapshot{}, err
	}
	components, locations, err := Gather(root, s, input)
	if err != nil {
		return Snapshot{}, err
	}
	out, err := SafePath(root, output)
	if err != nil {
		return Snapshot{}, err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return Snapshot{}, err
	}
	components, err = EngineReuse(root, s, components, locations)
	if err != nil {
		return Snapshot{}, err
	}
	registry, err := PublishedRegistry(root, ".ptsip/engine/developer/release/state/package-registry.json", s.SourceSHA)
	if err != nil {
		return Snapshot{}, err
	}
	components, err = ReusePublishedComponents(root, s, registry, components, locations)
	if err != nil {
		return Snapshot{}, err
	}
	registry.NumericBase = s.PackageBase
	snapshot, err := RefreshPackage(&registry, s.SourceSHA, components)
	if err != nil {
		return Snapshot{}, err
	}
	composition, err := ReadComposition(root)
	if err != nil {
		return Snapshot{}, err
	}
	if err := CompletePackage(snapshot, composition.Engines); err != nil {
		return Snapshot{}, err
	}
	for _, c := range components {
		for _, file := range c.Files {
			platform := file.Platform
			if platform == "any" {
				platform = "linux-x64"
			}
			p := locations[c.ID+"/"+platform+"/"+file.Name]
			b, err := os.ReadFile(p)
			if err != nil {
				return Snapshot{}, err
			}
			if err := os.WriteFile(filepath.Join(out, file.Name), b, 0644); err != nil {
				return Snapshot{}, err
			}
			if strings.HasSuffix(file.Name, ".whl") || strings.HasSuffix(file.Name, ".tar.gz") {
				pypi := filepath.Join(out, "pypi", c.ID)
				if err := os.MkdirAll(pypi, 0755); err != nil {
					return Snapshot{}, err
				}
				if err := os.WriteFile(filepath.Join(pypi, file.Name), b, 0644); err != nil {
					return Snapshot{}, err
				}
			}
		}
	}
	if err := WriteJSON(filepath.Join(out, "package-registry.json"), registry); err != nil {
		return Snapshot{}, err
	}
	if err := WriteJSON(filepath.Join(out, "package-"+snapshot.Version+".json"), snapshot); err != nil {
		return Snapshot{}, err
	}
	_, body, err := ReleasePresentation(root, s, snapshot)
	if err != nil {
		return Snapshot{}, err
	}
	if err := os.WriteFile(filepath.Join(out, "release-body.md"), []byte(body), 0644); err != nil {
		return Snapshot{}, err
	}
	if err := os.WriteFile(filepath.Join(out, snapshot.Version+".md"), []byte(body), 0644); err != nil {
		return Snapshot{}, err
	}
	manifest := ShippingManifest{Schema: "ptsip-shipping-manifest/v1", Selection: s, VerificationRun: run, Snapshot: snapshot, Assets: []File{}}
	if composition.PackagePyPI != "" && s.Type == "PTSIP_PACKAGE" {
		manifest.PackageDistribution, err = ConsumePackageReceipt(root, s, snapshot, input, output, composition.PackagePyPI)
		if err != nil {
			return Snapshot{}, err
		}
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return Snapshot{}, err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "shipping-manifest.json" {
			continue
		}
		b, e := os.ReadFile(filepath.Join(out, entry.Name()))
		if e != nil {
			return Snapshot{}, e
		}
		manifest.Assets = append(manifest.Assets, File{Name: entry.Name(), SHA256: Digest(b)})
	}
	if err := WriteJSON(filepath.Join(out, "shipping-manifest.json"), manifest); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func PublishCarrier(root string, s Selection, output string, resumeID int64) error {
	if err := AdmitCTRLPublicationPolicy(root); err != nil {
		return err
	}
	m, err := Shipping(root, s, output)
	if err != nil {
		return err
	}
	var main struct{ Object struct{ SHA string } }
	controlSource := m.PublicationSource
	if controlSource == "" {
		controlSource = s.SourceSHA
	}
	if err := GitHub("repos/Kinirin/PTSIP/git/ref/heads/main", &main); err != nil || main.Object.SHA != controlSource {
		return fmt.Errorf("main advanced beyond shipping source")
	}
	out, err := SafePath(root, output)
	if err != nil {
		return err
	}
	refs, err := Run(root, "git", "ls-remote", "origin", "refs/tags/"+s.Tag)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(refs)) != "" && !strings.HasPrefix(string(refs), s.SourceSHA+"\t") {
		return fmt.Errorf("existing source tag mismatch")
	}
	args := []string{"api", "repos/Kinirin/PTSIP/git/refs", "--method", "POST", "-f", "ref=refs/tags/" + s.Tag, "-f", "sha=" + s.SourceSHA}
	if strings.TrimSpace(string(refs)) == "" {
		if _, err := Run(root, append([]string{"gh"}, args...)...); err != nil {
			return err
		}
	}
	body, err := os.ReadFile(filepath.Join(out, "release-body.md"))
	if err != nil {
		return err
	}
	title, expectedBody, err := ReleasePresentation(root, s, m.Snapshot)
	if err != nil {
		return err
	}
	if string(body) != expectedBody {
		return fmt.Errorf("release body differs from the admitted presentation contract")
	}
	carrier, err := ResolveCarrier(s.Tag, s.SourceSHA, title, string(body), resumeID, GitHubReleaseAPI)
	if err != nil {
		return err
	}
	existing := map[string]string{}
	for _, a := range carrier.Assets {
		existing[a.Name] = a.Digest
	}
	draft, releaseID := carrier.Draft, carrier.ID
	assets := append([]File{}, m.Assets...)
	bm, err := os.ReadFile(filepath.Join(out, "shipping-manifest.json"))
	if err != nil {
		return err
	}
	assets = append(assets, File{Name: "shipping-manifest.json", SHA256: Digest(bm)})
	for _, entry := range assets {
		p := filepath.Join(out, entry.Name)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if old := existing[entry.Name]; old != "" {
			if old != "sha256:"+Digest(b) {
				return fmt.Errorf("immutable carrier asset mismatch")
			}
			continue
		}
		if !draft {
			return fmt.Errorf("cannot add unreviewed assets to published source carrier")
		}
		response, err := Run(root, "gh", "api", fmt.Sprintf("https://uploads.github.com/repos/Kinirin/PTSIP/releases/%d/assets?name=%s", releaseID, url.QueryEscape(entry.Name)), "--method", "POST", "--header", "Content-Type: application/octet-stream", "--input", p)
		if err != nil {
			return err
		}
		var asset Asset
		if json.Unmarshal(response, &asset) != nil || asset.Name != entry.Name || asset.Digest != "sha256:"+entry.SHA256 {
			return fmt.Errorf("uploaded release asset identity/hash differs")
		}
	}
	var uploaded struct {
		Assets []struct{ Name, Digest string }
	}
	if err := GitHub("repos/Kinirin/PTSIP/releases/"+strconv.FormatInt(releaseID, 10), &uploaded); err != nil {
		return err
	}
	for _, expected := range assets {
		found := false
		for _, a := range uploaded.Assets {
			if a.Name == expected.Name {
				found = a.Digest == "sha256:"+expected.SHA256
			}
		}
		if !found {
			return fmt.Errorf("carrier uploaded artifact missing or hash mismatch: %s", expected.Name)
		}
	}
	if draft {
		if err := GitHub("repos/Kinirin/PTSIP/git/ref/heads/main", &main); err != nil || main.Object.SHA != controlSource {
			return fmt.Errorf("main advanced before draft publication")
		}
		var published Carrier
		err = GitHubReleaseAPI("PATCH", fmt.Sprintf("repos/Kinirin/PTSIP/releases/%d", releaseID), map[string]any{"draft": false, "make_latest": "false"}, &published)
		if err == nil && (published.ID != releaseID || published.Tag != s.Tag || published.Draft) {
			return fmt.Errorf("GitHub publication state not observed")
		}
	}
	return err
}
