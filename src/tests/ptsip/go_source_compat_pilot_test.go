package contracts_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
	"github.com/Kinirin/PTSIP/src/ptsip/inspection"
	"github.com/Kinirin/PTSIP/src/ptsip/pilot"
	sourcecompat "github.com/Kinirin/PTSIP/src/ptsip/source_compat"
)

func frozenProfile034() string {
	return fmt.Sprintf(`ptsip:
  version: 0.3.4-draft
  specification:
    source: %s
    revision: %s
boundaries:
  product:
    roots: [src]
  toolchain:
    roots: [tools]
policies:
  product_to_toolchain_runtime_dependency: deny
  toolchain_in_product_package: deny
  independent_build_resolution: required
`, sourcecompat.SpecificationSource, sourcecompat.V034Revision)
}

func frozenProfile036() string {
	return fmt.Sprintf(`ptsip:
  version: 0.3.6-draft
  specification:
    source: %s
    revision: %s
responsibility_map:
  mode: explicit
components:
  - id: app
    classification: PRODUCT
    include: [src/**]
    purpose: product runtime
    roles: [IMPLEMENTATION]
policies:
  product_to_nonproduct_runtime_dependency: deny
  nonproduct_in_product_package: deny
  independent_build_resolution: required
`, sourcecompat.SpecificationSource, sourcecompat.V036Revision)
}

func provideFrozenProfileAssets(t *testing.T, packageRoot string) {
	t.Helper()
	for _, name := range []string{"ptsip-source-profile-0.3.4.schema.json", "ptsip-source-profile-0.3.6.schema.json"} {
		data, err := os.ReadFile(filepath.Join(productRoot(t), "specdata", name))
		if err != nil {
			t.Fatal(err)
		}
		writeRepositoryFixture(t, packageRoot, "specdata/"+name, string(data))
	}
}

func discoverHistoricalSource(t *testing.T, root string) sourcecompat.ObservedBinding {
	t.Helper()
	observed := repositoryOperation(t, "discovery", "discovery.profile.candidates", object{"root": root, "candidate_paths": array{}})
	data, err := json.Marshal(observed["source"])
	if err != nil {
		t.Fatal(err)
	}
	var generation sourcecompat.ObservedBinding
	if err := json.Unmarshal(data, &generation); err != nil || generation.Path == "" {
		t.Fatalf("missing source observation: %v %v", observed, err)
	}
	return generation
}

func TestGoFrozenSourceCompatibilityAndProjection(t *testing.T) {
	ctx := context.Background()
	install := installedProfileRuntime(t, "validation")
	provideFrozenProfileAssets(t, install.packageRoot)
	reader := sourcecompat.OpenInstalled(install.cli, install.packageRoot, install.scripts)
	for _, test := range []struct {
		name   string
		body   string
		family sourcecompat.Family
	}{
		{"boundaries", frozenProfile034(), sourcecompat.Tool035Profile},
		{"explicit components", frozenProfile036(), sourcecompat.Tool036Profile},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := repositoryFixture(t)
			writeRepositoryFixture(t, root, "ptsip.yaml", test.body)
			binding := discoverHistoricalSource(t, root)
			result := reader.ReadSourceProfile(ctx, root, binding)
			if !result.Complete || result.Profile.Authority != "SOURCE_DECLARATION_ONLY" || result.Profile.Family != test.family || len(result.Issues) != 0 {
				t.Fatalf("source declared fields not read: %+v", result)
			}
			source := repositoryOperation(t, "projection", "projection.migration.source", result.Profile.ProjectionInput())
			if len(source["coverages"].(array)) == 0 || len(source["issues"].(array)) != 0 {
				t.Fatalf("typed historical source handoff failed: %v", source)
			}
			if issues := reader.ValidateSourceReadBinding(ctx, root, result); len(issues) != 0 {
				t.Fatalf("stable source binding rejected: %+v", issues)
			}
			writeRepositoryFixture(t, root, "ptsip.yaml", test.body+"# changed after the observation\n")
			if issues := reader.ValidateSourceReadBinding(ctx, root, result); len(issues) != 1 || issues[0].Code != "SOURCE_CONTENT_STALE" {
				t.Fatalf("stale source binding accepted: %+v", issues)
			}
		})
	}
	root := repositoryFixture(t)
	writeRepositoryFixture(t, root, "ptsip.yaml", strings.Replace(frozenProfile034(), "roots: [tools]", "roots: [src/build]", 1))
	result := reader.ReadSourceProfile(ctx, root, discoverHistoricalSource(t, root))
	if result.Complete || len(result.Issues) != 1 || result.Issues[0].Code != "SOURCE_BOUNDARY_OVERLAP" {
		t.Fatalf("overlapping historical directories were accepted: %+v", result)
	}
	writeRepositoryFixture(t, root, "ptsip.yaml", strings.Replace(frozenProfile036(), "classification: PRODUCT", "classification: UNKNOWN", 1))
	result = reader.ReadSourceProfile(ctx, root, discoverHistoricalSource(t, root))
	if result.Complete || len(result.Issues) != 1 || result.Issues[0].Code != "SOURCE_SCHEMA_ERROR" || result.Issues[0].Pointer == nil || *result.Issues[0].Pointer != "/components/0/classification" {
		t.Fatalf("invalid frozen source was accepted: %+v", result)
	}
}

