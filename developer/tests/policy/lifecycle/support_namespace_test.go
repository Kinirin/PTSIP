package lifecycle_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestSupportNamespaceHasNoRetiredDocumentationDependency(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	policy, err := repo.Read("developer/policy/INFO/MPD-INFO-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	namespace := policy["rules"].(object)["unit_mpd_spec_0001_5630fc226d3b"].(object)["support_feature"].(object)
	if namespace["machine_policy_path"] != "src/policy/" {
		t.Fatal(namespace)
	}
	if _, exists := namespace["human_documentation_path"]; exists {
		t.Fatal("retired documentation declared as namespace")
	}
	path, err := repo.Path("AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "docs/Support_policy") {
		t.Fatal("retired documentation dependency in entry projection")
	}
	path, err = repo.Path("docs/Support_policy")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("retired support documentation tree exists", err)
	}
}

func TestCurrentProfilesExcludeRetiredSupportPolicySelectors(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, ref := range []string{".ptsip/profiles/main.ptsip.yaml", ".ptsip/profiles/main.ptsip.yaml"} {
		t.Run(ref, func(t *testing.T) {
			profile, err := repo.Read(ref)
			if err != nil {
				t.Fatal(err)
			}
			entries := append(profile["components"].([]any), profile["associated_artifacts"].([]any)...)
			for _, raw := range entries {
				row := raw.(object)
				for _, field := range []string{"include", "analysis_inputs"} {
					for _, path := range lifecycleStrings(row[field]) {
						if strings.HasPrefix(path, "docs/Support_policy/") {
							t.Fatal("retired support policy selector", path)
						}
					}
				}
			}
		})
	}
}

func lifecycleStrings(value any) []string {
	result := []string{}
	if items, ok := value.([]any); ok {
		for _, raw := range items {
			result = append(result, raw.(string))
		}
	}
	return result
}
