package releasecontroller

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v3"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type File struct {
	Name     string `json:"name"`
	SHA256   string `json:"sha256"`
	URL      string `json:"url"`
	Platform string `json:"platform"`
}
type PublishedComponent struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Version   string `json:"version"`
	SourceSHA string `json:"source_sha"`
	Files     []File `json:"files"`
}
type Snapshot struct {
	Version    string               `json:"version"`
	Digest     string               `json:"composition_sha256"`
	SourceSHA  string               `json:"source_sha"`
	Sequence   int                  `json:"sequence"`
	Components []PublishedComponent `json:"components"`
}
type Registry struct {
	Schema      string     `json:"schema_version"`
	NumericBase string     `json:"numeric_base"`
	Latest      *string    `json:"latest"`
	Snapshots   []Snapshot `json:"snapshots"`
}

func RefreshPackage(registry *Registry, source string, changes []PublishedComponent) (Snapshot, error) {
	if err := ValidateRegistry(*registry); err != nil {
		return Snapshot{}, err
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(registry.NumericBase) || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(source) || len(changes) == 0 {
		return Snapshot{}, fmt.Errorf("explicit Package base, source and published changes required")
	}
	components := map[string]PublishedComponent{}
	if len(registry.Snapshots) > 0 {
		for _, c := range registry.Snapshots[len(registry.Snapshots)-1].Components {
			components[c.ID] = c
		}
	}
	for _, c := range changes {
		c.Files = append([]File{}, c.Files...)
		if c.ID == "" || c.Version == "" || len(c.Files) == 0 {
			return Snapshot{}, fmt.Errorf("published component evidence missing")
		}
		for _, file := range c.Files {
			if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(file.SHA256) || file.URL == "" || filepath.Base(file.Name) != file.Name {
				return Snapshot{}, fmt.Errorf("published file identity invalid")
			}
		}
		sort.Slice(c.Files, func(i, j int) bool { return c.Files[i].Name < c.Files[j].Name })
		components[c.ID] = c
	}
	vector := []PublishedComponent{}
	for _, c := range components {
		vector = append(vector, c)
	}
	sort.Slice(vector, func(i, j int) bool { return vector[i].ID < vector[j].ID })
	data, _ := json.Marshal(vector)
	full := Digest(data)
	identity := registry.NumericBase + "." + full[:6]
	for _, old := range registry.Snapshots {
		if old.Version == identity {
			if old.Digest != full {
				return Snapshot{}, fmt.Errorf("Package SHA6 collision")
			}
			return old, nil
		}
	}
	snapshot := Snapshot{identity, full, source, len(registry.Snapshots) + 1, vector}
	if err := ValidateSnapshot(snapshot); err != nil {
		return Snapshot{}, err
	}
	registry.Schema = "ptsip-package-registry/v1"
	registry.Snapshots = append(registry.Snapshots, snapshot)
	registry.Latest = &identity
	return snapshot, nil
}

func ResolvePackage(registry Registry, selector string) (Snapshot, error) {
	if err := ValidateRegistry(registry); err != nil {
		return Snapshot{}, err
	}
	base := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	exact := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9a-f]{6}$`)
	if !base.MatchString(selector) && !exact.MatchString(selector) {
		return Snapshot{}, fmt.Errorf("invalid Package version selector")
	}
	var result *Snapshot
	for i := range registry.Snapshots {
		s := &registry.Snapshots[i]
		if exact.MatchString(selector) && s.Version == selector {
			return *s, nil
		}
		if base.MatchString(selector) && strings.HasPrefix(s.Version, selector+".") && (result == nil || s.Sequence > result.Sequence) {
			result = s
		}
	}
	if result != nil {
		return *result, nil
	}
	return Snapshot{}, fmt.Errorf("Package selector not published: %s", selector)
}

func ValidateSnapshot(s Snapshot) error {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(s.SourceSHA) || s.Sequence < 1 || len(s.Components) == 0 {
		return fmt.Errorf("invalid Package snapshot")
	}
	seen := map[string]bool{}
	for _, c := range s.Components {
		if c.ID == "" || c.Version == "" || seen[c.ID] || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(c.SourceSHA) || len(c.Files) == 0 {
			return fmt.Errorf("invalid Package component")
		}
		seen[c.ID] = true
		if c.Kind != "tool" && c.Kind != "profiles" && c.Kind != "engine" {
			return fmt.Errorf("invalid Package component kind")
		}
		files := map[string]bool{}
		for _, f := range c.Files {
			if f.Name == "" || filepath.Base(f.Name) != f.Name || strings.ContainsAny(f.Name, `\\/:`) || files[f.Name] || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(f.SHA256) || !strings.HasPrefix(f.URL, "https://") {
				return fmt.Errorf("invalid Package artifact")
			}
			files[f.Name] = true
		}
	}
	b, _ := json.Marshal(s.Components)
	if Digest(b) != s.Digest || !strings.HasSuffix(s.Version, "."+s.Digest[:6]) {
		return fmt.Errorf("Package composition digest mismatch")
	}
	return nil
}

func ValidateRegistry(r Registry) error {
	if r.Schema != "ptsip-package-registry/v1" && len(r.Snapshots) > 0 {
		return fmt.Errorf("unregistered Package schema")
	}
	seen := map[string]bool{}
	for i, s := range r.Snapshots {
		if err := ValidateSnapshot(s); err != nil {
			return err
		}
		if seen[s.Version] || s.Sequence != i+1 {
			return fmt.Errorf("ambiguous Package history")
		}
		seen[s.Version] = true
	}
	if len(r.Snapshots) > 0 && (r.Latest == nil || *r.Latest != r.Snapshots[len(r.Snapshots)-1].Version) {
		return fmt.Errorf("stale Package latest pointer")
	}
	return nil
}

func DownloadArtifact(url, digest string) ([]byte, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("HTTPS artifact required")
	}
	client := &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || len(via) >= 10 {
			return fmt.Errorf("unsafe artifact redirect")
		}
		return nil
	}}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("artifact unavailable: HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 268435457))
	if err != nil || len(b) > 268435456 || Digest(b) != digest {
		return nil, fmt.Errorf("artifact digest mismatch")
	}
	return b, nil
}

// A profile distribution contains public catalog resources, never executable
// profile implementation, local consumer declarations, or historical profiles.
func ProfileResources(data []byte) (map[string][]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	resources := map[string][]byte{}
	var registry []byte
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 || f.UncompressedSize64 > 16*1024*1024 {
			return nil, fmt.Errorf("unsafe Profile archive entry")
		}
		name := strings.TrimPrefix(f.Name, "src/ptsip/profiles/")
		isRegistry := f.Name == "registry/project-profile-contracts.yaml"
		if !isRegistry && (name == f.Name || filepath.Base(name) != name || (name != "index.yaml" && !strings.HasSuffix(name, ".ptsip.yaml"))) {
			return nil, fmt.Errorf("non-catalog Profile archive entry: %s", f.Name)
		}
		if _, ok := resources[name]; ok || (isRegistry && registry != nil) {
			return nil, fmt.Errorf("duplicate Profile resource")
		}
		stream, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, e := io.ReadAll(io.LimitReader(stream, 16*1024*1024+1))
		stream.Close()
		if e != nil || len(b) > 16*1024*1024 {
			return nil, fmt.Errorf("Profile resource read failed")
		}
		if isRegistry {
			registry = b
		} else {
			resources[name] = b
		}
	}
	if resources["index.yaml"] == nil || len(resources) < 2 || registry == nil {
		return nil, fmt.Errorf("incomplete Profile catalog")
	}
	resources["@project-profile-contracts.yaml"] = registry
	return resources, nil
}

func InstallPackage(snapshot Snapshot, platform, python string, registered map[string]string) error {
	if python == "" {
		return fmt.Errorf("explicit Python environment required")
	}
	if err := ValidateSnapshot(snapshot); err != nil {
		return err
	}
	if len(registered) == 0 {
		return fmt.Errorf("Package engine inventory is empty")
	}
	if err := CompletePackage(snapshot, registered); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "ptsip-package-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	wheels := []string{}
	var profileResources map[string][]byte
	var profileContract []byte
	for _, c := range snapshot.Components {
		matched := 0
		for _, file := range c.Files {
			if (file.Platform != "any" && file.Platform != platform) || (c.Kind == "profiles" && !strings.HasSuffix(file.Name, ".zip")) || (c.Kind != "profiles" && !strings.HasSuffix(file.Name, ".whl")) {
				continue
			}
			data, err := DownloadArtifact(file.URL, file.SHA256)
			if err != nil {
				return err
			}
			if c.Kind == "profiles" {
				profileResources, err = ProfileResources(data)
				if err != nil {
					return err
				}
				profileContract = profileResources["@project-profile-contracts.yaml"]
				delete(profileResources, "@project-profile-contracts.yaml")
				var registry struct {
					Current string `yaml:"current"`
				}
				if yaml.Unmarshal(profileContract, &registry) != nil || registry.Current != c.Version {
					return fmt.Errorf("Profile distribution contract version mismatch")
				}
				matched++
				continue
			}
			p := filepath.Join(dir, file.Name)
			if err := os.WriteFile(p, data, 0644); err != nil {
				return err
			}
			wheels = append(wheels, p)
			matched++
		}
		if matched != 1 {
			return fmt.Errorf("Package lacks one host wheel for %s", c.ID)
		}
	}
	if len(wheels) == 0 {
		return fmt.Errorf("Package has no installable wheels")
	}
	// Restore every component even after removal or a damaged same-version install.
	args := append([]string{"-m", "pip", "install", "--upgrade", "--force-reinstall"}, wheels...)
	cmd := exec.Command(python, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	// sysconfig belongs to the explicitly selected Python installation adapter.
	purelib, err := exec.Command(python, "-c", "import sysconfig; print(sysconfig.get_path('purelib'))").Output()
	if err != nil {
		return err
	}
	profileDir := filepath.Join(strings.TrimSpace(string(purelib)), "ptsip", "profiles")
	if !filepath.IsAbs(profileDir) {
		return fmt.Errorf("invalid target Python package directory")
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return err
	}
	for name, data := range profileResources {
		if err := os.WriteFile(filepath.Join(profileDir, name), data, 0644); err != nil {
			return err
		}
	}
	contractDir := filepath.Join(strings.TrimSpace(string(purelib)), "ptsip", "specdata")
	if err := os.MkdirAll(contractDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contractDir, "project-profile-contracts.yaml"), profileContract, 0644); err != nil {
		return err
	}
	for name, expected := range profileResources {
		actual, err := os.ReadFile(filepath.Join(profileDir, name))
		if err != nil || !bytes.Equal(actual, expected) {
			return fmt.Errorf("installed Profile catalog differs: %s", name)
		}
	}
	return VerifyInstalledComponents(snapshot, python)
}

// Python is the installation adapter only. Inventory, artifact selection and
// completion remain governed by the Go Package controller.
func VerifyInstalledComponents(snapshot Snapshot, python string) error {
	adapter := `import hashlib, importlib.metadata as metadata, json, pathlib, sys, sysconfig
snapshot = json.load(sys.stdin)
for component in snapshot['components']:
    if component['kind'] == 'profiles':
        continue
    project = 'PTSIP' if component['kind'] == 'tool' else 'ptsip-engine-' + component['id']
    expected = component['version']
    if component['kind'] == 'tool' and expected.startswith('T.'):
        from ptsip.constants import TOOL_VERSION
        if TOOL_VERSION != expected:
            raise RuntimeError('installed Tool identity differs')
        expected = expected[2:].rsplit('.', 1)
        expected = ''.join(expected)
        try:
            metadata.distribution('ptsip-tool')
            project = 'ptsip-tool'
        except metadata.PackageNotFoundError:
            pass
    distribution = metadata.distribution(project)
    if distribution.version != expected:
        raise RuntimeError('installed component version differs: ' + project)
    if component['kind'] == 'engine':
        origin = json.loads(distribution.read_text('ptsip-origin.json'))
        if origin['source_sha'] != component['source_sha'] or origin['engine_version'] != component['version']:
            raise RuntimeError('installed engine origin differs: ' + project)
        binary = pathlib.Path(sysconfig.get_path('scripts')) / origin['entrypoint']
        if not binary.is_file() or hashlib.sha256(binary.read_bytes()).hexdigest() != origin['binary_sha256']:
            raise RuntimeError('installed native executable missing or changed: ' + project)
print('Complete Package installed: Tool, Profiles and all registered native engines')
`
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	cmd := exec.Command(python, "-c", adapter)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func IdentityText(snapshot Snapshot, specRevision, specSource string) string {
	tool, profile := "UNAVAILABLE", "UNAVAILABLE"
	engines := []PublishedComponent{}
	for _, c := range snapshot.Components {
		switch c.Kind {
		case "tool":
			tool = c.Version
		case "profiles":
			profile = c.Version
		case "engine":
			engines = append(engines, c)
		}
	}
	text := fmt.Sprintf("PTSIP_Package: %s\nPTSIP_Tool: %s\nProject Profile: %s\nEngine:\n", snapshot.Version, tool, profile)
	for _, e := range engines {
		text += fmt.Sprintf("  %s: %s\n", strings.ToUpper(e.ID[:1])+e.ID[1:], e.Version)
	}
	return text
}
