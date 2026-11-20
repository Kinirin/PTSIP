package contracts_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func profileConsumerPython(t *testing.T) string {
	t.Helper()
	if configured := os.Getenv("PTSIP_TEST_PYTHON"); configured != "" {
		if !filepath.IsAbs(configured) {
			t.Fatal("PTSIP_TEST_PYTHON must be absolute")
		}
		return configured
	}
	repository := filepath.Dir(filepath.Dir(productRoot(t)))
	venv := filepath.Join(repository, ".venv", "bin", "python")
	if runtime.GOOS == "windows" {
		venv = filepath.Join(repository, ".venv", "Scripts", "python.exe")
	}
	if info, err := os.Stat(venv); err == nil && info.Mode().IsRegular() {
		return venv
	}
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	t.Fatal("Python is required to exercise existing supported Python consumer bridges")
	return ""
}

func TestProfileNativeConsumerBridgeAndLazyIdentity(t *testing.T) {
	installed := installedProfileRuntime(t, "discovery")
	validation := installedProfileRuntime(t, "validation")
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	validationName := "ptsip-engine-validation" + suffix
	data, err := os.ReadFile(filepath.Join(validation.scripts, validationName))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installed.scripts, validationName), data, 0o755); err != nil {
		t.Fatal(err)
	}
	registryData, err := os.ReadFile(filepath.Join(productRoot(t), "specdata", "project-profile-contracts.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(installed.packageRoot, "specdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installed.packageRoot, "specdata", "project-profile-contracts.yaml"), registryData, 0o644); err != nil {
		t.Fatal(err)
	}
	var registry object
	if err := yaml.Unmarshal(registryData, &registry); err != nil {
		t.Fatal(err)
	}
	bridges, ok := registry["historical_bridges"].([]any)
	if !ok || len(bridges) == 0 {
		t.Fatal("distributed historical bridge evidence is required")
	}
	bridge := bridges[0].(map[string]any)
	transition := object{
		"source_contract_version": bridge["declared_version"],
		"source_revision":         bridge["specification_revision"],
		"target_contract_version": bridge["target_contract"],
		"transitions":             registry["transitions"],
		"historical_bridges":      bridges,
	}
	transitionJSON, err := json.Marshal(transition)
	if err != nil {
		t.Fatal(err)
	}
	script := `import importlib, importlib.util, json, os, pathlib, sys
from ptsip.ptsip_specification.spec_identity import (ProfileNativeError, SpecIdentity,
    current_spec_identity, native_profile_operation)
# Import is lazy; only constructing the default identity enters native Discovery.
identity = current_spec_identity().as_dict()
source = pathlib.Path(os.environ["PTSIP_TEST_PRODUCT_ROOT"]) / "migration/analysis/target/semantics.py"
spec = importlib.util.spec_from_file_location("ptsip_native_target_semantics", source)
module = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = module
spec.loader.exec_module(module)
semantics = module.default_target_semantics().as_dict()
transition = native_profile_operation("validation.profile.transition", json.loads(os.environ["PTSIP_TEST_TRANSITION"]))
configured = os.environ["PTSIP_GO_CLI"]
os.environ["PTSIP_GO_CLI"] = "relative-ptsip-go"
try:
    current_spec_identity()
except ProfileNativeError:
    rejected_relative = True
else:
    rejected_relative = False
finally:
    os.environ["PTSIP_GO_CLI"] = configured
try:
    importlib.import_module("ptsip.migration.execution.direct.promotion")
except ModuleNotFoundError as exc:
    promotion_import_blocker = exc.name
else:
    promotion_import_blocker = None
print(json.dumps({"identity":identity,"semantics":semantics,"transition":transition,
    "rejected_relative":rejected_relative,"promotion_import_blocker":promotion_import_blocker}, ensure_ascii=True))
`
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, profileConsumerPython(t), "-c", script)
	cmd.Env = append(os.Environ(),
		"PYTHONPATH="+filepath.Dir(productRoot(t)), "PYTHONNOUSERSITE=1", "PYTHONDONTWRITEBYTECODE=1",
		"PTSIP_GO_CLI="+installed.cli, "PTSIP_ENGINE_SCRIPTS="+installed.scripts,
		"PTSIP_PROFILE_PACKAGE_ROOT="+installed.packageRoot,
		"PTSIP_TEST_PRODUCT_ROOT="+productRoot(t), "PTSIP_TEST_TRANSITION="+string(transitionJSON),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("existing Python consumer bridge failed: %v\n%s", err, output)
	}
	var result object
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("consumer output is invalid JSON: %v\n%s", err, output)
	}
	if result["identity"].(map[string]any)["project_profile_contract_version"] != "pp.3.00" ||
		result["semantics"].(map[string]any)["draft_version"] != "pp.3.00" ||
		result["transition"].(map[string]any)["allowed"] != true || result["rejected_relative"] != true {
		t.Fatalf("native consumer result mismatch: %#v", result)
	}
	if blocker := result["promotion_import_blocker"]; blocker != nil {
		t.Logf("Normal migration package import remains independently blocked by absent %v; actual target-semantics file was exercised through importlib without replacement modules.", blocker)
	}
}
