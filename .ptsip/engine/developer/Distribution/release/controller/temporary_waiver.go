package releasecontroller

import (
    "fmt"
    "strings"
)

// LightweightPackageWaiver is approved only for the first complete 0.3.8b6
// package with pp.2.00. It does not waive platform receipts, file digests,
// the source SHA, native engines, installed version, or published immutability.
func LightweightPackageWaiver(root string, s Selection) (bool,error) {
    c,err:=ReadComposition(root)
    if err!=nil{return false,err}
    if s.Type!="PTSIP_PACKAGE" || c.PackagePyPI!="0.3.8b6" || c.Profile!="pp.2.00" {
        return false,nil
    }
    const ref="developer/policy/registries/release-20261026-package-join-waiver.json"
    var raw struct {
        Schema string `json:"schema_version"`
        Decision string `json:"decision"`
        State string `json:"state"`
        Approval string `json:"approval_ref"`
        Scope struct {
            Project string `json:"project"`
            Package string `json:"package_version"`
            Profile string `json:"profile_version"`
            Release string `json:"release_type"`
            Platforms []string `json:"platforms"`
        } `json:"scope"`
        Exemption struct {
            Full string `json:"registered_full_package_join"`
            Source string `json:"source_bound_component_builds"`
            Artifact string `json:"immutable_original_artifact_sha256"`
            Install string `json:"lightweight_package_install"`
            Version string `json:"lightweight_package_version"`
            Selection string `json:"release_selection_sha256"`
            SHA string `json:"exact_source_sha"`
            Fallback string `json:"fallback_to_old_sha"`
            Immutability string `json:"previous_published_component_immutability"`
            Destinations string `json:"extra_destinations"`
        } `json:"exemption"`
        End string `json:"end_condition"`
    }
    if err:=ReadJSON(root,ref,&raw);err!=nil{return false,fmt.Errorf("missing approved temporary waiver: %w",err)}
    if raw.Schema!="ptsip-release-one-time-lightweight-join-waiver/v1" ||
      raw.Decision!="RECONCILE" || raw.State!="ACTIVE_UNTIL_FIRST_SUCCESSFUL_PUBLICATION_RECORD" ||
      raw.Approval!="developer/policy/approvals/MPA-20261026-RELEASE-JOIN-RECONCILE.yaml" ||
      raw.Scope.Project!="PTSIP" || raw.Scope.Package!="0.3.8b6" || raw.Scope.Profile!="pp.2.00" ||
      raw.Scope.Release!="PTSIP_PACKAGE" || strings.Join(raw.Scope.Platforms,",")!="windows-x64,linux-x64" ||
      raw.Exemption.Full!="WAIVED_FOR_THIS_PACKAGE_ONLY" ||
      raw.Exemption.Source!="REQUIRED" || raw.Exemption.Artifact!="REQUIRED" ||
      raw.Exemption.Install!="REQUIRED" || raw.Exemption.Version!="REQUIRED" ||
      raw.Exemption.Selection!="REQUIRED" || raw.Exemption.SHA!="REQUIRED" ||
      raw.Exemption.Fallback!="FORBIDDEN" || raw.Exemption.Immutability!="REQUIRED" ||
      raw.Exemption.Destinations!="FORBIDDEN" ||
      raw.End!="FIRST_SUCCESSFUL_PUBLICATION_RECORD_FOR_THIS_PACKAGE_VERSION" {
        return false,fmt.Errorf("temporary Package verification waiver differs from owner approval")
    }
    if strings.Join(s.Platforms,",")!="windows-x64,linux-x64" || len(s.Components)!=9 {
        return false,fmt.Errorf("temporary waiver requires complete original host/component coverage")
    }
    return true,nil
}
