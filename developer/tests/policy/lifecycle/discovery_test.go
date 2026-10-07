package lifecycle_test

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func discoveryFixture(t *testing.T) *testrepo.Repository {
	t.Helper()
	repo := testrepo.Open(t.TempDir())
	// Frozen bodies are deliberately absent: migration metadata is audit
	// evidence, and Root records remain the only developer authority input.
	testrepo.CopyTree(t, repo, "developer/policy")
	return repo
}

func TestDiscoveryPreservesReleaseSourceAsAuditOnlyWithoutArchivedBodies(t *testing.T) {
	repo := discoveryFixture(t)
	path, err := repo.Path("developer/policy/legacy/MPD-RELS-0002.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("fixture must not contain archived source bodies", err)
	}
	corpus, err := lifecycle.LoadConsistentPolicyCorpus(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.CheckDiscovery(repo, corpus, ""); err != nil {
		t.Fatal(err)
	}
	if corpus.Records["MPD-RELS-0002"] != nil {
		t.Fatal("migration source became a canonical policy record")
	}
	resolver, err := binding.NewResolver(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Policy("MPD-RELS-0002"); err == nil || !strings.Contains(err.Error(), "audit-only") {
		t.Fatal("ACTIVE source status must not grant execution authority", err)
	}
	owner, err := resolver.Policy("MPD-CTRL-0001")
	if err != nil {
		t.Fatal(err)
	}
	if owner["rules"].(object)["unit_mpd_rels_0002_e782809fbb27"] == nil {
		t.Fatal("canonical release publication gate is missing")
	}
}

func TestDiscoveryRejectsMigrationSourceMembershipAndMetadataDrift(t *testing.T) {
	for _, defect := range []string{"missing_source", "changed_status", "changed_path", "unknown_source", "missing_registry_ref", "unregistered_source", "inconsistent_header"} {
		t.Run(defect, func(t *testing.T) {
			repo := discoveryFixture(t)
			index, err := repo.Read(lifecycle.PolicyIndex)
			if err != nil {
				t.Fatal(err)
			}
			rows := index["policies"].([]any)
			var source object
			for i, raw := range rows {
				entry := raw.(object)
				if entry["id"] != "MPD-RELS-0002" {
					continue
				}
				source = entry
				switch defect {
				case "missing_source":
					index["policies"] = append(rows[:i], rows[i+1:]...)
				case "changed_status":
					entry["status"] = "DRAFT"
				case "changed_path":
					entry["path"] = "developer/policy/legacy/MPD-RELS-0001.yaml"
				}
				break
			}
			if source == nil {
				t.Fatal("release migration source not registered")
			}
			switch defect {
			case "unknown_source":
				rows = append(rows, object{"id": "MPD-RELS-9999", "path": "developer/policy/legacy/MPD-RELS-9999.yaml", "status": "ACTIVE", "policy_class": lifecycle.DeveloperClass, "authority_role": "MIGRATION_SOURCE"})
				sort.Slice(rows, func(i, j int) bool { return rows[i].(object)["id"].(string) < rows[j].(object)["id"].(string) })
				index["policies"] = rows
			case "missing_registry_ref":
				delete(index, "migration_registry_ref")
			case "unregistered_source", "inconsistent_header":
				const ref = "developer/policy/registries/root-family-migration.json"
				graph, err := repo.Read(ref)
				if err != nil {
					t.Fatal(err)
				}
				sources := graph["sources"].([]any)
				for i, raw := range sources {
					source := raw.(object)
					if source["source_policy_id"] != "MPD-RELS-0002" {
						continue
					}
					if defect == "unregistered_source" {
						graph["sources"] = append(sources[:i], sources[i+1:]...)
					} else {
						source["header"].(object)["policy"].(object)["status"] = "DRAFT"
					}
					break
				}
				raw, err := json.Marshal(graph)
				if err != nil {
					t.Fatal(err)
				}
				if err := repo.AtomicWrite(ref, raw, nil); err != nil {
					t.Fatal(err)
				}
			}
			testrepo.Write(t, repo, lifecycle.PolicyIndex, index)
			_, err = lifecycle.LoadConsistentPolicyCorpus(repo)
			lifecycleCode(t, err, "MIGRATION_SOURCE_CATALOG_MISMATCH")
		})
	}
}