func TestGoPilotReportsObservedStateBeforeAnExplicitWrite(t *testing.T) {
	packageRoot := installedProfileRuntime(t, "discovery").packageRoot
	provideFrozenProfileAssets(t, packageRoot)
	scripts := repositoryFixture(t)
	var cli string
	for _, kind := range []string{"context", "discovery", "projection", "validation"} {
		runtime := installedProfileRuntime(t, kind)
		cli = filepath.Join(scripts, filepath.Base(runtime.cli))
		entries, err := os.ReadDir(runtime.scripts)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), "ptsip-engine-") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(runtime.scripts, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(scripts, entry.Name()), data, 0755); err != nil {
				t.Fatal(err)
			}
		}
	}
	versions, err := versionsource.Read(filepath.Join(productRoot(t), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	buildCommand(t, "go", "-C", productRoot(t), "build", "-ldflags", "-X main.toolVersion="+versions.ToolVersion, "-o", cli, ".")
	root := repositoryFixture(t)
	state := repositoryFixture(t)
	t.Setenv("PTSIP_HOME", state)
	writeRepositoryFixture(t, root, "ptsip.yaml", frozenProfile034())
	writeRepositoryFixture(t, root, "src/go.mod", "module example/project\n")
	writeRepositoryFixture(t, root, "src/main.go", "package main\nimport \"fmt\"\n")
	writeRepositoryFixture(t, root, "src/script.py", "match value:\n    case None:\n        import sys\n")
	writeRepositoryFixture(t, root, "tools/test.txt", "fixture")
	result, err := pilot.OpenInstalled(cli, packageRoot, scripts).Run(context.Background(), root)
	if err != nil || result.Format != "ptsip-pilot-report/v2" || result.Tool["version"] != versions.ToolVersion || !result.Snapshot.Comparison.Stable || result.NonIntrusion.Status != "VERIFIED_NO_OBSERVED_CHANGE" {
		t.Fatalf("read-only Pilot failed: %+v, %v", result, err)
	}
	if result.Profile == nil || !result.Profile.Valid || len(result.Components.DeclaredPartition) == 0 || len(result.Components.Candidates) != 2 ||
		result.Components.Candidates[0].DecisionStatus != inspection.Unknown || result.Components.Candidates[0].Classification != nil {
		t.Fatalf("historical data was not separated from evidence candidates: %+v", result)
	}
	boundary := result.Evaluation["declared_dependency_boundaries"]
	if !result.Dependencies.CoverageComplete || len(result.Dependencies.Edges) != 2 ||
		result.Dependencies.Edges[0].Target != "fmt" || result.Dependencies.Edges[1].Target != "sys" ||
		boundary.Reason != "DECLARED_BOUNDARY_EVALUATOR_UNAVAILABLE" ||
		boundary.Status != "BLOCKED" || result.Conformance.Status != "NOT_EVALUATED" {
		t.Fatalf("Pilot must report dependency observations without asserting a boundary verdict: %+v", result)
	}
	if entries, err := os.ReadDir(state); err != nil || len(entries) != 0 {
		t.Fatalf("Run created report state before an explicit write: %v %v", entries, err)
	}
	path, err := pilot.WriteReport(&result, "")
	if err != nil || filepath.Dir(filepath.Dir(path)) != filepath.Join(state, "pilots") {
		t.Fatalf("Pilot report did not use the retained state location: %q %v", path, err)
	}
	inside := filepath.Join(root, "reports", "pilot.json")
	if _, err := pilot.WriteReport(&result, inside); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(inside)
	if err != nil || !result.NonIntrusion.OutputInsideRepository || !strings.Contains(string(data), `"output_inside_repository": true`) || strings.Contains(string(data), `"USER_AUTHORIZED_OUTPUT_WRITE"`) {
		t.Fatalf("explicit output did not record its actual boundary: %s %v", data, err)
	}
	current, err := json.Marshal(officialPP300Instance())
	if err != nil {
		t.Fatal(err)
	}
	writeRepositoryFixture(t, root, "ptsip.yaml", string(current))
	destination := filepath.Join(state, "cli-report.json")
	command := exec.Command(cli, "pilot", root, "--ptsip-root", packageRoot, "--scripts-dir", scripts, "--report", destination, "--json")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("installed Pilot CLI failed: %v %s", err, output)
	}
	var response struct {
		Format       string                    `json:"format"`
		Path         string                    `json:"report_path"`
		Conformance  struct{ Status string }   `json:"conformance"`
		Profile      *pilot.ProfileEvidence    `json:"profile"`
		Dependencies inspection.DependencyScan `json:"dependencies"`
	}
	if err := json.Unmarshal(output, &response); err != nil || response.Format != result.Format || response.Path != destination || response.Conformance.Status != "NOT_EVALUATED" {
		t.Fatalf("Pilot CLI did not expose the written evidence report: %s %v", output, err)
	}
	if response.Profile == nil || !response.Profile.Valid || response.Profile.Version != "pp.3.00" {
		t.Fatalf("Pilot CLI did not use the registered current Profile contract: %s", output)
	}
	if len(response.Dependencies.Edges) != 2 || response.Dependencies.Edges[0].Target != "fmt" || response.Dependencies.Edges[1].Target != "sys" {
		t.Fatalf("Pilot CLI did not report both dependency observations: %s", output)
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatal(err)
	}
}
