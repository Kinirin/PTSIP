package machine

import "testing"

func TestAuditTargetsRequireExactVerifiedGoMappingsAndNeverWaiveMissingFiles(t *testing.T) {
	const source = "developer/automation/dev_setup.py"
	for _, defect := range []string{"valid", "pending", "unknown_evidence", "empty", "outside", "missing", "duplicate", "reintroduced"} {
		t.Run(defect, func(t *testing.T) {
			r := policyTestRepo(t)
			const ref = "developer/policy/registries/go-automation-migration.json"
			inventory, err := r.Read(ref)
			if err != nil {
				t.Fatal(err)
			}
			rows := List(inventory["modules"])
			var module Object
			for _, raw := range rows {
				if Map(raw)["python_path"] == source {
					module = Map(raw)
				}
			}
			if module == nil {
				t.Fatal("registered retirement binding missing")
			}
			switch defect {
			case "pending":
				module["equivalence_verification"] = "PENDING"
			case "unknown_evidence":
				module["equivalence_verification"] = "ASSUMED_READY"
			case "empty":
				module["go_paths"] = []any{}
			case "outside":
				module["go_paths"] = []any{"developer/automation/../policy/metadata.go"}
			case "missing":
				module["go_paths"] = []any{"developer/automation/missing.go"}
			case "duplicate":
				inventory["modules"] = append(rows, policyClone(module))
			case "reintroduced":
				if err := r.AtomicWrite(source, []byte("raise RuntimeError('must not execute')\n"), nil); err != nil {
					t.Fatal(err)
				}
			}
			policyTestWrite(t, r, ref, inventory)
			err = r.VerifyAuditImplementationTarget(source)
			if (err == nil) != (defect == "valid") {
				t.Fatal("audit target mapping admission changed", defect, err)
			}
		})
	}
}
