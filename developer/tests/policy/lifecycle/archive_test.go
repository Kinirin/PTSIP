package lifecycle_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
)

const vpmsArchive = "developer/policy/legacy/MPD-VERI-0007.yaml"
const vpmsKeepLatest = "developer/policy/approvals/MPA-20261011-VPMS-VERI-KEEP-LATEST.yaml"

func TestWithdrawnVPMSAuthorityDoesNotRequireArchivedBody(t *testing.T) {
	repo := discoveryFixture(t)
	corpus, err := lifecycle.LoadConsistentPolicyCorpus(repo)
	if err != nil {
		t.Fatal(err)
	}
	if corpus.Records["MPD-VERI-0007"] != nil || lifecycle.Contains(corpus.CanonicalIDs, "MPD-VERI-0007") || lifecycle.Contains(corpus.IDs, "MPD-VERI-0007") {
		t.Fatal("withdrawn VPMS policy remains in current corpus", corpus)
	}
	if err := lifecycle.CheckDiscovery(repo, corpus, ""); err != nil {
		t.Fatal(err)
	}
	resolver, err := binding.NewResolver(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Policy("MPD-VERI-0007"); err == nil || !strings.Contains(err.Error(), "unknown or audit-only (not executable)") {
		t.Fatal("archived ACTIVE source became current authority", err)
	}
	result, err := lifecycle.InspectPolicy(repo, "MPD-VERI-0007")
	if err != nil || result["status"] != "NOT_FOUND" {
		t.Fatal(result, err)
	}
	ids, err := lifecycle.ActiveFamilyIDs(repo, vpmsClass, "ASSURE")
	if err != nil || len(ids) != 0 {
		t.Fatal(ids, err)
	}
	path, err := repo.Path(vpmsArchive)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("fixture unexpectedly contains frozen archive body", err)
	}
	approval, err := repo.Read(vpmsKeepLatest)
	if err != nil {
		t.Fatal(err)
	}
	decision := approval["approval"].(object)
	if decision["decision_source"] != "USER_EXPLICIT" || decision["requested_policy_id"] != "MPD-VERI-0007" || decision["target_status"] != "RETIRED" || decision["implementation_authorized"] != false {
		t.Fatal("KEEP_LATEST boundary changed", decision)
	}
}
