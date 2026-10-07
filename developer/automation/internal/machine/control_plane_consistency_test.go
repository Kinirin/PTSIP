package machine

import (
	"reflect"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/planning"
)

func TestAuthorizationRegistryArraysMustExactlyCoverSupportCatalog(t *testing.T) {
	for _, defect := range []string{"role_not_array", "role_missing", "role_duplicate", "role_unknown", "schema_missing", "schema_duplicate"} {
		t.Run(defect, func(t *testing.T) {
			r := policyTestRepo(t)
			ref, field, predicate := "src/policy/registries/ptsip-support-authority-role-registry.yaml", "policy_roles", "AUTHORITY_ROLE_REGISTRY_VALID"
			if defect == "schema_missing" || defect == "schema_duplicate" {
				ref, field, predicate = "src/policy/registries/ptsip-support-authority-schema-registry.yaml", "entries", "AUTHORITY_SCHEMA_REGISTRY_VALID"
			}
			registry, err := r.Read(ref)
			if err != nil {
				t.Fatal(err)
			}
			rows := List(registry[field])
			switch defect {
			case "role_not_array":
				registry[field] = Object{"unexpected": rows[0]}
			case "role_missing", "schema_missing":
				registry[field] = rows[:len(rows)-1]
			case "role_duplicate", "schema_duplicate":
				registry[field] = append(rows, policyClone(Map(rows[0])))
			case "role_unknown":
				Map(rows[len(rows)-1])["policy_id"] = "SFP-UNKNOWN-9999"
			}
			policyTestWrite(t, r, ref, registry)
			ready, err := r.AuthorizationReadiness()
			if err != nil {
				t.Fatal(err)
			}
			if ready[predicate] != false {
				t.Fatalf("%s accepted: %#v", defect, ready)
			}
			result, err := r.AuthorizationTransition("PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME", ready)
			if err != nil || result["state"] != "HOLD_NOT_AUTHORIZED" {
				t.Fatal(result, err)
			}
		})
	}
}

func TestLegacyDependencyCurrentSurfacePreservesFrozenBoundaryAndNativeCoverage(t *testing.T) {
	for _, test := range []struct {
		ref      string
		selected bool
	}{
		{"schemas/ptsip-profile-pp-1.01.schema.json", false},
		{"src/ptsip/specdata/ptsip-profile-pp-1.01.schema.json", false},
		{"developer/automation/release/identity.go", true},
		{"developer/automation/internal/machine/maintenance.go", true},
		{"developer/automation/internal/machine/control_plane_regression_test.go", false},
		{"developer/automation/internal/machine/testdata/control_plane_vectors.json", false},
		{"developer/policy/NORM/MPD-NORM-0001.yaml", true},
		{"schemas/ptsip-support-authority-semantics.schema.json", true},
		{"src/ptsip/governance/authority.py", true},
	} {
		if legacyDependencyCurrentSurface(test.ref) != test.selected {
			t.Fatalf("current surface changed: %#v", test)
		}
	}
	r := &Repository{Root: t.TempDir()}
	if err := r.AtomicWrite("developer/automation/release/fixture.go", []byte("package release\nconst invalid = \"decisions/ADR-9999.yaml\"\n"), nil); err != nil {
		t.Fatal(err)
	}
	if err := r.AtomicWrite("schemas/ptsip-profile-pp-1.01.schema.json", []byte("{\"description\":\"Frozen ADR-0021 provenance\"}"), nil); err != nil {
		t.Fatal(err)
	}
	hits, err := r.LegacyDependencyHits()
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("native current dependency was not detected")
	}
	for _, hit := range hits {
		if hit.Path != "developer/automation/release/fixture.go" {
			t.Fatal("frozen data classified as current dependency", hit)
		}
	}
}

func TestSparseRelationSourcesDoNotInventMigrationUnits(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	graph, err := r.Read("developer/policy/registries/root-family-migration.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(graph["sources"]) {
		source := Map(raw)
		if source["source_policy_id"] != "MPD-SPEC-0001" {
			continue
		}
		for _, raw := range List(source["units"]) {
			if Map(raw)["source_pointer"] == "/relations" {
				t.Fatal("empty optional relation became a source unit")
			}
		}
		return
	}
	t.Fatal("registered sparse source missing")
}

func TestNestedGovernanceFailurePreservesExactContainerPath(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := r.Read("developer/policy/registries/governance-source-registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"fixture.yaml: .source[0].source source uses legacy governance source and requires explicit role"}
	got := planning.GovernanceErrors(Object{"source": []any{Object{"source": "DIRECT_PROJECT_OWNER_INSTRUCTION"}}}, registry, "fixture.yaml")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostic path lost: %v", got)
	}
}
