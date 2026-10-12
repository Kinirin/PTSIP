package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

const CatalogPath = ".github/checks/release/release-contract.json"

type Catalog struct {
	Format                  string            `json:"format"`
	ApprovalRef             string            `json:"approval_ref"`
	PackageBase             string            `json:"package_base"`
	VersionRules            map[string]string `json:"version_rules"`
	TagPrefixes             map[string]string `json:"tag_prefixes"`
	Platforms               []string          `json:"platforms"`
	EngineIndex             string            `json:"engine_index"`
	EngineVersionFile       string            `json:"engine_version_file"`
	EngineVersionResolutionRef string         `json:"engine_version_resolution_ref"`
	EngineIDs               []string          `json:"engine_ids"`
	BootstrapEngineVersions map[string]string `json:"bootstrap_engine_versions"`
	Commands                []string          `json:"commands"`
}

type Component struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Version       string   `json:"version"`
	Tag           string   `json:"tag"`
	SourceSHA     string   `json:"source_sha"`
	Fingerprint   string   `json:"fingerprint"`
	State         string   `json:"state"`
	Script        string   `json:"script"`
	CargoManifest string   `json:"cargo_manifest,omitempty"`
	Manifest      string   `json:"manifest,omitempty"`
	Binding       string   `json:"binding,omitempty"`
	Roots         []string `json:"roots"`
	Platforms     []string `json:"platforms"`
}

type Selection struct {
	Format            string            `json:"format"`
	SourceSHA         string            `json:"source_sha"`
	PackageVersion    string            `json:"package_version"`
	CompositionSHA256 string            `json:"composition_sha256"`
	Versions          map[string]string `json:"versions"`
	Components        []Component       `json:"components"`
	PackageSelected   bool              `json:"package_selected"`
}

