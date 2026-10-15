package machine

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPPPreCommitCommandAdmissionDoesNotReadRetiredPolicy(t *testing.T) {
	repo := &Repository{Root: t.TempDir()}
	positional := []string{"pp", "pre-commit"}
	options := map[string]string{"--repository": repo.Root}
	if err := repo.NormalizeCommandOptions(positional, options, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.AdmitCommand(positional, options); err != nil {
		t.Fatal(err)
	}
	for status, expected := range map[string]int{"PASS": 0, "FAIL": 2, "BLOCKED": 2, "UNRESOLVED": 2} {
		code, err := repo.CommandStatusExitCode(positional, status)
		if err != nil || code != expected {
			t.Fatalf("status %s: exit=%d, error=%v", status, code, err)
		}
	}
	if _, err := repo.registeredCommand([]string{"pp", "delta"}); err == nil {
		t.Fatal("PP hook admission extended to another command without its contract")
	}
}

func TestPPPreCommitCommandRejectsInvalidArguments(t *testing.T) {
	repo := &Repository{Root: t.TempDir()}
	for _, request := range []struct {
		name        string
		positional  []string
		options     map[string]string
		repetitions map[string][]string
		message     string
	}{
		{"extra_positional", []string{"pp", "pre-commit", "extra"}, nil, nil, "argument count mismatch"},
		{"unknown_option", []string{"pp", "pre-commit"}, map[string]string{"--apply": "true"}, nil, "not registered"},
		{"missing_value", []string{"pp", "pre-commit"}, map[string]string{"--repository": ""}, nil, "requires a value"},
		{"duplicate_option", []string{"pp", "pre-commit"}, map[string]string{"--repository": repo.Root}, map[string][]string{"--repository": {repo.Root, repo.Root}}, "duplicate option"},
	} {
		t.Run(request.name, func(t *testing.T) {
			err := repo.NormalizeCommandOptions(request.positional, request.options, request.repetitions)
			if err == nil {
				err = repo.AdmitCommand(request.positional, request.options)
			}
			if err == nil || !strings.Contains(err.Error(), request.message) {
				t.Fatalf("invalid invocation admitted or wrong boundary: %v", err)
			}
		})
	}
}

func TestPPPreCommitCLIWithoutRetiredSchemaPreservesChecksAndIndex(t *testing.T) {
	repo, files := ppRetirementFixture(t)
	if err := repo.AtomicWrite("pyproject.toml", []byte("[project]\nname = 'pp-hook-fixture'\nversion = '0.0.0'\n"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo.Root, "developer/policy/schemas/root-family-policy.schema.json")); !os.IsNotExist(err) {
		t.Fatalf("fixture must not restore the retired schema: %v", err)
	}
	if err := repo.AtomicWrite("unrelated.txt", []byte("staged user change\n"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ppGit(repo.Root, "add", "--", "unrelated.txt"); err != nil {
		t.Fatal(err)
	}
	source, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	run := func(arguments ...string) ([]byte, error) {
		command := exec.Command("go", append([]string{"-C", filepath.Join(source.Root, "developer/automation"), "run", "-tags", "grammar_subset,grammar_subset_python", "./cmd/ptsip-dev"}, arguments...)...)
		return command.CombinedOutput()
	}
	snapshot := func() []byte {
		t.Helper()
		index, err := ppGit(repo.Root, "ls-files", "--stage", "-z")
		if err != nil {
			t.Fatal(err)
		}
		return index
	}
	before := snapshot()
	output, err := run("pp", "pre-commit", "--repository", repo.Root)
	if err != nil {
		t.Fatalf("hook rejected an unrelated staged change: %v\n%s", err, output)
	}
	var result Object
	if err := json.Unmarshal(output, &result); err != nil || result["status"] != "PASS" || result["classification"] != "NO_T2_AUTHORITY_DELTA" {
		t.Fatalf("unexpected PP result: %s, error=%v", output, err)
	}
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("unchanged PP hook altered the Git index")
	}

	invalid := append(append([]byte{}, files[PPEmbeddedRegistry]...), []byte("\n# mismatched embedded registry\n")...)
	if err := repo.AtomicWrite(PPEmbeddedRegistry, invalid, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ppGit(repo.Root, "add", "--", PPEmbeddedRegistry); err != nil {
		t.Fatal(err)
	}
	before = snapshot()
	output, err = run("pp", "pre-commit", "--repository", repo.Root)
	if err == nil || !strings.Contains(string(output), "PP_RUNTIME_REGISTRY_PROJECTION_MISMATCH") {
		t.Fatalf("invalid PP projection was not rejected: %v\n%s", err, output)
	}
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("rejected PP hook altered the Git index")
	}
	current, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(PPEmbeddedRegistry)))
	if err != nil || !bytes.Equal(current, invalid) {
		t.Fatal("rejected PP hook changed user content", err)
	}
	output, err = run("policy-validator", "validate", "--repository", repo.Root)
	if err == nil || !strings.Contains(string(output), "LEGACY_DEVELOPER_POLICY_CONTROL_PLANE_RETIRED") {
		t.Fatalf("retired policy validator was reactivated: %v\n%s", err, output)
	}
}
