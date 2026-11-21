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
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	for _, kind := range []string{"context", "validation", "projection"} {
		owner := installedProfileRuntime(t, kind)
		name := "ptsip-engine-" + kind + suffix
		data, err := os.ReadFile(filepath.Join(owner.scripts, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(installed.scripts, name), data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	transition := object{
		"source_contract_version": "0.3.4-draft",
		"source_revision":         "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e",
		"target_contract_version": "pp.1.01",
	}
	transitionJSON, err := json.Marshal(transition)
	if err != nil {
		t.Fatal(err)
	}
	root := repositoryFixture(t)
	writeRepositoryFixture(t, root, "ptsip.yaml", `ptsip:
  version: 0.3.4-draft
  specification:
    source: https://github.com/Kinirin/PTSIP
    revision: b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e
policies:
  product_to_toolchain_runtime_dependency: deny
  toolchain_in_product_package: deny
  independent_build_resolution: required
components:
  - id: application
    classification: PRODUCT
    include: [src/**]
    purpose: example
`)
	writeRepositoryFixture(t, root, "src/example.txt", "covered source")
	script := `import json, os, pathlib, sys
from ptsip.evidence.contract import EvidenceEvaluationContext, NormalizedEvidenceSet, SnapshotBinding, SourceGenerationBinding
from ptsip.ptsip_specification.spec_identity import (ProfileNativeError, current_spec_identity,
    capture_snapshot, discover_direct_profile_convergence, discover_profile_transition, native_profile_operation)
configured = os.environ["PTSIP_GO_CLI"]
os.environ["PTSIP_GO_CLI"] = "relative-ptsip-go"
from ptsip.migration import analyze_direct_profile_convergence, analyze_source_migration, default_target_semantics, target_state_from_mapping
from ptsip.migration.execution.direct.promotion import build_legacy_target_identity_rewrite_plan
from ptsip.source_compat import read_source_profile
try:
    current_spec_identity()
except ProfileNativeError:
    rejected_relative = True
else:
    rejected_relative = False
finally:
    os.environ["PTSIP_GO_CLI"] = configured
identity = current_spec_identity().as_dict()
semantics = default_target_semantics().as_dict()
transition = native_profile_operation("validation.profile.transition", json.loads(os.environ["PTSIP_TEST_TRANSITION"]))
root = pathlib.Path(sys.argv[1])
legacy = discover_profile_transition(root)
assert legacy.valid, legacy.as_dict()
source = read_source_profile(root, legacy.state.canonical_source)
assert source.profile is not None and not source.issues, source
assert source.profile.components[0].id == "application"
generation = source.profile.generation
snapshot = capture_snapshot(root)
evidence = NormalizedEvidenceSet(EvidenceEvaluationContext("fixture",
    SnapshotBinding(str(root), snapshot.head, snapshot.status_fingerprint, snapshot.tracked_content_fingerprint),
    SourceGenerationBinding(generation.profile_path, generation.declared_version,
        generation.specification_revision, generation.content_sha256)), (), ())
target_state = target_state_from_mapping({"ptsip": {"version": "pp.3.00", "specification": {"revision": "a" * 40}},
    "responsibility_map": {"mode": "DEFAULT"}, "components": [
        {"id": "application", "classification": "PRODUCT", "include": ["src/**"]}]})
source_analysis = analyze_source_migration(root, source.profile, evidence, target_state=target_state)
assert source_analysis.valid, source_analysis.as_dict()
assert len(source_analysis.required) == 1 and source_analysis.required[0].path == "src/example.txt", source_analysis.as_dict()
assert source_analysis.required[0].source_declaration_id == "application"
assert source_analysis.required[0].resolved, source_analysis.as_dict()
assert "evidence" not in source_analysis.required[0].as_dict()
profile = root / "ptsip.yaml"
profile.write_text("ptsip:\n  version: pp.3.00\n  specification:\n    revision: " + "a" * 40 + "\n", encoding="utf-8")
current = discover_direct_profile_convergence(root)
assert current.valid, current.as_dict()
analysis = analyze_direct_profile_convergence(root, current.state)
assert analysis.valid, analysis.as_dict()
profile.write_text(profile.read_text(encoding="utf-8") + "# changed after observation\n", encoding="utf-8")
stale = analyze_direct_profile_convergence(root, current.state)
assert not stale.valid and stale.issues[0].code == "STALE_DIRECT_CONVERGENCE_SNAPSHOT", stale.as_dict()
print(json.dumps({"identity":identity,"semantics":semantics,"transition":transition,
    "rejected_relative":rejected_relative,"mode":analysis.mode.value}, ensure_ascii=True))
`
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, profileConsumerPython(t), "-c", script, root)
	cmd.Env = append(os.Environ(),
		"PYTHONPATH="+filepath.Dir(productRoot(t)), "PYTHONNOUSERSITE=1", "PYTHONDONTWRITEBYTECODE=1",
		"PTSIP_GO_CLI="+installed.cli, "PTSIP_ENGINE_SCRIPTS="+installed.scripts,
		"PTSIP_PROFILE_PACKAGE_ROOT="+installed.packageRoot,
		"PTSIP_TEST_TRANSITION="+string(transitionJSON),
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
		result["transition"].(map[string]any)["allowed"] != true || result["rejected_relative"] != true || result["mode"] != "CURRENT" {
		t.Fatalf("native consumer result mismatch: %#v", result)
	}
}
