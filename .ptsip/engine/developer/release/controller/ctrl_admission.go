package releasecontroller

import (
    "fmt"
    "encoding/json"
)

// AdmitCTRLPublicationPolicy validates the current normative CTRL release
// policy, not a mutable "publication admitted" flag. Per-run evidence is
// independently required by AdmitVerification and the shipping manifest.
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
        owner.Responsibility.Verification != "ASSURE" ||
        !owner.Activation.Direction || !owner.Activation.Document || !owner.Activation.Validator {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: CTRL/ASSURE normative release owner not active")
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
        activation.Effective.Evidence != "ASSURE" ||
        !activation.Gates.Validated || !activation.Gates.Enabled ||
        activation.Gates.Admitted {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: invalid conditional admission activation")
    }
    // The v1 derived design report intentionally continues to report
    // publication_authorized=false: it is NOT a normative permission source.
    var index struct {
        Policy string `json:"effective_ctrl_policy_ref"`
        Validator string `json:"effective_ctrl_validator"`
        Schema string `json:"effective_ctrl_schema_ref"`
        Authority string `json:"effective_ctrl_policy_status"`
        DesignAuthorizes bool `json:"publication_authorized"`
    }
    if err := ReadJSON(root, ".ptsip/contracts/release/index.json", &index); err != nil {
        return err
    }
    if index.Policy != "developer/policy/CTRL/release-admission.policy.json" ||
        index.Validator != "ctrl-release-admission-check" ||
        index.Schema != ".ptsip/contracts/schemas/release/ctrl-release-admission.schema.json" ||
        index.Authority != "ACTIVE_PER_RELEASE_CONDITIONAL" || index.DesignAuthorizes {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: active CTRL policy registration invalid")
    }
    raw, err := Run(root, "go", "-C", ".ptsip/engine/developer", "run", ".",
        "--validator", "ctrl-release-admission-check", "--root", root)
    if err != nil {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: CTRL normative admission verifier failed: %w", err)
    }
    var report struct {
        Status string `json:"status"`
        Evidence string `json:"per_release_evidence"`
        Publication bool `json:"publication_authorized"`
        Owner string `json:"admission_owner"`
    }
    if err := json.Unmarshal(raw,&report);err != nil{return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: invalid independent verifier result: %w",err)}
    if report.Status!="ACTIVE_POLICY_VERIFIED" || report.Owner!="CTRL" ||
        report.Evidence!="NOT_ASSERTED" || report.Publication {
        return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: normative policy verifier did not return conditional admission")
    }
    return nil
}
