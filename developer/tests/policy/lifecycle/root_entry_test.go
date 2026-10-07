package lifecycle_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestRootFamilyVocabularyHasExactlyFourteenRegisteredValues(t *testing.T) {
	want := []string{"NORM", "GOV", "INTENT", "ARCH", "INFO", "CNTR", "RISK", "SUPPLY", "REAL", "ASSURE", "CTRL", "CHANGE", "OPS", "RECORD"}
	if !reflect.DeepEqual(lifecycle.RootFamilies, want) {
		t.Fatal(lifecycle.RootFamilies)
	}
}

func TestSameFamilyTokenResolvesDistinctClassScopedAuthority(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, test := range []struct{ class, prefix, plane, schema string }{{lifecycle.DeveloperClass, "MPD", "developer/policy", "developer/policy/schemas/root-family-policy.schema.json"}, {"PTSIP_SUPPORT_FEATURE", "SFP", "src/policy", "src/policy/schemas/ptsip-support-root-family-policy.schema.json"}} {
		t.Run(test.class, func(t *testing.T) {
			entry, err := lifecycle.FamilyEntry(repo, test.class, "ARCH")
			if err != nil {
				t.Fatal(err)
			}
			want := object{"policy_class": test.class, "responsibility_family": "ARCH"}
			id, ok := entry["allocated_policy_id"].(string)
			if !ok || !strings.HasPrefix(id, test.prefix+"-ARCH-") || entry["canonical_path"] != test.plane+"/ARCH/"+id+".yaml" || entry["schema_ref"] != test.schema || entry["semantic_inheritance"] != "FORBIDDEN" || !reflect.DeepEqual(entry["authority_identity"], want) {
				t.Fatal(entry)
			}
		})
	}
}

func TestLegacyDeveloperFamilyCannotResolveAsNewRootEntry(t *testing.T) {
	_, err := lifecycle.FamilyEntry(testrepo.Open(testrepo.Root(t)), lifecycle.DeveloperClass, "SPEC")
	if err == nil || !strings.Contains(err.Error(), "unregistered Root Family") {
		t.Fatal(err)
	}
}

func TestRootFamilyIDInspectionIsExactAndClassScoped(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, test := range []struct{ id, class, path string }{{"MPD-ARCH-0001", lifecycle.DeveloperClass, "developer/policy/ARCH/MPD-ARCH-0001.yaml"}, {"SFP-ARCH-0001", "PTSIP_SUPPORT_FEATURE", "src/policy/ARCH/SFP-ARCH-0001.yaml"}} {
		t.Run(test.id, func(t *testing.T) {
			result, err := lifecycle.InspectFamilyID(repo, test.id)
			if err != nil || result["policy_class"] != test.class || result["responsibility_family"] != "ARCH" || result["canonical_path"] != test.path {
				t.Fatalf("%#v %v", result, err)
			}
		})
	}
}

func TestUnregisteredFamilyDoesNotFallBack(t *testing.T) {
	_, err := lifecycle.FamilyEntry(testrepo.Open(testrepo.Root(t)), lifecycle.DeveloperClass, "UNKNOWN")
	if err == nil || !strings.Contains(err.Error(), "unregistered Root Family") {
		t.Fatal(err)
	}
}

func TestSupportIndexAcceptsExactFamilyPathsAndRejectsMisrouting(t *testing.T) {
	const schema = "src/policy/schemas/ptsip-support-feature-policy-index.schema.json"
	for _, family := range lifecycle.RootFamilies {
		t.Run(family, func(t *testing.T) {
			repo := testrepo.Open(testrepo.Root(t))
			index, err := repo.Read("src/policy/index.yaml")
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.Validate(schema, index); err != nil {
				t.Fatal(err)
			}
			entry, err := lifecycle.FamilyEntry(repo, "PTSIP_SUPPORT_FEATURE", family)
			if err != nil {
				t.Fatal(err)
			}
			path := strings.TrimPrefix(entry["canonical_path"].(string), "src/policy/")
			route := object{"id": entry["allocated_policy_id"], "path": path, "status": "DRAFT"}
			index["policies"] = append(index["policies"].([]any), route)
			if err := repo.Validate(schema, index); err != nil {
				t.Fatal(err)
			}
			other := "NORM"
			if family == other {
				other = "GOV"
			}
			for _, invalid := range []string{strings.TrimSuffix(path, ".yaml") + "Xyaml", "../" + path, other + "/" + strings.SplitN(path, "/", 2)[1]} {
				route["path"] = invalid
				if err := repo.Validate(schema, index); err == nil {
					t.Fatal("misrouted path admitted", invalid)
				}
			}
		})
	}
}
