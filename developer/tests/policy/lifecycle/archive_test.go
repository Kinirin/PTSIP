package lifecycle_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	domainpolicy "github.com/Kinirin/PTSIP/developer/automation/policy"
	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
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
	if _, err := resolver.Policy("MPD-VERI-0007"); err == nil || !strings.Contains(err.Error(), "unknown developer policy identity") {
		t.Fatal("archived ACTIVE source became current authority", err)
	}
	result, err := lifecycle.InspectPolicy(repo, "MPD-VERI-0007")
	if err != nil || result["status"] != "NOT_FOUND" {
		t.Fatal(result, err)
	}
	ids, err := lifecycle.ActiveFamilyIDs(repo, vpmsClass, "VERI")
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

func TestWithdrawnVPMSArchivePreservesAdmittedSourceDigest(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	registry, err := repo.Read("developer/policy/registries/vpms-api-implementation.json")
	if err != nil {
		t.Fatal(err)
	}
	const admittedDigest = "e5557fda8564db439e81c6295f3452354f5cc2ae4e632ab8dcb8d061b27914f4"
	found := false
	for _, raw := range registry["preserved_files"].([]any) {
		entry := raw.(object)
		if entry["path"] == "developer/policy/VERI/MPD-VERI-0007.yaml" {
			found = true
			if entry["lf_sha256"] != admittedDigest {
				t.Fatal("historical trust digest changed", entry)
			}
		}
	}
	if !found {
		t.Fatal("historical VPMS source evidence missing")
	}
	path, err := repo.Path(vpmsArchive)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if domainpolicy.SHA256(bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))) != admittedDigest {
		t.Fatal("frozen VPMS source bytes changed")
	}
}
