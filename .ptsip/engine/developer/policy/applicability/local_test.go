package applicability

import (
    "strings"
    "testing"
)

func TestBuildTriageStableAndNonAuthoritative(t *testing.T) {
    head := strings.Repeat("a", 40)
    input := Input{SourceHead:head, Policies:[]Policy{
        {
            ID:"MPD-ARCH-0001", Plane:"PTSIP_DEVELOPER_POLICY",
            Path:"developer/policy/ARCH/MPD-ARCH-0001.yaml",
            Rules:[]Rule{{ID:"R1", SourceRef:"SRC-0002"}, {ID:"R2", SourceRef:"SRC-0003"}},
            Items:[]SourceItem{
                {ID:"SRC-0001", SourcePointer:"/other_evidence"},
                {ID:"SRC-0002", SourcePointer:"/rules/0"},
                {ID:"SRC-0003", SourcePointer:"/rules/1"},
            },
        },
        {
            ID:"SFP-GOV-0001", Plane:"PTSIP_SUPPORT_FEATURE",
            Path:"src/policy/GOV/SFP-GOV-0001.yaml",
            Rules:[]Rule{{ID:"R3", SourceRef:"SRC-0004"}},
            Items:[]SourceItem{{ID:"SRC-0004", SourcePointer:"/rules/3"}},
        },
    }}
    result := Output{
        SourceHead:head, BindingCount:3,
        Counts:map[string]int{"UNRESOLVED":2, "INELIGIBLE_POLICY":1},
        Decisions:[]Decision{
            {PolicyID:"MPD-ARCH-0001", RuleID:"R1", ContractKind:"RA",
                PolicyPath:"developer/policy/ARCH/MPD-ARCH-0001.yaml",
                SourcePointer:"/rules/0", State:"UNRESOLVED", Reason:"NO_CONFIRMED_CONTRACT_CONSUMPTION",
                EvidenceRefs:[]string{"SRC-0001", "SRC-0002"}},
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
    example := tr.Groups[0].Examples[0]
    if example.SourceItemID != "SRC-0002" || example.SourcePointer != "/rules/0" {
        t.Fatalf("expected the rule's source item and restored pointer, got %+v", example)
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
    for _, change := range []func(*Decision){
        func(d *Decision) { d.SourcePointer = "/other_evidence" },
        func(d *Decision) { d.PolicyPath = "developer/policy/ARCH/MPD-ARCH-0002.yaml" },
        func(d *Decision) { d.RuleID = "R-UNKNOWN" },
    } {
        bad = result
        bad.Decisions = append([]Decision(nil), result.Decisions...)
        change(&bad.Decisions[0])
        if _, err := BuildTriage(input, bad); err == nil || !strings.Contains(err.Error(), "TRIAGE_RULE_SOURCE_MISMATCH") {
            t.Fatalf("expected triage to reject a mismatched rule locator, got %v", err)
        }
    }
}
