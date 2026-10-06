package machine

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func policySeedFixture(t *testing.T) *Repository {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname='ptsip'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	write := func(path string, value Object) { t.Helper(); policyTestWrite(t, r, path, value) }
	write("registry/project-profile-contracts.yaml", Object{"current": "pp.1.01", "contracts": []any{Object{"version": "pp.1.01", "schema": "schemas/ptsip-profile-pp-1.01.schema.json"}}})
	write("profiles/index.yaml", Object{"profiles": []any{Object{"id": "example", "resource": "example.ptsip.yaml", "contract": "pp.1.01"}, Object{"id": "hybrid-python-package", "resource": "hybrid.ptsip.yaml", "contract": "pp.1.01"}}})
	header := Object{"ptsip": Object{"version": "pp.1.01", "specification": Object{"source": "Kinirin/PTSIP", "revision": strings.Repeat("a", 40), "family": "0.3.7-draft"}}, "components": []any{Object{"id": "product-runtime", "include": []any{"product/app/**"}}, Object{"id": "development-toolkit", "include": []any{"developer/**"}, "analysis_inputs": []any{"product/**"}}, Object{"id": "repository-architecture", "include": []any{"README.md"}}}, "associated_artifacts": []any{Object{"id": "development-toolkit-docs", "include": []any{"developer/docs/**"}}}}
	write("profiles/example.ptsip.yaml", policyClone(header))
	hybrid := policyClone(header)
	hybrid["responsibility_map"] = Object{"overrides": Object{"components": []any{Object{"id": "package", "include": []any{"pkg/**"}}, Object{"id": "package-tests", "include": []any{"tests/**"}}}}}
	write("profiles/hybrid.ptsip.yaml", hybrid)
	write("ptsip.yaml", policyClone(header))
	write("developer/profiles/ptsip-repository.yaml", policyClone(header))
	write("schemas/ptsip-profile-pp-1.01.schema.json", Object{"properties": Object{"ptsip": Object{"required": []any{"version", "specification"}, "properties": Object{"version": Object{"const": "pp.1.01"}, "specification": Object{"required": []any{"source", "revision", "family"}, "properties": Object{"source": Object{"type": "string"}, "revision": Object{"type": "string"}, "family": Object{"type": "string"}}}}}}})
	docs := map[string][]string{}
	for _, marker := range policySeedMarkers {
		docs[marker[0]] = append(docs[marker[0]], marker[1])
	}
	for path, text := range docs {
		if err := r.AtomicWrite(path, []byte(strings.Join(text, "\n")), nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "Native Go Fixture"}, {"config", "user.email", "fixture@example.invalid"}, {"add", "."}, {"-c", "core.hooksPath=NUL", "commit", "-qm", "pp.1.01 source fixture"}} {
		if _, err := r.GitOutput(args...); err != nil {
			t.Fatal(err)
		}
	}
	return r
}
func policySeedCommit(t *testing.T, r *Repository) {
	t.Helper()
	if _, err := r.GitOutput("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GitOutput("-c", "core.hooksPath=NUL", "commit", "-qm", "fixture update"); err != nil {
		t.Fatal(err)
	}
}

