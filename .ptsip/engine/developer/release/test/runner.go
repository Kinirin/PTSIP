package releasetest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
	"go.yaml.in/yaml/v3"
)

type TestContract struct {
	Schema  string            `json:"schema_version"`
	Runners map[string]string `json:"runners"`
	Runtime struct {
		Python     string `json:"python_version"`
		FetchDepth int    `json:"git_fetch_depth"`
	} `json:"runtime"`
}
type Execution struct {
	Selection release.Selection `json:"selection"`
	Matrix    struct {
		Include []release.MatrixEntry `json:"include"`
	} `json:"matrix"`
	Python     string `json:"python"`
	FetchDepth int    `json:"git_fetch_depth"`
	Join       struct {
		Include []release.MatrixEntry `json:"include"`
	} `json:"join_matrix"`
}
type Receipt struct {
	Schema       string         `json:"schema_version"`
	SourceSHA    string         `json:"source_sha"`
	SelectionSHA string         `json:"selection_sha256"`
	Component    string         `json:"component"`
	Kind         string         `json:"kind"`
	Version      string         `json:"version"`
	Platform     string         `json:"platform"`
	Status       string         `json:"status"`
	Files        []release.File `json:"files"`
}

func Plan(root, mode, targets, optional string) (Execution, error) {
	var result Execution
	data, err := os.ReadFile(filepath.Join(root, ".github/test_modes.yaml"))
	if err != nil {
		return result, err
	}
	var registry struct {
		Defaults struct {
			Mode    string `yaml:"workflow_mode"`
			Targets string `yaml:"release_targets"`
		} `yaml:"defaults"`
		Modes []struct {
			ID        string `yaml:"id"`
			Execution struct {
				Contract string `yaml:"contract_ref"`
				Runner   string `yaml:"runner_ref"`
			} `yaml:"execution"`
		} `yaml:"modes"`
	}
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return result, err
	}
	if mode == "" {
		mode = registry.Defaults.Mode
	}
	if targets == "" {
		targets = registry.Defaults.Targets
	}
	contractRef, runnerRef := "", ""
	for _, binding := range registry.Modes {
		if binding.ID == mode {
			if contractRef != "" {
				return result, fmt.Errorf("duplicate registered mode")
			}
			contractRef, runnerRef = binding.Execution.Contract, binding.Execution.Runner
		}
	}
	if contractRef == "" || runnerRef != ".ptsip/engine/developer/release/test/cmd/ptsip-release-test" {
		return result, fmt.Errorf("mode is not registered with this runner")
	}
	if !strings.HasPrefix(contractRef, ".ptsip/engine/developer/release/test/") {
		return result, fmt.Errorf("verification contract outside admitted test root")
	}
	var contract TestContract
	if err := release.ReadJSON(root, contractRef, &contract); err != nil {
		return result, err
	}
	if contract.Schema != "ptsip-release-test/v1" {
		return result, fmt.Errorf("verification contract not admitted")
	}
	policyData, err := release.Run(root, "go", "-C", "developer/automation", "run", "-tags", "grammar_subset,grammar_subset_python", "./cmd/ptsip-dev", "policy-resolver", "get", "MPD-ASSURE-0001", "--section", "release_mode_verification")
	if err != nil {
		return result, err
	}
	var policy struct {
		Record map[string]any `json:"record"`
	}
	if json.Unmarshal(policyData, &policy) != nil || policy.Record["contract_ref"] != contractRef || policy.Record["execution_workflow"] != ".github/workflows/tooling-test.yml" || policy.Record["mode_registry"] != ".github/test_modes.yaml" || policy.Record["release_test_change_root"] != ".ptsip/engine/developer/release/test/" {
		return result, fmt.Errorf("registered release mode lacks canonical ASSURE admission")
	}
	result.Selection, err = release.Select(root, targets, optional)
	if err != nil {
		return result, err
	}
	result.Matrix.Include, err = release.Matrix(result.Selection, contract.Runners)
	result.Join.Include = []release.MatrixEntry{}
	c, e := release.ReadComposition(root)
	if e != nil {
		return result, e
	}
	if c.PackagePyPI != "" && result.Selection.Type == "PTSIP_PACKAGE" {
		for _, p := range result.Selection.Platforms {
			result.Join.Include = append(result.Join.Include, release.MatrixEntry{Component: "package", Kind: "package", Version: c.PackagePyPI, Platform: p, Runner: contract.Runners[p]})
		}
	}
	result.Python = contract.Runtime.Python
	result.FetchDepth = contract.Runtime.FetchDepth
	return result, err
}

