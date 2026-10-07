package machine

import (
	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"strings"
	"testing"
)

func TestPolicyDiscoveryRejectsCatalogAndSubjectOmissions(t *testing.T) {
	for _, id := range []string{"MPD-RISK-0001", "MPD-RELS-0002"} {
		t.Run(id, func(t *testing.T) {
			r := policyTestRepo(t)
			index, err := r.Read(policyIndex)
			if err != nil {
				t.Fatal(err)
			}
			rows := []any{}
			found := false
			for _, raw := range List(index["policies"]) {
				if Map(raw)["id"] == id {
					found = true
					continue
				}
				rows = append(rows, raw)
			}
			if !found {
				t.Fatal("regression identity is not registered", id)
			}
			index["policies"] = rows
			policyTestWrite(t, r, policyIndex, index)
			subject, err := r.Read(policySubjectRegistry)
			if err != nil {
				t.Fatal(err)
			}
			values := []any{}
			for _, raw := range List(Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"]) {
				if raw != id {
					values = append(values, raw)
				}
			}
			Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"] = values
			policyTestWrite(t, r, policySubjectRegistry, subject)

			// Current authority omission fails closed. Historical omission is
			// reported only by the explicit migration evidence audit.
			errors := r.ValidateDeveloperPolicy()
			if id == "MPD-RELS-0002" {
				if len(errors) != 0 {
					t.Fatal("history constrained current authority", errors)
				}
				graph, err := r.Read("developer/policy/registries/root-family-migration.json")
				if err != nil {
					t.Fatal(err)
				}
				err = lifecycle.CheckMigrationSourceCatalog(index, graph)
				if err == nil || !strings.Contains(err.Error(), id) {
					t.Fatal("historical omission was not detected", err)
				}
				return
			}
			if len(errors) == 0 {
				t.Fatal("policy-validator accepted omitted catalog membership", id)
			}
			if id == "MPD-RELS-0002" && !strings.Contains(strings.Join(errors, "\n"), id) {
				t.Fatal("diagnostic lost the missing source identity", errors)
			}
			corpus, err := r.LoadConsistentPolicyCorpus()
			if err == nil {
				err = r.policyCheckDiscovery(corpus, "")
			}
			if err == nil {
				t.Fatal("policy lifecycle accepted omitted catalog membership", id)
			}
		})
	}
}
