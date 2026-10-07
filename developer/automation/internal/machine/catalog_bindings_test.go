package machine

import (
	"strings"
	"testing"
)

func TestNativeCatalogBindingsFailClosedWithoutFallingBackToPythonSources(t *testing.T) {
	for _, defect := range []string{"missing_function", "missing_source_selector", "unknown_source", "duplicate_source", "empty_targets"} {
		t.Run(defect, func(t *testing.T) {
			r := policyTestRepo(t)
			record, err := r.Read(policyCatalogContracts)
			if err != nil {
				t.Fatal(err)
			}
			rows := List(record["go_implementation_bindings"])
			binding := Map(rows[1])
			if binding["source_path"] != "developer/automation/policy_validator.py" {
				t.Fatal("fixture lost exact validator binding")
			}
			switch defect {
			case "missing_function":
				Map(List(binding["targets"])[0])["go_functions"] = []any{"UnregisteredCompiler"}
			case "missing_source_selector":
				binding["source_functions"] = List(binding["source_functions"])[1:]
			case "unknown_source":
				binding["source_path"] = "developer/automation/unknown.py"
			case "duplicate_source":
				record["go_implementation_bindings"] = append(rows, policyClone(binding))
			case "empty_targets":
				binding["targets"] = []any{}
			}
			policyTestWrite(t, r, policyCatalogContracts, record)
			failures := r.ValidateNeutralCatalogContractRegistration()
			if len(failures) == 0 {
				t.Fatal("invalid native binding accepted", defect)
			}
			if defect == "missing_function" && !strings.Contains(strings.Join(failures, "\n"), "Go selector missing: UnregisteredCompiler") {
				t.Fatal("valid Python source hid a missing native implementation", failures)
			}
		})
	}
}
