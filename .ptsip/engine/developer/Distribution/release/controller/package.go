package releasecontroller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
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
		// The old member is immutable archive data only, never a local source lookup.
		isRegistry := f.Name == ".ptsip/contracts/release/Platform/Common/profile/project-profile-contracts.yaml" ||
			f.Name == ".ptsip/contracts/release/profile/project-profile-contracts.yaml"
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

type installedOrigin struct {
	Schema     string `json:"schema_version"`
	ID         string `json:"engine_id"`
	Version    string `json:"engine_version"`
	Target     string `json:"host_target"`
	Entrypoint string `json:"entrypoint"`
	Digest     string `json:"binary_sha256"`
	Source     string `json:"source_sha"`
	Permission *bool  `json:"execution_authorized"`
}

// Python locates the selected installation. Go verifies the native bundle.
func VerifyInstalledComponents(snapshot Snapshot, python string) error {
	locate := `import importlib.metadata as m, json, pathlib, sysconfig
from ptsip.constants import TOOL_VERSION
d = m.distribution('PTSIP')
files = [p for p in d.files or () if p.name == 'METADATA' and p.parent.name.endswith('.dist-info')]
if len(files) != 1:
    raise RuntimeError('Unambiguous installed PTSIP metadata required')
print(json.dumps(dict(package_root=str(pathlib.Path(d.locate_file('ptsip'))),
    metadata_root=str(pathlib.Path(d.locate_file(files[0])).parent),
    scripts_root=sysconfig.get_path('scripts'), package_version=d.version, tool_version=TOOL_VERSION)))
`
	cmd := exec.Command(python, "-c", locate)
	cmd.Stderr = os.Stderr
	raw, err := cmd.Output()
	if err != nil {
		return err
	}
	var location struct {
		Package  string `json:"package_root"`
		Metadata string `json:"metadata_root"`
		Scripts  string `json:"scripts_root"`
		Version  string `json:"package_version"`
		Tool     string `json:"tool_version"`
	}
	if err := json.Unmarshal(raw, &location); err != nil {
		return err
	}
	if !filepath.IsAbs(location.Package) || !filepath.IsAbs(location.Metadata) || !filepath.IsAbs(location.Scripts) ||
		filepath.Dir(location.Package) != filepath.Dir(location.Metadata) || location.Version == "" {
		return fmt.Errorf("invalid PTSIP installation location")
	}
	var index struct {
		Schema  string `json:"schema_version"`
		Entries []struct {
			Kind    string `json:"kind"`
			ID      string `json:"engine_id"`
			Version string `json:"version"`
			Binding string `json:"execution_binding_ref"`
		} `json:"official_engine_entries"`
	}
	if err := ReadJSON(location.Package, "engines/index.json", &index); err != nil {
		return err
	}
	var bundle struct {
		Schema     string                     `json:"schema_version"`
		Version    string                     `json:"package_version"`
		Platform   string                     `json:"platform"`
		Permission *bool                      `json:"execution_authorized"`
		Origins    map[string]installedOrigin `json:"engines"`
	}
	if err := ReadJSON(location.Metadata, "ptsip-engines.json", &bundle); err != nil {
		return err
	}
	platform, target, suffix := "", "", ""
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		platform, target, suffix = "windows-x64", "x86_64-pc-windows-msvc", ".exe"
	case "linux/amd64":
		platform, target = "linux-x64", "x86_64-unknown-linux-gnu"
	case "darwin/arm64":
		platform, target = "macos-arm64", "aarch64-apple-darwin"
	default:
		return fmt.Errorf("unsupported native Package host")
	}
	if index.Schema != "ptsip-official-engine-index/v1" ||
		bundle.Schema != "ptsip-bundled-engine-origins/v1" || bundle.Version != location.Version ||
		bundle.Platform != platform || bundle.Permission == nil || *bundle.Permission ||
		len(index.Entries) == 0 || len(index.Entries) != len(bundle.Origins) {
		return fmt.Errorf("inconsistent installed native engine inventory")
	}
	seen := make(map[string]installedOrigin)
	ids := make(map[string]bool)
	validSource := regexp.MustCompile(`^[a-f0-9]{40}$`)
	for _, entry := range index.Entries {
		key := strings.ToLower(entry.Kind)
		origin, ok := bundle.Origins[key]
		if !ok || key == "" || entry.ID == "" || entry.Version == "" || ids[entry.ID] || seen[key].ID != "" ||
			origin.Schema != "ptsip-independent-wheel-origin/v1" || origin.ID != entry.ID ||
			origin.Version != entry.Version || origin.Target != target || origin.Permission == nil ||
			*origin.Permission || !validSource.MatchString(origin.Source) {
			return fmt.Errorf("missing or inconsistent installed origin: %s", key)
		}
		ref, ok := strings.CutPrefix(entry.Binding, "src/ptsip/")
		if !ok || !strings.HasPrefix(ref, "engines/") {
			return fmt.Errorf("invalid installed engine binding: %s", key)
		}
		var binding struct {
			Rows []struct {
				ID         string `json:"engine_id"`
				Version    string `json:"engine_version"`
				Entrypoint string `json:"entrypoint"`
				Permission *bool  `json:"permission_grant"`
			} `json:"bindings"`
		}
		if err := ReadJSON(location.Package, ref, &binding); err != nil {
			return err
		}
		if len(binding.Rows) != 1 {
			return fmt.Errorf("ambiguous installed engine binding: %s", key)
		}
		row := binding.Rows[0]
		if row.ID != entry.ID || row.Version != entry.Version || row.Permission == nil || *row.Permission ||
			row.Entrypoint == "" || strings.ContainsAny(row.Entrypoint, `\/:`) ||
			origin.Entrypoint != row.Entrypoint+suffix {
			return fmt.Errorf("installed engine entrypoint differs: %s", key)
		}
		if err := verifyInstalledBinary(location.Scripts, origin); err != nil {
			return fmt.Errorf("installed engine %s: %w", key, err)
		}
		seen[key] = origin
		ids[entry.ID] = true
	}
	matched := make(map[string]bool)
	for _, component := range snapshot.Components {
		switch component.Kind {
		case "engine":
			origin, ok := seen[component.ID]
			if !ok || matched[component.ID] || origin.Source != component.SourceSHA || origin.Version != component.Version {
				return fmt.Errorf("installed native engine original source differs: %s", component.ID)
			}
			matched[component.ID] = true
		case "tool":
			if location.Tool != component.Version {
				return fmt.Errorf("integrated PTSIP Tool version differs")
			}
		}
	}
	if len(matched) != len(seen) {
		return fmt.Errorf("installed engine inventory differs from the Package snapshot")
	}
	return nil
}

func verifyInstalledBinary(scripts string, origin installedOrigin) error {
	if origin.Entrypoint == "." || origin.Entrypoint == ".." {
		return fmt.Errorf("invalid native executable name")
	}
	path := filepath.Join(scripts, origin.Entrypoint)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("native executable missing or is an alias")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("native executable permission is missing")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != origin.Digest {
		return fmt.Errorf("native executable digest differs")
	}
	return nil
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
