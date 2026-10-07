package repository_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func repositoryText(t *testing.T, ref string) string {
	t.Helper()
	path, err := testrepo.Open(testrepo.Root(t)).Path(ref)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestRepositorySelfProfileIsExplicitAndHasNoRootBridge(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	path, err := repo.Path("ptsip.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("retired root bridge exists", err)
	}
	validation := testrepo.ConsumerCLI(t, "validate", ".", "--profile", ".ptsip/profiles/main.ptsip.yaml", "--json")
	if validation["valid"] != true {
		t.Fatal(validation)
	}
	for _, field := range []string{"errors", "warnings"} {
		if len(validation[field].([]any)) != 0 {
			t.Fatal(validation)
		}
	}
	clarification := testrepo.ConsumerCLI(t, "clarify", ".", "--profile", ".ptsip/profiles/main.ptsip.yaml", "--json")
	if clarification["status"] != "NO_CLARIFICATION_REQUIRED" || len(clarification["requests"].([]any)) != 0 {
		t.Fatal(clarification)
	}
}

func TestRepositoryTestModeDefaultsAndProfileHaveNoRootBridgeDependency(t *testing.T) {
	for _, ref := range []string{".github/scripts/validate_test_modes.py", ".github/scripts/resolve_test_modes.py"} {
		source := repositoryText(t, ref)
		if !strings.Contains(source, ".ptsip/profiles/main.ptsip.yaml") {
			t.Fatal("explicit self-profile default missing", ref)
		}
	}
	text := repositoryText(t, ".ptsip/profiles/main.ptsip.yaml")
	if strings.Contains(text, `"ptsip.yaml"`) || !strings.Contains(text, ".ptsip/profiles/main.ptsip.yaml") {
		t.Fatal("self-profile root bridge dependency")
	}
	repo := testrepo.Open(testrepo.Root(t))
	profile, err := repo.Read(".ptsip/profiles/main.ptsip.yaml")
	if err != nil {
		t.Fatal(err)
	}
	modes, err := repo.Read(".github/test_modes.yaml")
	if err != nil {
		t.Fatal(err)
	}
	components := map[string]testrepo.Object{}
	for _, raw := range profile["components"].([]any) {
		row := raw.(testrepo.Object)
		components[row["id"].(string)] = row
	}
	seen := map[string]bool{}
	for _, raw := range modes["modes"].([]any) {
		row := raw.(testrepo.Object)
		id := row["id"].(string)
		if seen[id] {
			t.Fatal("duplicate mode", id)
		}
		seen[id] = true
		component := components[row["component_ref"].(string)]
		if component == nil {
			t.Fatal("unregistered verification component", row)
		}
		found := false
		for _, role := range component["roles"].([]any) {
			if role == "VERIFICATION" {
				found = true
			}
		}
		if !found {
			t.Fatal("mode routes to a non-verification owner", row)
		}
	}
}

func TestLocalCatalogDeclarationsUseExplicitDeveloperProfile(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	catalog := testrepo.ReadJSON(t, repo, ".ptsip/profiles/index.json")
	if catalog["default_profile"] != "main" {
		t.Fatal(catalog)
	}
	found := false
	for _, raw := range catalog["profiles"].([]any) {
		row := raw.(testrepo.Object)
		if row["id"] != "main" {
			continue
		}
		found = true
		if row["resource"] != "main.ptsip.yaml" {
			t.Fatal(row)
		}
		profile, err := repo.Read(".ptsip/profiles/" + row["resource"].(string))
		if err != nil {
			t.Fatal(err)
		}
		architecture := false
		for _, raw := range profile["components"].([]any) {
			component := raw.(testrepo.Object)
			for _, field := range []string{"include", "analysis_inputs"} {
				if values, ok := component[field].([]any); ok {
					for _, path := range values {
						if path == "ptsip.yaml" {
							t.Fatal("local profile root bridge dependency")
						}
						if component["id"] == "repository-architecture" && path == ".ptsip/profiles/main.ptsip.yaml" {
							architecture = true
						}
					}
				}
			}
		}
		if !architecture {
			t.Fatal("self-profile is not in architecture inputs")
		}
	}
	if !found {
		t.Fatal("main profile missing")
	}
}

func TestConsumerDefaultProfilePathRemainsRootPtsipYAML(t *testing.T) {
	source := repositoryText(t, "src/ptsip/repository/profile_path.py")
	pattern := regexp.MustCompile(`(?m)^DEFAULT_PROFILE_PATH\s*=\s*["']ptsip\.yaml["']`)
	if !pattern.MatchString(source) {
		t.Fatal("consumer profile default changed")
	}
}

func TestFullCISelfManagementCommandsSelectDeveloperProfile(t *testing.T) {
	source := repositoryText(t, ".github/workflows/tooling-test.yml")
	for _, command := range []string{"ptsip validate . --profile .ptsip/profiles/main.ptsip.yaml --json", "ptsip clarify . --profile .ptsip/profiles/main.ptsip.yaml --json", "ptsip gate . --profile .ptsip/profiles/main.ptsip.yaml --coordination local --json", "ptsip conform . --profile .ptsip/profiles/main.ptsip.yaml --artifact-evidence $evidencePath --json"} {
		if !strings.Contains(source, command) {
			t.Fatal("explicit self-profile CI command missing", command)
		}
	}
}
