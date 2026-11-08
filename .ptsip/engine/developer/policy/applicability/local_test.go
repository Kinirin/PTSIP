package applicability

import (
    "strings"
    "testing"
)

func TestBuildTriageStableAndNonAuthoritative(t *testing.T) {
    head := strings.Repeat("a", 40)
    input := Input{SourceHead:head, Policies:[]Policy{
        {ID:"MPD-ARCH-0001", Plane:"PTSIP_DEVELOPER_POLICY"},
        {ID:"SFP-GOV-0001", Plane:"PTSIP_SUPPORT_FEATURE"},
    }}
    result := Output{
        SourceHead:head, BindingCount:3,
        Counts:map[string]int{"UNRESOLVED":2, "INELIGIBLE_POLICY":1},
        Decisions:[]Decision{
            {PolicyID:"MPD-ARCH-0001", RuleID:"R1", ContractKind:"RA",
                PolicyPath:"developer/policy/ARCH/MPD-ARCH-0001.yaml",
                SourcePointer:"/rules/0", State:"UNRESOLVED", Reason:"NO_CONFIRMED_CONTRACT_CONSUMPTION"},
            {PolicyID:"MPD-ARCH-0001", RuleID:"R2", ContractKind:"RA",
                PolicyPath:"developer/policy/ARCH/MPD-ARCH-0001.yaml",
                SourcePointer:"/rules/1", State:"UNRESOLVED", Reason:"NO_CONFIRMED_CONTRACT_CONSUMPTION"},
            {PolicyID:"SFP-GOV-0001", RuleID:"R3", ContractKind:"AEC",
                PolicyPath:"src/policy/GOV/SFP-GOV-0001.yaml",
                SourcePointer:"/rules/3", State:"INELIGIBLE_POLICY", Reason:"POLICY_LIFECYCLE_NOT_EFFECTIVE"},
        },
    }
    tr, err := BuildTriage(input,result)
    if err != nil { t.Fatal(err) }
    if tr.PolicyMutationAuthorized || tr.EvaluationAuthorized || tr.ExecutionAuthorized {
        t.Fatal("derived triage cannot grant authority")
    }
    if len(tr.Groups)!=2 || tr.Groups[0].Count!=2 || tr.Groups[1].Count!=1 {
        t.Fatalf("unexpected groups: %+v",tr.Groups)
    }
    bad := result
    bad.BindingCount = 4
    if _, err := BuildTriage(input,bad); err == nil {
        t.Fatal("binding count mismatch must be rejected")
    }
    bad = result
    bad.SourceHead = strings.Repeat("b", 40)
    if _, err := BuildTriage(input,bad); err == nil {
        t.Fatal("stale source mismatch must be rejected")
    }
}
