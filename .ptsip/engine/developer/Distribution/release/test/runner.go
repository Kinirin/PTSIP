package releasetest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
	boundary "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"
	"go.yaml.in/yaml/v3"
)

type TestContract struct {
	Schema       string            `json:"schema_version"`
	Workflow     string            `json:"workflow"`
	ModeRegistry string            `json:"mode_registry"`
	TestRoot     string            `json:"test_change_root"`
	Production   string            `json:"production_contract"`
	Boundary     string            `json:"verification_boundary_ref"`
	Required     []string          `json:"required_platforms"`
	Publication  *bool             `json:"publication_operations"`
	Runners      map[string]string `json:"runners"`
	Runtime      struct {
		Python     string `json:"python_version"`
		FetchDepth int    `json:"git_fetch_depth"`
	} `json:"runtime"`
}
type Execution struct {
	// A successful test is not normative release or publication approval.
	PolicyAuthority       string            `json:"policy_authority"`
	PublicationAuthorized bool              `json:"publication_authorized"`
	VerificationScope     string            `json:"verification_scope"`
	Selection             release.Selection `json:"selection"`
	Matrix                struct {
		Include []release.MatrixEntry `json:"include"`
	} `json:"matrix"`
	Python     string `json:"python"`
	FetchDepth int    `json:"git_fetch_depth"`
	Join       struct {
		Include []release.MatrixEntry `json:"include"`
	} `json:"join_matrix"`
	Lightweight struct {
		Include []release.MatrixEntry `json:"include"`
	} `json:"lightweight_matrix"`
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
			ID           string `yaml:"id"`
			ComponentRef string `yaml:"component_ref"`
			Execution    struct {
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
			if binding.ComponentRef != "repository-release-verification" {
				return result, fmt.Errorf("registered mode has wrong verification component owner")
			}
			contractRef, runnerRef = binding.Execution.Contract, binding.Execution.Runner
		}
	}
	if contractRef == "" || runnerRef != ".ptsip/engine/developer/Distribution/release/test/cmd/ptsip-release-test" {
		return result, fmt.Errorf("mode is not registered with this runner")
	}
	if !strings.HasPrefix(contractRef, ".ptsip/contracts/release/") {
		return result, fmt.Errorf("verification contract outside admitted test root")
	}
	var contract TestContract
	if err := release.ReadJSON(root, contractRef, &contract); err != nil {
		return result, err
	}
	if contract.Schema != "ptsip-release-test/v1" {
		return result, fmt.Errorf("verification contract not admitted")
	}
	// The current contract permits source-bound test evidence only.
	if err := verifyNonNormativeTestRoute(root, contractRef, contract); err != nil {
		return result, err
	}
	result.PolicyAuthority = "NOT_ASSERTED"
	result.PublicationAuthorized = false
	result.VerificationScope = "SOURCE_BOUND_TEST_EVIDENCE_ONLY"
	result.Selection, err = release.Select(root, targets, optional)
	if err != nil {
		return result, err
	}
	result.Matrix.Include, err = release.Matrix(result.Selection, contract.Runners)
	result.Join.Include = []release.MatrixEntry{}
	result.Lightweight.Include = []release.MatrixEntry{}
	c, e := release.ReadComposition(root)
	if e != nil {
		return result, e
	}
	if c.PackagePyPI != "" && result.Selection.Type == "PTSIP_PACKAGE" {
		for _, p := range result.Selection.Platforms {
			entry := release.MatrixEntry{Component: "package", Kind: "package", Version: c.PackagePyPI, Platform: p, Runner: contract.Runners[p]}
			result.Join.Include = append(result.Join.Include, entry)
		}
	}
	result.Python = contract.Runtime.Python
	result.FetchDepth = contract.Runtime.FetchDepth
	return result, err
}

