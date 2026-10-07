package rootfamily

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentRootValidationHasNoMigrationArchiveOrProgramDependency(t *testing.T) {
	root := copyDeveloperWithoutLegacy(t)
	for _, plane := range contract(t).Planes {
		for _, ref := range []string{"registries/root-family-migration.json", "registries/root-family-projection.module.json"} {
			if err := os.Remove(filepath.Join(root, plane.Path, ref)); err != nil {
				t.Fatal(err)
			}
		}
	}
	binary := buildAutomation(t)
	for _, plane := range contract(t).Planes {
		result, err, output := automation(t, binary, root, "root-family-entry", "validate", "--policy-class", plane.PolicyClass)
		if err != nil || result["status"] != "PASS" {
			t.Fatal(result, err, output)
		}
	}
	inspected, err, output := automation(t, binary, root, "policy-lifecycle", "inspect", "MPD-INFO-0003")
	if err != nil || inspected["policy_status"] != "APPROVED" {
		t.Fatal(inspected, err, output)
	}
	result := python(t, "-c", `import json,sys
from pathlib import Path
from ptsip.governance.authority import AuthorityCatalog
from ptsip.governance.model import GovernanceAuthorityError
c=AuthorityCatalog(Path(sys.argv[1]))
ids=c.validate_current_corpus()
try:
 c.load_current_record('SFP-0001')
 retired=False
except GovernanceAuthorityError as exc:
 retired=exc.code=='CURRENT_SUPPORT_POLICY_NOT_SELECTED'
print(json.dumps({'count':len(ids),'retired_source_rejected':retired,'current_only':all('-' in i[4:] for i in ids)}))`, root)
	if result["retired_source_rejected"] != true || result["current_only"] != true {
		t.Fatal(result)
	}
}

func TestCurrentRootRulesMayEvolveAndCurrentContractErrorsStillFailClosed(t *testing.T) {
	binary := buildAutomation(t)
	for _, defect := range []string{"valid_rule_change", "owner_missing", "foreign_class", "index_status", "required_schema_field", "kernel_mismatch"} {
		t.Run(defect, func(t *testing.T) {
			root := copyDeveloperWithoutLegacy(t)
			path := filepath.Join(root, "developer/policy/INFO/MPD-INFO-0003.yaml")
			owner := read(t, path)
			switch defect {
			case "valid_rule_change":
				mapping(t, owner["rules"])["current_information_contract"] = map[string]any{"current_revision": "independent_current_contract"}
			case "owner_missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "foreign_class":
				owner["policy_class"] = "PTSIP_SUPPORT_FEATURE"
			case "index_status":
				mapping(t, owner["policy"])["status"] = "DRAFT"
			case "required_schema_field":
				delete(owner, "authority_subject")
			case "kernel_mismatch":
				owner["exclusive_kernel"] = []any{"UNREGISTERED_KERNEL"}
			}
			if defect != "owner_missing" {
				write(t, path, owner)
			}
			result, err, output := automation(t, binary, root, "root-family-entry", "validate", "--policy-class", "PTSIP_DEVELOPER_POLICY")
			if defect == "valid_rule_change" {
				if err != nil || result["status"] != "PASS" {
					t.Fatal(result, err, output)
				}
			} else if err == nil {
				t.Fatal("invalid current contract accepted", defect, result)
			}
			if strings.Contains(output, "ROOT_MIGRATION_FROZEN_INTERFACE_CHANGED") {
				t.Fatal("current authority still depends on source reconstruction", output)
			}
		})
	}
}
