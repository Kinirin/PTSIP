package testrepo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ConsumerCLI exercises the existing consumer product; it never imports or runs
// Python Developer Automation as an implementation of a Go control-plane test.
func ConsumerCLI(t *testing.T, args ...string) Object {
	t.Helper()
	root := Root(t)
	interpreter := os.Getenv("PTSIP_TEST_PYTHON")
	if interpreter == "" {
		candidate := filepath.Join(root, ".venv", "Scripts", "python.exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			interpreter = candidate
		} else {
			var err error
			interpreter, err = exec.LookPath("python")
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	command := exec.Command(interpreter, append([]string{"-m", "ptsip"}, args...)...)
	command.Dir = root
	for _, value := range os.Environ() {
		if !strings.EqualFold(strings.SplitN(value, "=", 2)[0], "PYTHONPATH") {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "PYTHONPATH="+root+string(os.PathListSeparator)+filepath.Join(root, "src"), "PYTHONUTF8=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("consumer CLI: %v\n%s", err, output)
	}
	var result Object
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("consumer CLI JSON: %v\n%s", err, output)
	}
	return result
}
