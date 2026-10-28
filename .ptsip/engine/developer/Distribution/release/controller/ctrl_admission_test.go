package releasecontroller

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestCTRLAdmissionFailClosedWithoutActiveNormativePolicy(t *testing.T) {
    root := t.TempDir()
    if err := AdmitCTRLPublicationPolicy(root); err == nil {
        t.Fatal("missing CTRL admission ownership permitted publication")
    }
    owner := map[string]any{
        "schema_version": "ptsip-release-admission-owner-selection/v1",
        "owner_decision": "CTRL_CENTERED",
        "responsibility": map[string]any{
            "admission_decision": "CTRL",
            "exact_source_verification_evidence": "ASSURE",
        },
        "activation": map[string]any{
            "direction_approved": true,
            "runtime_policy_document_complete": false,
            "required_policy_schema_and_validator_complete": false,
        },
    }
    activation := map[string]any{
        "resolution": "KEEP_LATEST",
        "effective_decision": map[string]any{
            "publication_activation_direction": "APPROVED",
            "admission_authority_owner": "CTRL",
            "verification_evidence_owner": "ASSURE",
        },
        "machine_gates": map[string]any{
            "normative_successor_validated": false,
            "publish_workflow_enabled": false,
            "actual_publication_admitted": false,
        },
    }
    for path, value := range map[string]any{
        "developer/policy/registries/release-admission-ctrl-ownership.json": owner,
        "developer/policy/registries/release-admission-publication-activation.json": activation,
    } {
        p := filepath.Join(root, filepath.FromSlash(path))
        if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil { t.Fatal(err) }
        if err := WriteJSON(p, value); err != nil { t.Fatal(err) }
    }
    if err := AdmitCTRLPublicationPolicy(root); err == nil ||
        !strings.Contains(err.Error(), "RELEASE_PUBLICATION_BLOCKED") {
        t.Fatalf("unvalidated CTRL direction was promoted to publication: %v", err)
    }
    owner["activation"].(map[string]any)["runtime_policy_document_complete"] = true
    owner["activation"].(map[string]any)["required_policy_schema_and_validator_complete"] = true
    activation["machine_gates"].(map[string]any)["normative_successor_validated"] = true
    activation["machine_gates"].(map[string]any)["publish_workflow_enabled"] = true
    activation["machine_gates"].(map[string]any)["actual_publication_admitted"] = true
    if err := WriteJSON(filepath.Join(root, "developer/policy/registries/release-admission-ctrl-ownership.json"), owner); err != nil { t.Fatal(err) }
    if err := WriteJSON(filepath.Join(root, "developer/policy/registries/release-admission-publication-activation.json"), activation); err != nil { t.Fatal(err) }
    // Even forgeable implementation flags cannot replace the active policy index.
    if err := AdmitCTRLPublicationPolicy(root); err == nil {
        t.Fatal("forged completion flags permitted publication without normative index")
    }
}

func TestCTRLAdmissionBlocksWithdrawnPolicy(t *testing.T) {
    root, err := filepath.Abs("../../../../../..")
    if err != nil { t.Fatal(err) }
    err = AdmitCTRLPublicationPolicy(root)
    if err == nil || !strings.Contains(err.Error(), "RELEASE_PUBLICATION_BLOCKED") {
        t.Fatalf("withdrawn CTRL policy unexpectedly admitted: %v", err)
    }
}
