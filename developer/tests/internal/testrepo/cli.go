package testrepo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func BuildCLI(t *testing.T) string {
	t.Helper()
	name := "ptsip-dev"
	if os.PathSeparator == '\\' {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	command := exec.Command("go", "-C", filepath.Join(Root(t), "developer/automation"), "build", "-tags", "grammar_subset,grammar_subset_python", "-o", binary, "./cmd/ptsip-dev")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("native CLI build: %v\n%s", err, output)
	}
	return binary
}

func CLI(t *testing.T, binary, root string, args ...string) (Object, error, string) {
	t.Helper()
	command := exec.Command(binary, append([]string{"--repository", root}, args...)...)
	for _, value := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(value, "=", 2)[0])
		if key != "PATH" && key != "PYTHONPATH" && key != "PYTHONHOME" {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "PATH=", "PYTHONPATH=", "PYTHONHOME=")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, err, string(output)
	}
	var result Object
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("native CLI JSON: %v\n%s", err, output)
	}
	return result, nil, string(output)
}

func ReadJSON(t *testing.T, r *Repository, ref string) Object {
	t.Helper()
	path, err := r.Path(ref)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value Object
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func WriteJSON(t *testing.T, r *Repository, ref string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.AtomicWrite(ref, append(raw, '\n'), nil); err != nil {
		t.Fatal(err)
	}
}
