package module_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func developerCLI(t *testing.T) string {
	t.Helper()
	name := "developer"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	command := exec.Command("go", "-C", filepath.Join(testrepo.Root(t), ".ptsip/engine/developer"), "build", "-mod=readonly", "-o", binary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build developer validator: %v\n%s", err, output)
	}
	return binary
}

func runDeveloper(t *testing.T, binary, root, validator string) (map[string]any, int) {
	t.Helper()
	command := exec.Command(binary, "--root", root, "--validator", validator)
	output, err := command.Output()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("developer validator did not return JSON: %v\n%s", err, output)
	}
	return report, code
}

func TestModuleIndexesAndEntrypointStayInSync(t *testing.T) {
	root := testrepo.Root(t)
	binary := developerCLI(t)
	report, code := runDeveloper(t, binary, root, "developer-layout-check")
	if code != 0 {
		t.Fatalf("invalid module layout: %v", report)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".ptsip/engine/developer/index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Validators map[string]string `json:"validators"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	listed, code := runDeveloper(t, binary, root, "__unregistered__")
	names, ok := listed["available"].([]any)
	if code != 2 || listed["reason"] != "UNREGISTERED_DEVELOPER_VALIDATOR" || !ok {
		t.Fatalf("developer validator did not report executable registrations: %+v", listed)
	}
	if len(names) != len(index.Validators) {
		t.Fatal("validator index and executable registry differ")
	}
	seen := map[string]bool{}
	for _, value := range names {
		name, ok := value.(string)
		if !ok || seen[name] || index.Validators[name] == "" {
			t.Fatalf("validator missing from module index: %s", name)
		}
		seen[name] = true
	}
}

func TestLayoutRejectsMissingOrAuthorityGrantingIndex(t *testing.T) {
	root := t.TempDir()
	binary := developerCLI(t)
	if _, code := runDeveloper(t, binary, root, "developer-layout-check"); code == 0 {
		t.Fatal("missing module index accepted")
	}
	raw, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".ptsip/engine/developer/index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]any
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	index["execution_authorized"] = true
	filename := filepath.Join(root, ".ptsip/engine/developer/index.json")
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, raw, 0644); err != nil {
		t.Fatal(err)
	}
	report, code := runDeveloper(t, binary, root, "developer-layout-check")
	if code == 0 || report["problem"] != "TEMPORARY_VALIDATOR_BOUNDARY_INVALID" {
		t.Fatalf("invalid authority boundary accepted: %v", report)
	}
}