func loadCatalog(repo *machine.Repository) (Catalog, error) {
	var c Catalog
	owner, err := repo.RootSection("MPD-INFO-0001", "independent_release_versioning")
	if err != nil {
		return c, err
	}
	if machine.Map(owner)["contract_ref"] != CatalogPath {
		return c, fmt.Errorf("release catalog lacks its exact current INFO owner")
	}
	data, err := repo.Read(CatalogPath)
	if err != nil {
		return c, err
	}
	if err = repo.Validate(machine.Text(machine.Map(owner)["schema_ref"]), data); err != nil {
		return c, err
	}
	p, err := repo.Path(CatalogPath)
	if err != nil {
		return c, err
	}
	f, err := os.Open(p)
	if err != nil {
		return c, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	if c.Format != "ptsip-independent-release/v1" || c.EngineVersionFile != "version.yaml" || len(c.EngineIDs) != 7 || len(c.Platforms) == 0 {
		return c, fmt.Errorf("invalid admitted release catalog")
	}
	for _, pattern := range c.VersionRules {
		if _, err = regexp.Compile(pattern); err != nil {
			return c, err
		}
	}
    ownerFields := machine.Map(owner)
    if machine.Text(ownerFields["engine_public_version"]) != "MAJOR_MINOR_PATCH" ||
        machine.Text(ownerFields["engine_public_version_pattern"]) != c.VersionRules["engine"] ||
        machine.Text(ownerFields["engine_public_version_resolution_ref"]) != c.EngineVersionResolutionRef {
        return c, fmt.Errorf("engine release contract does not match current KEEP_LATEST policy")
    }
	return c, nil
}

func CargoVersion(public string) (string, error) {
	m := regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:(a|b|rc)([1-9][0-9]*))?$`).FindStringSubmatch(public)
	if m == nil {
		return "", fmt.Errorf("engine version must be major.minor with optional aN, bN or rcN: %s", public)
	}
	v := m[1] + "." + m[2] + ".0"
	if m[3] != "" {
		names := map[string]string{"a": "alpha", "b": "beta", "rc": "rc"}
		v += "-" + names[m[3]] + "." + m[4]
	}
	return v, nil
}

func PackageIdentity(base string, versions map[string]string) (string, string, error) {
	if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(base) {
		return "", "", fmt.Errorf("Package base requires an explicit major.minor.micro value")
	}
	if len(versions) != 9 {
		return "", "", fmt.Errorf("Package requires Tool, Profile and exactly seven engine versions")
	}
	keys := make([]string, 0, len(versions))
	for k, v := range versions {
		if v == "" {
			return "", "", fmt.Errorf("empty component version: %s", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vector := make([][2]string, 0, len(keys))
	for _, k := range keys {
		vector = append(vector, [2]string{k, versions[k]})
	}
	data, _ := json.Marshal(vector)
	sum := sha256.Sum256(data)
	full := hex.EncodeToString(sum[:])
	return base + "." + full[:6], full, nil
}

func selectionSet(text string, allowed map[string]bool) (map[string]bool, error) {
	out := map[string]bool{}
	if text == "all" {
		for k := range allowed {
			out[k] = true
		}
		return out, nil
	}
	if text == "" {
		return out, nil
	}
	for _, v := range strings.Split(text, ",") {
		v = strings.TrimSpace(v)
		if !allowed[v] || out[v] {
			return nil, fmt.Errorf("unknown or duplicated release target: %s", v)
		}
		out[v] = true
	}
	return out, nil
}

func componentFingerprint(repo *machine.Repository, sha string, c Component) (string, error) {
	args := append([]string{"ls-tree", "-r", "--full-tree", sha, "--"}, c.Roots...)
	text, err := requireRun(repo, "git", args...)
	if err != nil {
		return "", err
	}
	lines := []string{}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid tracked source closure")
		}
		p := parts[1]
		if c.ID == "tool" && (strings.HasPrefix(p, "src/ptsip/engines/") || (strings.HasPrefix(p, "src/ptsip/profiles/") && !strings.HasSuffix(p, ".py"))) {
			continue
		}
		if c.ID == "profile" && (strings.HasPrefix(p, "src/ptsip/profiles/history/") || strings.HasSuffix(p, ".py")) {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("empty tracked release source closure: %s", c.ID)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func Plan(repo *machine.Repository, targets, disabled, packageBase, repository, apiRoot string, offline bool) (Selection, error) {
	var out Selection
	catalog, err := loadCatalog(repo)
	if err != nil {
		return out, err
	}
	sha, err := requireRun(repo, "git", "rev-parse", "HEAD")
	if err != nil {
		return out, err
	}
	toolVersion, err := projectVersion(repo)
	if err != nil {
		return out, err
	}
	profile, err := repo.CurrentProfileContract()
	if err != nil {
		return out, err
	}
	profileVersion := machine.Text(profile["version"])
	if !regexp.MustCompile(catalog.VersionRules["tool"]).MatchString(toolVersion) || !regexp.MustCompile(catalog.VersionRules["profile"]).MatchString(profileVersion) {
		return out, fmt.Errorf("Tool or Profile version violates its independent contract")
	}
	out = Selection{Format: "ptsip-release-selection/v1", SourceSHA: sha, Versions: map[string]string{"tool": toolVersion, "profile": profileVersion}, Components: []Component{}}
	components := []Component{
		{ID: "tool", Kind: "TOOL", Version: toolVersion, Tag: catalog.TagPrefixes["tool"] + toolVersion, Script: "Tool_release.ps1", Roots: []string{"pyproject.toml", "MANIFEST.in", "setup.py", "README.md", "License-Authority", "src/ptsip", "src/vpms", "src/agent_contracts", "src/policy"}, Platforms: []string{"any"}},
		{ID: "profile", Kind: "PROFILE", Version: profileVersion, Tag: catalog.TagPrefixes["profile"] + profileVersion, Script: "Profile_release.ps1", Roots: []string{"src/ptsip/profiles", "registry/project-profile-contracts.yaml", machine.Text(profile["schema"])}, Platforms: []string{"any"}},
	}
	index, err := repo.Read(catalog.EngineIndex)
	if err != nil {
		return out, err
	}
	allowed := map[string]bool{"tool": true, "profile": true, "package": true}
	expected := map[string]bool{}
	for _, id := range catalog.EngineIDs {
		expected[id] = true
		allowed[id] = true
	}
	seen := map[string]bool{}
	for _, raw := range machine.List(index["official_engine_entries"]) {
		e := machine.Map(raw)
		id := strings.ToLower(machine.Text(e["kind"]))
		version := machine.Text(e["version"])
		canonical, canonicalErr := loadEngineVersion(repo, catalog, id)
		if canonicalErr != nil { return out, canonicalErr }
		if version != canonical { return out, fmt.Errorf("engine index version diverges from canonical version.yaml: %s", id) }
		if !expected[id] || seen[id] {
			return out, fmt.Errorf("unregistered or duplicate official engine: %s", id)
		}
		seen[id] = true
		if !regexp.MustCompile(catalog.VersionRules["engine"]).MatchString(version) && catalog.BootstrapEngineVersions[id] != version {
			return out, fmt.Errorf("invalid engine public version: %s", version)
		}
		cargo, manifest, binding := machine.Text(e["cargo_manifest_ref"]), machine.Text(e["manifest_ref"]), machine.Text(e["execution_binding_ref"])
		for _, ref := range []string{cargo, manifest, binding} {
			p, err := repo.Path(ref)
			if err != nil {
				return out, err
			}
			if _, err = os.Stat(p); err != nil {
				return out, err
			}
		}
		m, err := repo.Read(manifest)
		if err != nil {
			return out, err
		}
		if m["implementation_version"] != version || m["engine_id"] != e["engine_id"] {
			return out, fmt.Errorf("engine manifest identity mismatch: %s", id)
		}
		if err = verifyEngineVersionProjections(repo, catalog, id, machine.Text(e["engine_id"]), cargo, binding, version); err != nil { return out, err }
		shared := machine.Map(index["engine_schema_refs"])
		roots := []string{filepath.ToSlash(filepath.Dir(cargo)), machine.Text(index["transport_contract_ref"])}
		for _, key := range []string{"core-manifest", "capability-contract", "execution-binding"} {
			roots = append(roots, machine.Text(shared[key]))
		}
		components = append(components, Component{ID: id, Kind: "ENGINE", Version: version, Tag: catalog.TagPrefixes["engine"] + id + "-v" + version, Script: "engines_release/" + id + "_release.ps1", CargoManifest: cargo, Manifest: manifest, Binding: binding, Roots: roots, Platforms: catalog.Platforms})
		out.Versions[id] = version
	}
	if len(seen) != 7 {
		return out, fmt.Errorf("all seven registered engines are required")
	}
	selected, err := selectionSet(targets, allowed)
	if err != nil {
		return out, err
	}
	off, err := selectionSet(disabled, allowed)
	if err != nil {
		return out, err
	}
	out.PackageSelected = selected["package"] && !off["package"]
	if packageBase == "" {
		packageBase = catalog.PackageBase
	}
	if packageBase != "" {
		out.PackageVersion, out.CompositionSHA256, err = PackageIdentity(packageBase, out.Versions)
		if err != nil {
			return out, err
		}
	} else if out.PackageSelected {
		return out, fmt.Errorf("Package base has not been set by its owner; pass --package-base")
	}
	var client *githubClient
	if !offline {
		client, err = newGitHubClient(repository, apiRoot)
		if err != nil {
			return out, err
		}
	}
	for _, c := range components {
		c.SourceSHA = sha
		c.Fingerprint, err = componentFingerprint(repo, sha, c)
		if err != nil {
			return out, err
		}
		c.State = "UNSELECTED"
		if off[c.ID] {
			c.State = "DISABLED"
		} else if selected[c.ID] {
			c.State = "BUILD"
		}
		if client != nil && (c.State == "BUILD" || out.PackageSelected) {
			found, err := publishedComponent(client, repo, c)
			if err != nil {
				return out, err
			}
			if found {
				c.SourceSHA, err = requireRun(repo, "git", "rev-parse", c.Tag+"^{commit}")
				if err != nil {
					return out, err
				}
				if c.State != "DISABLED" {
					c.State = "UNCHANGED"
				}
			}
		}
		out.Components = append(out.Components, c)
	}
	return out, nil
}

func publishedComponent(client *githubClient, repo *machine.Repository, c Component) (bool, error) {
	var record struct {
		Draft bool   `json:"draft"`
		Tag   string `json:"tag_name"`
	}
	err := client.request("GET", "/repos/"+client.repository+"/releases/tags/"+c.Tag, nil, &record)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	if record.Draft {
		return false, fmt.Errorf("existing draft requires explicit completion: %s", c.Tag)
	}
	if record.Tag != c.Tag {
		return false, fmt.Errorf("published identity mismatch")
	}
	if _, err = requireRun(repo, "git", "fetch", "origin", "tag", c.Tag); err != nil {
		return false, err
	}
	previous, err := componentFingerprint(repo, c.Tag, c)
	if err != nil {
		return false, err
	}
	if previous != c.Fingerprint {
		return false, fmt.Errorf("component source changed without a new version: %s %s", c.ID, c.Version)
	}
	return true, nil
}