func command(root string, environment []string, argv ...string) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), environment...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Validate(root string) error {
	return command(root, nil, "go", "-C", ".ptsip/engine/developer/release/test", "test", "./...")
}

func Execute(root string, selection release.Selection, id, platform string) (Receipt, error) {
	if err := release.ValidateSelection(root, selection); err != nil {
		return Receipt{}, err
	}
	head, err := release.Run(root, "git", "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != selection.SourceSHA {
		return Receipt{}, fmt.Errorf("test checkout differs from selected source")
	}
	tracked, err := release.Run(root, "git", "status", "--porcelain", "--untracked-files=no")
	if err != nil || len(tracked) != 0 {
		return Receipt{}, fmt.Errorf("tracked source changed before verification")
	}
	var result Receipt
	var c *release.Component
	for i := range selection.Components {
		if selection.Components[i].ID == id {
			c = &selection.Components[i]
		}
	}
	if c == nil {
		return result, fmt.Errorf("component outside selected release")
	}
	allowed := false
	for _, p := range selection.Platforms {
		allowed = allowed || p == platform
	}
	if !allowed {
		return result, fmt.Errorf("platform not selected")
	}
	out := filepath.Join(root, ".artifacts/release/components", id, platform)
	if err := os.MkdirAll(out, 0755); err != nil {
		return result, err
	}
	result = Receipt{Schema: "ptsip-release-test-receipt/v1", SourceSHA: selection.SourceSHA, SelectionSHA: selection.SelectionSHA, Component: id, Kind: c.Kind, Version: c.Version, Platform: platform, Status: "FAIL", Files: []release.File{}}
	switch c.Kind {
	case "engine":
		target, suffix := "x86_64-unknown-linux-gnu", ""
		if platform == "windows-x64" {
			target, suffix = "x86_64-pc-windows-msvc", ".exe"
		}
		if platform == "macos-arm64" {
			target = "aarch64-apple-darwin"
		}
		buildRoot := filepath.Join(root, ".artifacts/native-engine-test-target", id, platform)
		env := []string{"CARGO_TARGET_DIR=" + buildRoot}
		manifest := "src/ptsip/engines/" + id + "/Cargo.toml"
		if err := command(root, env, "cargo", "test", "--locked", "--manifest-path", manifest); err != nil {
			return result, err
		}
		if err := command(root, env, "cargo", "build", "--release", "--locked", "--manifest-path", manifest); err != nil {
			return result, err
		}
		binary := filepath.Join(buildRoot, "release", "ptsip-engine-"+id+suffix)
		if platform == "linux-x64" {
			if err := command(root, nil, "go", "-C", ".ptsip/engine/developer/release/engine-verification", "run", "./cmd/engine-wheel", "abi-check", "--binary", binary); err != nil {
				return result, err
			}
		}
		if err := command(root, nil, "go", "-C", ".ptsip/engine/developer/release/engine-verification", "run", "./cmd/engine-wheel", "--root", root, "--kind", id, "--target", target, "--binary", binary, "--output", out, "--source-sha", selection.SourceSHA); err != nil {
			return result, err
		}
		if err := command(root, []string{"PTSIP_ENGINE_WHEEL_DIR=" + out}, "go", "-C", "developer/tests", "test", "./engines", "-run", "^TestIndependentEngineWheel/"+id+"$", "-count=1", "-timeout=15m"); err != nil {
			return result, err
		}
	case "tool":
		stage, cleanup, err := toolSource(root, selection.SourceSHA)
		if err != nil {
			return result, err
		}
		defer cleanup()
		if err := release.MaterializeToolProject(stage); err != nil {
			return result, err
		}
		if err := command(root, nil, "python", "-m", "pip", "install", "build>=1.2,<2", "twine>=6,<8", "PyYAML>=6,<7", "jsonschema>=4.23,<5"); err != nil {
			return result, err
		}
		if err := command(stage, nil, "python", "-m", "build", "--outdir", out); err != nil {
			return result, err
		}
		entries, err := os.ReadDir(out)
		if err != nil {
			return result, err
		}
		args := []string{"python", "-m", "twine", "check"}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".whl") || strings.HasSuffix(e.Name(), ".tar.gz") {
				args = append(args, filepath.Join(out, e.Name()))
			}
		}
		if err := command(root, nil, args...); err != nil {
			return result, err
		}
		if err := verifyTool(root, out, c.Version, selection.SourceSHA); err != nil {
			return result, err
		}
	case "profiles":
		if err := command(root, nil, "go", "-C", "developer/tests", "test", "./pp", "-count=1"); err != nil {
			return result, err
		}
		archive := filepath.Join(out, "profiles-"+c.Version+".zip")
		files, err := release.Run(root, "git", "ls-tree", "-r", "--name-only", selection.SourceSHA, "src/ptsip/profiles", "registry/project-profile-contracts.yaml")
		if err != nil {
			return result, err
		}
		args := []string{"git", "archive", "--format=zip", "--output", archive, selection.SourceSHA, "registry/project-profile-contracts.yaml"}
		for _, ref := range strings.Fields(string(files)) {
			name := strings.TrimPrefix(ref, "src/ptsip/profiles/")
			if filepath.Base(name) == name && (name == "index.yaml" || strings.HasSuffix(name, ".ptsip.yaml")) {
				args = append(args, ref)
			}
		}
		if err := command(root, nil, args...); err != nil {
			return result, err
		}
		b, err := os.ReadFile(archive)
		if err != nil {
			return result, err
		}
		if _, err := release.ProfileResources(b); err != nil {
			return result, err
		}
	default:
		return result, fmt.Errorf("unregistered component test kind")
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "test-receipt.json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			return result, err
		}
		result.Files = append(result.Files, release.File{Name: name, SHA256: release.Digest(data), Platform: platform})
	}
	if len(result.Files) == 0 {
		return result, fmt.Errorf("no verified artifacts were produced")
	}
	result.Status = "PASS"
	if err := release.WriteJSON(filepath.Join(out, "test-receipt.json"), result); err != nil {
		return result, err
	}
	return result, nil
}

