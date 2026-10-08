package rootfamily

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// The new composition authority must not resolve through a retired Support ID.
// The archival migration registry is deliberately outside this current-policy test.
func TestSourceCompositionCurrentAuthorityHasNoLegacyIdentity(t *testing.T) {
	root := repository(t)
	path := filepath.Join(root, "src/policy/INFO/SFP-INFO-0004.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	legacyIdentity := regexp.MustCompile(`\bSFP-[0-9]{4}\b|\bunit_sfp_[0-9]{4}_[0-9a-f]+\b`)
	if legacyIdentity.Match(data) {
		t.Fatal("current source-composition authority references a legacy Support identity")
	}

	policy := read(t, path)
	identity := mapping(t, policy["policy"])
	if identity["id"] != "SFP-INFO-0004" || identity["status"] != "DRAFT" ||
		policy["policy_class"] != "PTSIP_SUPPORT_FEATURE" || policy["responsibility_family"] != "INFO" {
		t.Fatal("current source-composition identity/status mismatch")
	}
	semantics := mapping(t, policy["authority_semantics"])
	independence := mapping(t, semantics["independence"])
	if independence["legacy_policy_implicit_inheritance"] != "FORBIDDEN" ||
		independence["responsibility_map_semantic_authority"] != "EXCLUDED" {
		t.Fatal("source-composition policy cannot inherit legacy or architecture authority")
	}
	decisions := mapping(t, semantics["approved_decisions"])
	if decisions["automatic_grouping_determinism"] != "DETERMINISTIC_GROUPING" {
		t.Fatal("source-composition policy lost its approved grouping determinism")
	}

	index := read(t, filepath.Join(root, "src/policy/index.yaml"))
	found := 0
	for _, raw := range sequence(t, index["policies"]) {
		entry := mapping(t, raw)
		if entry["id"] != "SFP-INFO-0004" {
			continue
		}
		found++
		if entry["path"] != "INFO/SFP-INFO-0004.yaml" ||
			entry["authority_role"] != "CANONICAL_AUTHORITY" ||
			entry["status"] != "DRAFT" {
			t.Fatal("source-composition catalog identity, role, or status mismatch", entry)
		}
	}
	if found != 1 {
		t.Fatalf("source-composition authority registration count = %d, want 1", found)
	}
}
