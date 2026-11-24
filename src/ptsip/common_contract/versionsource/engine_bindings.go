package versionsource

import "fmt"

type EngineBindingReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

type EngineBinding struct {
	ID              string                 `json:"binding_id"`
	EngineID        string                 `json:"engine_id"`
	EngineVersion   string                 `json:"engine_version"`
	CapabilityID    string                 `json:"capability_id"`
	ContractVersion string                 `json:"contract_version"`
	Entrypoint      string                 `json:"entrypoint"`
	Transport       EngineBindingReference `json:"transport_ref"`
	Permission      *bool                  `json:"permission_grant"`
}

type EngineBindings struct {
	Bindings []EngineBinding `json:"bindings"`
}

// Select resolves the exact current capability within the compatibility list
// for engineVersion. Transport verification remains the caller's responsibility.
func (doc EngineBindings) Select(engineID, engineVersion, capabilityID, contractVersion string) (EngineBinding, error) {
	if engineID == "" || !enginePattern.MatchString(engineVersion) || capabilityID == "" || contractVersion == "" {
		return EngineBinding{}, fmt.Errorf("incomplete engine compatibility selection")
	}
	seenIDs := make(map[string]bool, len(doc.Bindings))
	seenTargets := make(map[[3]string]bool, len(doc.Bindings))
	var selected EngineBinding
	for _, row := range doc.Bindings {
		if row.ID == "" || row.EngineID != engineID || !enginePattern.MatchString(row.EngineVersion) ||
			row.CapabilityID == "" || row.ContractVersion == "" || row.Entrypoint == "" ||
			row.Permission == nil || *row.Permission || row.Transport.ID == "" ||
			row.Transport.Version == "" || row.Transport.Path == "" || row.Transport.SHA256 == "" {
			return EngineBinding{}, fmt.Errorf("invalid execution compatibility binding %q", row.ID)
		}
		target := [3]string{row.EngineVersion, row.CapabilityID, row.ContractVersion}
		if seenIDs[row.ID] || seenTargets[target] {
			return EngineBinding{}, fmt.Errorf("ambiguous execution compatibility binding %q", row.ID)
		}
		seenIDs[row.ID], seenTargets[target] = true, true
		if target == [3]string{engineVersion, capabilityID, contractVersion} {
			selected = row
		}
	}
	if selected.ID == "" {
		return EngineBinding{}, fmt.Errorf("no execution compatibility binding for engine %s version %s and contract %s version %s", engineID, engineVersion, capabilityID, contractVersion)
	}
	return selected, nil
}
