package releasetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
	"github.com/pelletier/go-toml/v2"
)

func TestToolIdentityAndCompletePackageHaveIndependentMetadata(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := release.ParseComposition(b)
	if err != nil || c.Tool != "T.0.3.8.b3" || c.PackagePyPI != "0.3.8b3" || c.PackageDisplay != "0.3.8.b3" {
		t.Fatalf("beta2 identity: %+v %v", c, err)
	}
	if v, err := release.ToolDistributionVersion(c.Tool); err != nil || v != "0.3.8b3" {
		t.Fatalf("Tool projection: %s %v", v, err)
	}
	if _, err := release.ParseComposition([]byte(strings.Replace(string(b), `version = "0.3.8b3"`, `version = "T.0.3.8.b3"`, 1))); err == nil {
		t.Fatal("T prefix admitted as PyPI version")
	}
	fixture := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixture, "pyproject.toml"), b, 0644); err != nil {
		t.Fatal(err)
	}
	if err := release.MaterializeToolProject(fixture); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(fixture, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Project struct {
			Name, Version string
			Dependencies  []string
		}
	}
	if err := toml.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if d.Project.Name != "ptsip-tool" || d.Project.Version != "0.3.8b3" || len(d.Project.Dependencies) != 2 {
		t.Fatalf("standalone Tool metadata: %+v", d)
	}
}

func TestBeta2AdmissionRejectsPartialPackageAndPolicyPass(t *testing.T) {
	root := repoRoot(t)
	s, err := release.Select(root, "package", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := release.AdmitBeta2(root, s); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"tool", "context", "context,projection"} {
		partial, err := release.Select(root, target, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := release.AdmitBeta2(root, partial); err == nil {
			t.Fatalf("partial %s admitted as beta2", target)
		}
	}
	b, err := os.ReadFile(filepath.Join(root, release.Beta2AuthorizationPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{`"merge_to_main": true`, `"automated_policy_validation_available": true`} {
		field := strings.Split(change, ":")[0]
		mutated := strings.Replace(string(b), field+": false", change, 1)
		fixture := t.TempDir()
		p := filepath.Join(fixture, release.Beta2AuthorizationPath)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(mutated), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := release.Beta2Branch(fixture); err == nil {
			t.Fatal("main write or policy PASS admitted")
		}
	}
}
