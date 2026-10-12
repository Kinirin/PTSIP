package release

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

type Artifact struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type Receipt struct {
	Format       string     `json:"format"`
	Component    string     `json:"component"`
	Version      string     `json:"version"`
	Tag          string     `json:"tag"`
	SourceSHA    string     `json:"source_sha"`
	Fingerprint  string     `json:"fingerprint"`
	Platform     string     `json:"platform"`
	Verification string     `json:"verification"`
	RunID        string     `json:"run_id,omitempty"`
	Artifacts    []Artifact `json:"artifacts"`
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
func digest(path string) (Artifact, error) {
	f, err := os.Open(path)
	if err != nil {
		return Artifact{}, err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil || !s.Mode().IsRegular() {
		return Artifact{}, fmt.Errorf("artifact must be a regular file: %s", path)
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return Artifact{}, err
	}
	return Artifact{Name: filepath.Base(path), SHA256: hex.EncodeToString(h.Sum(nil)), Size: s.Size()}, nil
}
func hostPlatform() string {
	return map[string]string{"linux/amd64": "linux-x64", "windows/amd64": "windows-x64", "darwin/arm64": "macos-arm64"}[runtime.GOOS+"/"+runtime.GOARCH]
}

func readSelection(repo *machine.Repository, path string) (Selection, error) {
	var s Selection
	p, err := repo.Path(path)
	if err != nil {
		return s, err
	}
	f, err := os.Open(p)
	if err != nil {
		return s, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err = d.Decode(&s); err != nil {
		return s, err
	}
	base := ""
	if s.PackageVersion != "" {
		v := strings.Split(s.PackageVersion, ".")
		if len(v) != 4 {
			return s, fmt.Errorf("invalid Package identity")
		}
		base = strings.Join(v[:3], ".")
	}
	expected, err := Plan(repo, "all", "", base, "", "", true)
	if err != nil {
		return s, err
	}
	if s.Format != expected.Format || s.SourceSHA != expected.SourceSHA || s.PackageVersion != expected.PackageVersion || s.CompositionSHA256 != expected.CompositionSHA256 || !reflect.DeepEqual(s.Versions, expected.Versions) || len(s.Components) != len(expected.Components) {
		return s, fmt.Errorf("release selection is stale or its composition has changed")
	}
	for i, c := range s.Components {
		want := expected.Components[i]
		state := c.State
		if c.SourceSHA != s.SourceSHA {
			tagSHA, e := requireRun(repo, "git", "rev-parse", c.Tag+"^{commit}")
			if e != nil || tagSHA != c.SourceSHA {
				return s, fmt.Errorf("reused source must match its immutable component tag")
			}
			fp, e := componentFingerprint(repo, tagSHA, c)
			if e != nil || fp != c.Fingerprint {
				return s, fmt.Errorf("reused source closure mismatch")
			}
		}
		c.State = want.State
		c.SourceSHA = want.SourceSHA
		if !reflect.DeepEqual(c, want) {
			return s, fmt.Errorf("release binding was modified: %s", c.ID)
		}
		if state != "BUILD" && state != "UNCHANGED" && state != "UNSELECTED" && state != "DISABLED" {
			return s, fmt.Errorf("invalid component selection state")
		}
	}
	return s, nil
}
func selectedComponent(s Selection, id string) (Component, error) {
	for _, c := range s.Components {
		if c.ID == id {
			return c, nil
		}
	}
	return Component{}, fmt.Errorf("unregistered release target: %s", id)
}

func zipFiles(repo *machine.Repository, path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	z := zip.NewWriter(f)
	keys := []string{}
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		source := files[name]
		info, err := os.Lstat(source)
		if err != nil || !info.Mode().IsRegular() {
			z.Close()
			f.Close()
			return fmt.Errorf("non-regular release content: %s", source)
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(name)
		h.Method = zip.Deflate
		h.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		w, err := z.CreateHeader(h)
		if err != nil {
			return err
		}
		r, err := os.Open(source)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		r.Close()
		if err != nil {
			return err
		}
	}
	if err = z.Close(); err != nil {
		return err
	}
	return f.Close()
}

func contractFiles(repo *machine.Repository, c Component) (map[string]string, error) {
	out := map[string]string{}
	root := filepath.ToSlash(filepath.Dir(c.CargoManifest))
	for _, ref := range c.Roots {
		path, err := repo.Path(ref)
		if err != nil {
			return nil, err
		}
		if ref != root {
			out["contracts/"+filepath.Base(ref)] = path
			continue
		}
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if d.Name() == "target" || d.Name() == "src" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".json") {
				rel, _ := filepath.Rel(path, p)
				prefix := "engine/"
				if ref != root {
					prefix = "contracts/"
				}
				out[prefix+filepath.ToSlash(rel)] = p
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func BuildComponent(repo *machine.Repository, selection, id, platform, output string) (Receipt, error) {
	var receipt Receipt
	s, err := readSelection(repo, selection)
	if err != nil {
		return receipt, err
	}
	c, err := selectedComponent(s, id)
	if err != nil {
		return receipt, err
	}
	if c.State != "BUILD" {
		return receipt, fmt.Errorf("component is not selected for a new build: %s (%s)", id, c.State)
	}
	if platform == "" {
		platform = "any"
		if c.Kind == "ENGINE" {
			platform = hostPlatform()
		}
	}
	allowed := false
	for _, p := range c.Platforms {
		if p == platform {
			allowed = true
		}
	}
	if !allowed {
		return receipt, fmt.Errorf("unsupported release platform: %s", platform)
	}
	if c.Kind == "ENGINE" && platform != hostPlatform() {
		return receipt, fmt.Errorf("native engine must be tested on the requested host platform")
	}
	dir, err := repo.Path(output)
	if err != nil {
		return receipt, err
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return receipt, err
	}
	files := map[string]string{}
	artifacts := []Artifact{}
	switch c.Kind {
	case "ENGINE":
		metadata, err := requireRun(repo, "cargo", "metadata", "--no-deps", "--locked", "--format-version", "1", "--manifest-path", c.CargoManifest)
		if err != nil {
			return receipt, err
		}
		var m struct {
			Target   string                           `json:"target_directory"`
			Packages []struct{ Name, Version string } `json:"packages"`
		}
		if err = json.Unmarshal([]byte(metadata), &m); err != nil {
			return receipt, err
		}
		if len(m.Packages) != 1 {
			return receipt, fmt.Errorf("engine must have an independent Cargo package")
		}
		want, e := CargoVersion(c.Version)
		if e != nil {
			catalog, _ := loadCatalog(repo)
			if catalog.BootstrapEngineVersions[id] != c.Version {
				return receipt, e
			}
			want = c.Version
		}
		if m.Packages[0].Version != want {
			return receipt, fmt.Errorf("Cargo version projection mismatch: %s != %s", m.Packages[0].Version, want)
		}
		for _, args := range [][]string{{"test", "--release", "--locked", "--manifest-path", c.CargoManifest}, {"build", "--release", "--locked", "--manifest-path", c.CargoManifest}} {
			if _, err = requireRun(repo, "cargo", args...); err != nil {
				return receipt, err
			}
		}
		b, err := repo.Read(c.Binding)
		if err != nil {
			return receipt, err
		}
		bindings := machine.List(b["bindings"])
		if len(bindings) != 1 {
			return receipt, fmt.Errorf("engine entrypoint binding must resolve once")
		}
		binding := machine.Map(bindings[0])
		binary := filepath.Join(m.Target, "release", machine.Text(binding["entrypoint"]))
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		compiled, err := requireRun(repo, binary, "--manifest")
		if err != nil {
			return receipt, err
		}
		var actual machine.Object
		if err = json.Unmarshal([]byte(compiled), &actual); err != nil {
			return receipt, err
		}
		manifest, err := repo.Read(c.Manifest)
		if err != nil {
			return receipt, err
		}
		if !reflect.DeepEqual(actual, manifest) {
			return receipt, fmt.Errorf("compiled engine manifest differs from its registered source")
		}
		if _, err = requireRun(repo, "go", "-C", "developer/tests", "test", "-tags", "grammar_subset,grammar_subset_python", "./engines", "-run", "^TestNativeIndependentRustEnginePackages/"+id+"$"); err != nil {
			return receipt, err
		}
		files, err = contractFiles(repo, c)
		if err != nil {
			return receipt, err
		}
		files[filepath.Base(binary)] = binary
	case "PROFILE":
		issues, err := repo.ProfileRegistryErrors()
		if err != nil {
			return receipt, err
		}
		if len(issues) > 0 {
			return receipt, fmt.Errorf("Profile contract validation: %v", issues)
		}
		paths, err := requireRun(repo, "git", append([]string{"ls-files", "--"}, c.Roots...)...)
		if err != nil {
			return receipt, err
		}
		for _, p := range strings.Split(paths, "\n") {
			if p == "" || strings.Contains(p, "/history/") || strings.HasSuffix(p, ".py") {
				continue
			}
			path, err := repo.Path(p)
			if err != nil {
				return receipt, err
			}
			files[p] = path
		}
	case "TOOL":
		if _, err = requireRun(repo, "python", "-m", "build", "--outdir", dir); err != nil {
			return receipt, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return receipt, err
		}
		args := []string{"-m", "twine", "check"}
		for _, f := range entries {
			if strings.HasSuffix(f.Name(), ".whl") || strings.HasSuffix(f.Name(), ".tar.gz") {
				a, err := digest(filepath.Join(dir, f.Name()))
				if err != nil {
					return receipt, err
				}
				artifacts = append(artifacts, a)
				args = append(args, filepath.Join(dir, f.Name()))
			}
		}
		if len(artifacts) != 2 {
			return receipt, fmt.Errorf("Tool requires exactly one wheel and sdist")
		}
		if _, err = requireRun(repo, "python", args...); err != nil {
			return receipt, err
		}
	default:
		return receipt, fmt.Errorf("unregistered component kind")
	}
	if c.Kind != "TOOL" {
		license, err := repo.Path("License-Authority/LICENSE.md")
		if err != nil {
			return receipt, err
		}
		files["LICENSE.md"] = license
		name := id + "-" + c.Version + "-" + platform + ".zip"
		if err = zipFiles(repo, filepath.Join(dir, name), files); err != nil {
			return receipt, err
		}
		a, err := digest(filepath.Join(dir, name))
		if err != nil {
			return receipt, err
		}
		artifacts = append(artifacts, a)
	}
	receipt = Receipt{Format: "ptsip-component-release-receipt/v1", Component: id, Version: c.Version, Tag: c.Tag, SourceSHA: c.SourceSHA, Fingerprint: c.Fingerprint, Platform: platform, Verification: "PASS", RunID: os.Getenv("GITHUB_RUN_ID"), Artifacts: artifacts}
	if err = writeJSON(filepath.Join(dir, id+"-"+c.Version+"-"+platform+".receipt.json"), receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func VerifyReceipts(s Selection, id, dir string, requireAll bool) ([]Receipt, error) {
	c, err := selectedComponent(s, id)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []Receipt{}
	platforms := map[string]bool{}
	allowedFiles := map[string]bool{}
	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".receipt.json") {
			continue
		}
		var r Receipt
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		if r.Component != id {
			continue
		}
		if r.Format != "ptsip-component-release-receipt/v1" || r.Verification != "PASS" || r.Version != c.Version || r.Tag != c.Tag || r.Fingerprint != c.Fingerprint || r.SourceSHA != c.SourceSHA || platforms[r.Platform] || len(r.Artifacts) == 0 {
			return nil, fmt.Errorf("stale, duplicated or invalid release receipt: %s", file.Name())
		}
		platforms[r.Platform] = true
		if file.Name() != id+"-"+c.Version+"-"+r.Platform+".receipt.json" {
			return nil, fmt.Errorf("receipt filename does not match its identity")
		}
		allowedFiles[file.Name()] = true
		for _, a := range r.Artifacts {
			if filepath.Base(a.Name) != a.Name || strings.ContainsAny(a.Name, "/\\") {
				return nil, fmt.Errorf("artifact path escapes receipt")
			}
			actual, err := digest(filepath.Join(dir, a.Name))
			if err != nil {
				return nil, err
			}
			if actual != a {
				return nil, fmt.Errorf("release artifact hash mismatch: %s", a.Name)
			}
			allowedFiles[a.Name] = true
		}
		out = append(out, r)
	}
	for _, file := range entries {
		if file.IsDir() || !allowedFiles[file.Name()] {
			return nil, fmt.Errorf("unverified content in component artifact directory: %s", file.Name())
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no verified artifacts for %s", id)
	}
	for p := range platforms {
		found := false
		for _, expected := range c.Platforms {
			found = found || p == expected
		}
		if !found {
			return nil, fmt.Errorf("unregistered artifact platform")
		}
	}
	if requireAll && len(platforms) != len(c.Platforms) {
		return nil, fmt.Errorf("component release missing required platforms: %s", id)
	}
	return out, nil
}

func RunCommand(repo *machine.Repository, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = repo.Root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