func Environment(root, id, platform string) (string, string, error) {
	paths := []string{filepath.ToSlash(filepath.Join(root, ".artifacts/native-engine-test-target", id, platform))}
	if id == "tool" || id == "profiles" {
		return "release-adapter-v1-" + platform + "-" + id, strings.Join(paths, "\n"), nil
	}
	if cargoHome := os.Getenv("CARGO_HOME"); cargoHome != "" {
		paths = append(paths, filepath.ToSlash(filepath.Join(cargoHome, "registry")), filepath.ToSlash(filepath.Join(cargoHome, "git")))
	} else if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.ToSlash(filepath.Join(home, ".cargo/registry")), filepath.ToSlash(filepath.Join(home, ".cargo/git")))
	}
	compiler, err := release.Run(root, "rustc", "-vV")
	if err != nil {
		return "", "", err
	}
	data := []byte(platform + "\n" + id + "\n" + string(compiler))
	for _, ref := range []string{"src/ptsip/engines/" + id + "/Cargo.lock", "src/ptsip/engines/" + id + "/Cargo.toml"} {
		p, err := release.SafePath(root, ref)
		if err == nil {
			if b, err := os.ReadFile(p); err == nil {
				data = append(data, b...)
			}
		}
	}
	head, err := release.Run(root, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	return "release-test-v2-" + platform + "-" + id + "-" + release.Digest(data) + "-" + strings.TrimSpace(string(head)), strings.Join(paths, "\n"), nil
}

func ReadSelection(root, ref string) (release.Selection, error) {
	var s release.Selection
	err := release.ReadJSON(root, ref, &s)
	return s, err
}
func Encode(v any) string { b, _ := json.Marshal(v); return string(b) }
