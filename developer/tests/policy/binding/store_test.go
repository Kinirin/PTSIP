package binding_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const firstSection = "unit_mpd_spec_0006_d25095d9c02a"
const secondSection = "unit_mpd_spec_0006_d1ce0ab7249e"

func storeFixture(t *testing.T) (*testrepo.Repository, *binding.Store) {
	t.Helper()
	repo := testrepo.Open(t.TempDir())
	testrepo.CopyCatalogContracts(t, repo)
	testrepo.CopyFiles(t, repo,
		binding.BindingSchemaPath,
		"developer/policy/index.yaml", "developer/policy/schemas/root-family-policy.schema.json",
		"developer/policy/contracts/go-policy-resolver.v1.yaml",
		"developer/policy/policy-resolver-bindings/registry.yaml", "developer/policy/policy-resolver-bindings/bindings.jsonl",
		"developer/policy/schemas/policy-resolver-binding.schema.json",
		"developer/policy/CNTR/MPD-CNTR-0004.yaml", "developer/policy/INFO/MPD-INFO-0001.yaml")
	identity := object{}
	for _, key := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
		identity[key] = created()[key]
	}
	testrepo.Write(t, repo, created()["plan_ref"].(string), object{"plan_identity": identity})
	return repo, binding.NewStore(repo)
}

func TestRootBindingRelationsDistinguishExactSectionSets(t *testing.T) {
	_, store := storeFixture(t)
	first, second := created(), created()
	first["policy_sections"], second["policy_sections"] = []any{firstSection}, []any{secondSection}
	second["binding_id"] = "PPB-0002"
	if failures := store.ValidateBindingRegistry(registry(first, second)); len(failures) != 0 {
		t.Fatal(failures)
	}
	first["policy_sections"], second["policy_sections"] = []any{firstSection, secondSection}, []any{secondSection, firstSection}
	if failures := store.ValidateBindingRegistry(registry(first, second)); len(failures) != 1 || !strings.Contains(failures[0], "duplicate created relation") {
		t.Fatalf("equivalent section sets were not duplicates: %v", failures)
	}
	first["policy_sections"] = []any{"unregistered_section"}
	if failures := store.ValidateBindingRegistry(registry(first)); len(failures) != 1 || !strings.Contains(failures[0], "unknown policy section") {
		t.Fatalf("unknown section admitted: %v", failures)
	}
}

func TestRootBindingsMaterializeDisjointSectionsAndRejectDuplicateWithoutMutation(t *testing.T) {
	repo, store := storeFixture(t)
	rows := []object{}
	for i, sections := range [][]any{{firstSection}, {secondSection}, {firstSection}} {
		row := object{"binding_id": []string{"PPB-0001", "PPB-0002", "PPB-0003"}[i], "policy_ref": "MPD-INFO-0001", "policy_sections": sections, "planning_state": "NOT_CREATED"}
		rows = append(rows, row)
	}
	testrepo.Write(t, repo, binding.BindingRegistryPath, registry(rows...))
	for _, id := range []string{"PPB-0001", "PPB-0002"} {
		values := created()
		delete(values, "planning_state")
		values["binding_id"] = id
		result, err := store.LinkPlan(values)
		if err != nil || result["status"] != "LINKED" {
			t.Fatalf("link %s: %v, %#v", id, err, result)
		}
		result, err = store.LinkPlan(values)
		if err != nil || result["status"] != "CURRENT" || result["changed"] != false {
			t.Fatalf("idempotent link %s: %v, %#v", id, err, result)
		}
	}
	path, _ := repo.Path(binding.BindingRegistryPath)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	values := created()
	values["binding_id"] = "PPB-0003"
	if _, err := store.LinkPlan(values); err == nil || !strings.Contains(err.Error(), "DUPLICATE_CREATED_RELATION") {
		t.Fatalf("duplicate materialization admitted: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("failed link changed registry: %v", err)
	}
}

func TestBindingStorePreservesCASAndRejectsEmptyOrUnknownQueries(t *testing.T) {
	repo, store := storeFixture(t)
	testrepo.Write(t, repo, binding.BindingRegistryPath, registry())
	snapshot, err := store.LoadBindingRegistry(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReplaceBindingRegistry(registry(), "stale", store.ValidateBindingRegistry); err == nil || !strings.Contains(err.Error(), "BINDING_REGISTRY_STALE") {
		t.Fatalf("stale CAS admitted: %v", err)
	}
	current, err := store.LoadBindingRegistry(true)
	if err != nil || current.Digest != snapshot.Digest {
		t.Fatalf("stale CAS changed registry: %v", err)
	}
	for _, query := range []object{{}, {"plan_id": "WU-07"}, {"binding_id": ""}} {
		if _, err := store.ResolveBindings(query); err == nil {
			t.Fatalf("invalid query admitted: %#v", query)
		}
	}
}