func TestPolicySeedPP102NativeSourceBoundary(t *testing.T) {
	r := policySeedFixture(t)
	originalRoot, err := r.Read("ptsip.yaml")
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.SeedPP102Transition(true)
	if err != nil || result["status"] != "PASS" {
		t.Fatalf("result=%v err=%v", result, err)
	}
	profile, err := r.Read(".ptsip/profiles/main.ptsip.yaml")
	if err != nil {
		t.Fatal(err)
	}
	header := Map(profile["ptsip"])
	spec := Map(header["specification"])
	if header["version"] != "pp.1.02" || header["revision"] != "Rev.0001" || header["profile_role"] != "PROJECT" || spec["revision"] != strings.Repeat("a", 40) || spec["source"] != "Kinirin/PTSIP" || spec["family"] != nil {
		t.Fatalf("project header boundary changed: %v", header)
	}
	rootAfter, err := r.Read("ptsip.yaml")
	if err != nil || !reflect.DeepEqual(rootAfter, originalRoot) {
		t.Fatal("compatibility input root profile was overwritten")
	}
	example, err := r.Read("profiles/example.ptsip.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if Map(example["ptsip"])["profile_role"] != "DISTRIBUTED_EXAMPLE" || Map(example["ptsip"])["version"] != "pp.1.01" {
		t.Fatal("distributed source claimed PROJECT or skipped automatic transition")
	}
	if !reflect.DeepEqual(Map(List(example["components"])[0])["include"], []any{"${PRODUCT_RUNTIME_ROOT}/**"}) {
		t.Fatal("public path was not unresolved project-owned placeholder")
	}
	hybrid, err := r.Read("profiles/hybrid.ptsip.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(Map(List(Map(Map(hybrid["responsibility_map"])["overrides"])["components"])[0])["include"], []any{"${PACKAGE_ROOT}/**"}) {
		t.Fatal("hybrid override path leaked as authority")
	}
	registry, err := r.Read("registry/project-profile-contracts.yaml")
	if err != nil || registry["current"] != "pp.1.01" {
		t.Fatal("seeder manually created automatic PP transition output")
	}
	for _, path := range Strings(result["changed_paths"]) {
		if strings.Contains(path, ".py") || strings.HasPrefix(path, "profiles/history/") || path == "schemas/ptsip-profile-pp-1.02.schema.json" || path == "registry/project-profile-contracts.yaml" {
			t.Fatalf("unregistered/automatic output seeded: %s", path)
		}
	}
	generated, err := r.Path(policySeedAcceptancePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), generated, nil, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
	schema, err := r.Read("schemas/ptsip-profile-pp-1.01.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	definition := Map(Map(schema["properties"])["ptsip"])
	if !reflect.DeepEqual(definition["required"], []any{"version", "revision", "profile_role", "specification"}) {
		t.Fatal("source schema did not add explicit revision/profile role contract")
	}
	if _, err := r.SeedPP102Transition(true); err == nil {
		t.Fatal("dirty seeded worktree replay allowed")
	}
}
func TestPolicySeedGuardsPreserveSource(t *testing.T) {
	for _, kind := range []string{"apply", "dirty", "replay", "marker", "escape"} {
		t.Run(kind, func(t *testing.T) {
			r := policySeedFixture(t)
			apply := true
			switch kind {
			case "apply":
				apply = false
			case "dirty":
				if err := r.AtomicWrite("dirty.txt", []byte("uncommitted"), nil); err != nil {
					t.Fatal(err)
				}
			case "replay":
				registry, _ := r.Read("registry/project-profile-contracts.yaml")
				registry["current"] = "pp.1.02"
				policyTestWrite(t, r, "registry/project-profile-contracts.yaml", registry)
				policySeedCommit(t, r)
			case "marker":
				if err := r.AtomicWrite("STATUS.md", []byte("missing exact marker"), nil); err != nil {
					t.Fatal(err)
				}
				policySeedCommit(t, r)
			case "escape":
				catalog, _ := r.Read("profiles/index.yaml")
				Map(List(catalog["profiles"])[0])["resource"] = "../ptsip.yaml"
				policyTestWrite(t, r, "profiles/index.yaml", catalog)
				policySeedCommit(t, r)
			}
			before, err := r.GitOutput("status", "--porcelain")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.SeedPP102Transition(apply); err == nil {
				t.Fatal("invalid seeder input admitted")
			}
			after, err := r.GitOutput("status", "--porcelain")
			if err != nil || before != after {
				t.Fatal("failed seed changed source state")
			}
			if _, err := os.Stat(filepath.Join(r.Root, ".ptsip/profiles/main.ptsip.yaml")); !os.IsNotExist(err) {
				t.Fatal("failed seed wrote a partial project profile")
			}
		})
	}
}
