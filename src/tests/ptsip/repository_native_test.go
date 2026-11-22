package contracts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func repositoryFixture(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func writeRepositoryFixture(t *testing.T, root, path, data string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repositoryOperation(t *testing.T, kind, operation string, input object) object {
	t.Helper()
	result, err := runProfileOperation(t, kind, operation, input)
	if err != nil || result["status"] != "PASS" {
		t.Fatalf("%s failed: %v %v", operation, result, err)
	}
	return result["output"].(object)
}

func TestRepositoryNativeGitObservation(t *testing.T) {
	root := repositoryFixture(t)
	git := func(args ...string) string {
		t.Helper()
		base := []string{"-C", root, "-c", "user.name=PTSIP test", "-c", "user.email=ptsip@example.test", "-c", "commit.gpgSign=false", "-c", "core.autocrlf=false"}
		output, err := exec.Command("git", append(base, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	git("init", "--initial-branch=main", "--template=")
	git("remote", "add", "origin", "https://github.com/example/repository.git")
	writeRepositoryFixture(t, root, "src/tracked.bin", "original\x00bytes")
	git("add", "src/tracked.bin")
	git("commit", "-m", "fixture")
	head := git("rev-parse", "HEAD")
	observe := func() object {
		return repositoryOperation(t, "context", "context.repository.snapshot", object{"root": root})
	}
	inspect := func(path string) object {
		return repositoryOperation(t, "discovery", "discovery.repository.inspect", object{"path": path})
	}
	compare := func(before, after object) object {
		return repositoryOperation(t, "validation", "validation.repository.snapshot", object{"before": before, "after": after})
	}
	first := observe()
	if first["is_git"] != true || first["head"] != head || first["tracked_files"] != float64(1) || len(first["observation_errors"].(array)) != 0 {
		t.Fatalf("Git snapshot mismatch: %v", first)
	}
	if _, err := time.Parse(time.RFC3339, first["captured_at"].(string)); err != nil {
		t.Fatal(err)
	}
	clean := inspect(root)
	if clean["commit"] != head || clean["branch"] != "main" || clean["dirty"] != false ||
		clean["remote"].(object)["repository"] != "example/repository" ||
		!reflect.DeepEqual(clean["files"], array{"src/tracked.bin"}) {
		t.Fatalf("Git discovery mismatch: %v", clean)
	}
	subroot := inspect(filepath.Join(root, "src"))
	if subroot["root"] != clean["root"] || !reflect.DeepEqual(subroot["files"], array{"tracked.bin"}) {
		t.Fatalf("selected subdirectory inventory mismatch: %v", subroot)
	}
	if result := compare(first, observe()); result["stable"] != true {
		t.Fatalf("unchanged observation invalidated: %v", result)
	}
	writeRepositoryFixture(t, root, "untracked.txt", "new file")
	statusOnly := observe()
	if statusOnly["status_fingerprint"] == first["status_fingerprint"] || statusOnly["tracked_content_fingerprint"] != first["tracked_content_fingerprint"] || inspect(root)["dirty"] != true {
		t.Fatalf("untracked status was not separated from tracked content: %v", statusOnly)
	}
	writeRepositoryFixture(t, root, "src/tracked.bin", "changed\x00bytes")
	modified := observe()
	if modified["tracked_content_fingerprint"] == statusOnly["tracked_content_fingerprint"] || compare(statusOnly, modified)["stable"] != false {
		t.Fatalf("tracked content change was not detected: %v", modified)
	}
	git("add", "src/tracked.bin")
	git("commit", "-m", "second fixture revision")
	if result := compare(modified, observe()); result["stable"] != false || !strings.Contains(strings.Join(valuesAsStrings(result["reasons"].(array)), "\n"), "HEAD changed") {
		t.Fatalf("HEAD change was not detected: %v", result)
	}
	modified["observation_errors"] = array{"test observation failure"}
	if result := compare(modified, modified); result["stable"] != false {
		t.Fatalf("incomplete observation was accepted: %v", result)
	}
}

func valuesAsStrings(values array) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.(string)
	}
	return result
}

func TestRepositoryNativeProfileBindings(t *testing.T) {
	root := repositoryFixture(t)
	source := func(version string) string {
		return "ptsip:\n  version: " + version + "\n  specification:\n    source: https://github.com/Kinirin/PTSIP\n    revision: " + strings.Repeat("a", 40) + "\n"
	}
	writeRepositoryFixture(t, root, "ptsip.yaml", source("0.3.4-draft"))
	writeRepositoryFixture(t, root, "ptsip_0.3.6.yaml", source("0.3.6-draft"))
	writeRepositoryFixture(t, root, "sub/ptsip_0.3.6.yaml", source("0.3.6-draft"))
	writeRepositoryFixture(t, root, "malformed.yaml", "ptsip: {}\nptsip: {}\n")
	inspect := repositoryOperation(t, "discovery", "discovery.repository.inspect", object{"path": root})
	if inspect["is_git"] != false || inspect["commit"] != nil || len(inspect["files"].(array)) != 4 {
		t.Fatalf("filesystem inventory mismatch: %v", inspect)
	}
	snapshot := repositoryOperation(t, "context", "context.repository.snapshot", object{"root": root})
	if snapshot["is_git"] != false || snapshot["head"] != nil || snapshot["tracked_files"] != float64(4) {
		t.Fatalf("filesystem snapshot mismatch: %v", snapshot)
	}
	resolve := repositoryOperation(t, "validation", "validation.profile.path", object{"root": root, "path": `./sub\ptsip_0.3.6.yaml`})
	if resolve["normalized_path"] != "sub/ptsip_0.3.6.yaml" || resolve["path_on_disk"] != filepath.Join(root, "sub", "ptsip_0.3.6.yaml") {
		t.Fatalf("relative path normalization mismatch: %v", resolve)
	}
	for _, path := range []string{"../outside.yaml", "C:/outside.yaml", "ptsip.yaml/descendant.yaml"} {
		result, err := runProfileOperation(t, "validation", "validation.profile.path", object{"root": root, "path": path})
		if err == nil || result["status"] != "FAIL" {
			t.Fatalf("invalid Profile path accepted: %s %v %v", path, result, err)
		}
	}
	t.Run("existing alias", func(t *testing.T) {
		if err := os.Symlink(filepath.Join(root, "sub"), filepath.Join(root, "alias")); err != nil {
			t.Skipf("directory symlink unavailable: %v", err)
		}
		defer os.Remove(filepath.Join(root, "alias"))
		result, err := runProfileOperation(t, "validation", "validation.profile.path", object{"root": root, "path": "alias/new.yaml"})
		if err == nil || result["status"] != "FAIL" {
			t.Fatalf("directory alias accepted: %v %v", result, err)
		}
	})
	observed := repositoryOperation(t, "discovery", "discovery.profile.candidates", object{"root": root})
	if len(observed["diagnostics"].(array)) != 0 || len(observed["candidates"].(array)) != 1 || observed["source"].(object)["content_sha256"] != fileHash(t, filepath.Join(root, "ptsip.yaml")) {
		t.Fatalf("candidate evidence mismatch: %v", observed)
	}
	input := object{"root": root, "source": observed["source"], "candidates": observed["candidates"], "snapshot": snapshot}
	state := repositoryOperation(t, "validation", "validation.profile.transition-state", input)
	if state["valid"] != true || state["state"].(object)["mode"] != "SIMPLE" || state["state"].(object)["final_point"].(object)["path"] != "ptsip_0.3.6.yaml" {
		t.Fatalf("historical draft transition mismatch: %v", state)
	}
	invalid := repositoryOperation(t, "discovery", "discovery.profile.candidates", object{"root": root, "candidate_paths": array{"malformed.yaml", "sub//ptsip_0.3.6.yaml"}})
	if len(invalid["diagnostics"].(array)) != 2 || len(invalid["candidates"].(array)) != 0 {
		t.Fatalf("ambiguous paths or malformed YAML accepted: %v", invalid)
	}
	duplicate := repositoryOperation(t, "discovery", "discovery.profile.candidates", object{"root": root, "candidate_paths": array{"ptsip_0.3.6.yaml", "sub/ptsip_0.3.6.yaml"}})
	input["candidates"] = duplicate["candidates"]
	state = repositoryOperation(t, "validation", "validation.profile.transition-state", input)
	if state["valid"] != false || state["diagnostics"].(array)[0].(object)["code"] != "DUPLICATE_TARGET_IDENTITY" {
		t.Fatalf("duplicate draft identities accepted: %v", state)
	}
	selected := repositoryOperation(t, "discovery", "discovery.profile.contract", object{"root": installedProfileRuntime(t, "discovery").packageRoot})
	writeRepositoryFixture(t, root, "ptsip.yaml", source("pp.2.00"))
	observed = repositoryOperation(t, "discovery", "discovery.profile.candidates", object{"root": root, "candidate_paths": array{}})
	convergence := object{"source": observed["source"], "candidates": array{}, "snapshot": repositoryOperation(t, "context", "context.repository.snapshot", object{"root": root}),
		"selected_contract": selected}
	state = repositoryOperation(t, "validation", "validation.profile.convergence", convergence)
	if state["valid"] != false || state["diagnostics"].(array)[0].(object)["code"] != "PP_CONVERGENCE_UNSUPPORTED_SOURCE_TARGET" {
		t.Fatalf("unregistered pp.2.00 to pp.3.00 transition accepted: %v", state)
	}
	currentProfile := source("pp.3.00")
	writeRepositoryFixture(t, root, "ptsip.yaml", currentProfile)
	observed = repositoryOperation(t, "discovery", "discovery.profile.candidates", object{"root": root, "candidate_paths": array{}})
	currentSnapshot := repositoryOperation(t, "context", "context.repository.snapshot", object{"root": root})
	convergence["source"], convergence["snapshot"] = observed["source"], currentSnapshot
	state = repositoryOperation(t, "validation", "validation.profile.convergence", convergence)
	if state["valid"] != true || state["state"].(object)["mode"] != "CURRENT" {
		t.Fatalf("current Profile was not recognized by native convergence: %v", state)
	}
	writeRepositoryFixture(t, root, "ptsip.yaml", currentProfile+"# changed after observation\n")
	after := repositoryOperation(t, "context", "context.repository.snapshot", object{"root": root})
	compared := repositoryOperation(t, "validation", "validation.repository.snapshot", object{"before": currentSnapshot, "after": after})
	if compared["stable"] != false || currentSnapshot["tracked_content_fingerprint"] == after["tracked_content_fingerprint"] {
		t.Fatalf("changed Profile left its previous convergence evidence valid: %v", compared)
	}
	oldTarget := repositoryOperation(t, "discovery", "discovery.profile.contract", object{"root": installedProfileRuntime(t, "discovery").packageRoot, "contract_version": "pp.2.00"})
	writeRepositoryFixture(t, root, "ptsip.yaml", source("pp.1.02"))
	observed = repositoryOperation(t, "discovery", "discovery.profile.candidates", object{"root": root, "candidate_paths": array{}})
	convergence["selected_contract"], convergence["source"] = oldTarget, observed["source"]
	convergence["snapshot"] = repositoryOperation(t, "context", "context.repository.snapshot", object{"root": root})
	state = repositoryOperation(t, "validation", "validation.profile.convergence", convergence)
	if state["valid"] != true || state["state"].(object)["mode"] != "IDENTITY_ONLY" {
		t.Fatalf("declared exact transition was not resolved: %v", state)
	}
	convergence["transitions"] = array{}
	result, err := runProfileOperation(t, "validation", "validation.profile.convergence", convergence)
	if err == nil || result["status"] != "FAIL" {
		t.Fatalf("partial explicit transition evidence accepted: %v %v", result, err)
	}
	convergence["historical_bridges"] = array{}
	state = repositoryOperation(t, "validation", "validation.profile.convergence", convergence)
	if state["valid"] != false {
		t.Fatalf("explicit empty evidence was replaced with the engine catalog: %v", state)
	}
}