// Only the registered read-only route may run until normative authority
// exists. Every missing, changed or publication-granting input fails closed.
func verifyNonNormativeTestRoute(root, contractRef string, contract TestContract) error {
	if contract.Boundary != boundary.Path {
		return fmt.Errorf("verification boundary binding missing")
	}
	if err := boundary.CheckOperation(root, "SOURCE_BOUND_TEST_EVIDENCE"); err != nil {
		return err
	}
	if contractRef != ".ptsip/contracts/release/Platform/Common/verification-contract.json" ||
		contract.Workflow != ".github/workflows/tooling-test.yml" ||
		contract.ModeRegistry != ".github/test_modes.yaml" ||
		contract.TestRoot != ".ptsip/engine/developer/Distribution/release/test/" ||
		contract.Production != ".ptsip/contracts/release/Platform/Common/composition-contract.json" ||
		contract.Publication == nil || *contract.Publication ||
		len(contract.Required) != 2 ||
		contract.Required[0] != "windows-x64" ||
		contract.Required[1] != "linux-x64" ||
		contract.Runners["windows-x64"] != "windows-2025" ||
		contract.Runners["linux-x64"] != "ubuntu-24.04" {
		return fmt.Errorf("registered test-only verification contract differs or requests publication")
	}
	var gate struct {
		Schema      string `json:"schema_version"`
		Policy      string `json:"normative_policy_status"`
		Fallback    string `json:"old_root_policy_fallback"`
		Module      string `json:"verification_module"`
		Publication *bool  `json:"publication_authorized"`
		Boundary    string `json:"verification_boundary_ref"`
	}
	if err := release.ReadJSON(root, ".ptsip/contracts/release/Platform/Common/admission.json", &gate); err != nil {
		return fmt.Errorf("read-only release admission boundary missing: %w", err)
	}
	if gate.Schema != "ptsip-release-admission-direction-projection/v1" ||
		gate.Policy != "NOT_ACTIVE_NORMATIVE_POLICY" ||
		gate.Fallback != "FORBIDDEN" ||
		gate.Module != ".ptsip/engine/developer/Distribution/release/" ||
		gate.Publication == nil || *gate.Publication || gate.Boundary != boundary.Path {
		return fmt.Errorf("policy authority remains unresolved; test-only execution requires publication blocked")
	}
	return nil
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
	return command(root, nil, "go", "-C", ".ptsip/engine/developer/Distribution/release/test", "test", "./...")
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
			if err := command(root, nil, "go", "-C", ".ptsip/engine/developer/Distribution/release/verification/engine", "run", "./cmd/engine-wheel", "abi-check", "--binary", binary); err != nil {
				return result, err
			}
		}
		if err := command(root, nil, "go", "-C", ".ptsip/engine/developer/Distribution/release/verification/engine", "run", "./cmd/engine-wheel", "--root", root, "--kind", id, "--target", target, "--binary", binary, "--output", out, "--source-sha", selection.SourceSHA); err != nil {
			return result, err
		}
		if err := command(root, []string{"PTSIP_ENGINE_WHEEL_DIR=" + out}, "go", "-C", "developer/tests", "test", "./engines", "-run", "^TestIndependentEngineWheel/"+id+"$", "-count=1", "-timeout=15m"); err != nil {
			return result, err
		}
		if id == "discovery" {
			if err := command(root, nil, "python", "-m", "pip", "install", "--disable-pip-version-check", "PyYAML>=6,<7", "jsonschema>=4.23,<5"); err != nil {
				return result, err
			}
			if err := command(root, []string{"PTSIP_TEST_DISCOVERY_BINARY=" + binary}, "go", "-C", "developer/tests", "test", "./engines", "-run", "^(TestOfficialEngineEnvironmentDiagnostics|TestOfficialEngineCompatibilityCLIEncoding)$", "-count=1", "-timeout=5m"); err != nil {
				return result, err
			}
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
		if err := VerifyProjectProfiles(root, c.Version); err != nil {
			return result, fmt.Errorf("native Project Profile verification failed: %w", err)
		}
		archive := filepath.Join(out, "profiles-"+c.Version+".zip")
		files, err := release.Run(root, "git", "ls-tree", "-r", "--name-only", selection.SourceSHA, "src/ptsip/profiles", ".ptsip/contracts/release/Platform/Common/profile/project-profile-contracts.yaml")
		if err != nil {
			return result, err
		}
		args := []string{"git", "archive", "--format=zip", "--output", archive, selection.SourceSHA, ".ptsip/contracts/release/Platform/Common/profile/project-profile-contracts.yaml"}
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
