package releasecontroller

import (
    "fmt"
)

// AdmitCTRLPublicationPolicy is the publication-only authority boundary.
// The retired developer policy resolver and the non-normative design checker
// cannot grant admission. Current owner direction is approved, but publication
// is gated on a separately validated normative successor.
func AdmitCTRLPublicationPolicy(root string) error {
    var owner struct {
        Schema string `json:"schema_version"`
        Decision string `json:"owner_decision"`
        Responsibility struct {
            Admission string `json:"admission_decision"`
            Verification string `json:"exact_source_verification_evidence"`
        } `json:"responsibility"`
        Activation struct {
            Direction bool `json:"direction_approved"`
            Document bool `json:"runtime_policy_document_complete"`
            Validator bool `json:"required_policy_schema_and_validator_complete"`
        } `json:"activation"`
    }
    if err := ReadJSON(root, "developer/policy/registries/release-admission-ctrl-ownership.json", &owner); err != nil {
        return fmt.Errorf("CTRL admission ownership contract unavailable: %w", err)
    }
    if owner.Schema != "ptsip-release-admission-owner-selection/v1" ||
        owner.Decision != "CTRL_CENTERED" ||
        owner.Responsibility.Admission != "CTRL" ||
        owner.Responsibility.Verification != "ASSURE" || !owner.Activation.Direction {
        return fmt.Errorf("CTRL/ASSURE release responsibility is not authorized")
    }

    var activation struct {
        Resolution string `json:"resolution"`
        Effective struct {
            Direction string `json:"publication_activation_direction"`
            Owner string `json:"admission_authority_owner"`
            Evidence string `json:"verification_evidence_owner"`
        } `json:"effective_decision"`
        Gates struct {
            Validated bool `json:"normative_successor_validated"`
            Enabled bool `json:"publish_workflow_enabled"`
            Admitted bool `json:"actual_publication_admitted"`
        } `json:"machine_gates"`
    }
    if err := ReadJSON(root, "developer/policy/registries/release-admission-publication-activation.json", &activation); err != nil {
        return fmt.Errorf("release activation contract unavailable: %w", err)
    }
    if activation.Resolution != "KEEP_LATEST" ||
        activation.Effective.Direction != "APPROVED" ||
        activation.Effective.Owner != "CTRL" ||
        activation.Effective.Evidence != "ASSURE" {
        return fmt.Errorf("unresolved Release Admission activation authority")
    }

    // Implementation flags are NOT policy. All must be true and an actual
    // registered successor normative validator must independently succeed.
    if !owner.Activation.Document || !owner.Activation.Validator ||
        !activation.Gates.Validated || !activation.Gates.Enabled || !activation.Gates.Admitted {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: effective CTRL normative policy and automated validation are not yet complete")
    }
    var index struct {
        Activation string `json:"normative_policy_activation"`
        Authorized bool `json:"publication_authorized"`
    }
    if err := ReadJSON(root, ".ptsip/contracts/release/index.json", &index); err != nil {
        return err
    }
    if index.Activation != "ACTIVE" || !index.Authorized {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: active normative policy admission not registered")
    }
    // This validator must be independently registered against an approved
    // family schema; absent/retired validators and arbitrary contract edits
    // cannot promote design-time evidence into publication authority.
    if _, err := Run(root, "go", "-C", ".ptsip/engine/developer", "run", ".",
        "--validator", "ctrl-release-admission-check", "--root", root); err != nil {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: CTRL normative admission verifier failed: %w", err)
    }
    return nil
}
