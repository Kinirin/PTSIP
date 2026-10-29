package profileverify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// The snapshot is a developer artifact-shape contract, not a replacement
// authority for consumer profile identity, lifecycle or runtime admission.
func verifyRouting(root string) error {
	raw, err := os.ReadFile(filepath.Join(root, ".ptsip/contracts/release/Platform/Common/profile/index.json"))
	if err != nil {
		return err
	}
	var index struct {
		Schema      string `json:"schema_version"`
		Lifecycle   string `json:"lifecycle"`
		Replacement bool   `json:"replacement_required"`
		Scope       string `json:"scope"`
		Authority   string `json:"policy_authority"`
		Consumer    *bool  `json:"consumer_contract_authority"`
		Publication *bool  `json:"publication_authorized"`
		Snapshot    string `json:"snapshot"`
		Schemas     string `json:"schema_storage_root"`
		Projection  string `json:"projection"`
		Legacy      *bool  `json:"legacy_fallback"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		return err
	}
	if index.Schema != "ptsip-release-profile-verification/v1" || index.Lifecycle != "TEMPORARY_REPLACEABLE" || !index.Replacement || index.Scope != "STRUCTURE_ONLY_SOURCE_SNAPSHOT" || index.Authority != "NONE" ||
		index.Consumer == nil || *index.Consumer || index.Publication == nil || *index.Publication || index.Legacy == nil || *index.Legacy ||
		index.Snapshot != profileContractPath || index.Schemas != ".ptsip/contracts/schemas/release/profile/" || index.Projection != profileProjectionPath {
		return fmt.Errorf("PROFILE_VERIFICATION_ROUTING_INVALID")
	}
	return nil
}
