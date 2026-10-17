package lifecycle_test

import (
    "testing"

    "github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

// Only currently approved independent version semantics are tested.
// Historical GOV version/status coupling and automatic increments are retired.
func TestIndependentVersionContractRegistry(t *testing.T) {
    repo := testrepo.Open(testrepo.Root(t))
    authority, err := repo.Read("developer/policy/registries/lifecycle-status-contract-authority.json")
    if err != nil { t.Fatal(err) }
    details := authority["version"].(object)
    ref := "developer/contracts/lifecycle/Policy_version/Policy_version.schema.json"
    if details["contract"] != ref || details["version_field_required"] != true {
        t.Fatalf("current contract authority mismatch: %v", details)
    }
    catalog, err := repo.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
    if err != nil { t.Fatal(err) }
    resources := catalog["lifecycle_schema_resources"].(object)
    if resources["version"] != ref {
        t.Fatalf("schema registry points to retired contract: %v", resources["version"])
    }
    for _, tc := range []struct { name string; value object; valid bool }{
        {"simple", object{"version": "1.0"}, true},
        {"unconstrained-format", object{"version": "v1.0-preview"}, true},
        {"missing", object{}, false},
        {"number", object{"version": 1}, false},
    } {
        t.Run(tc.name, func(t *testing.T) {
            err := repo.Validate(ref, tc.value)
            if (err == nil) != tc.valid { t.Fatalf("valid=%v error=%v", tc.valid, err) }
        })
    }
}
