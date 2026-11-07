package applicability

import (
    "strings"
    "testing"
)

const testHead = "1111111111111111111111111111111111111111"

func fixture() Input {
    contracts := map[string]Contract{}
    for _, kind := range kinds {
        contracts[kind] = Contract{
            SourceRef: ".ptsip/contracts/compile-sealed-rbac/" + strings.ToLower(kind) + "/owner.json",
            ActivationState: "UNREGISTERED",
        }
    }
    return Input{
        SchemaVersion: InputVersion, SourceHead: testHead, Contracts: contracts,
        Policies: []Policy{{
            ID: "MPD-ARCH-0001", Path: "developer/policy/ARCH/MPD-ARCH-0001.yaml",
            Plane: "PTSIP_DEVELOPER_POLICY", Family: "ARCH",
            CatalogStatus: "APPROVED", DocumentStatus: "APPROVED.Applying",
            Items: []SourceItem{{ID: "SRC-0001", SourcePointer: "/Policy_Rules/Definition/0"}},
            Rules: []Rule{{
                ID: "RULE-1", Kind: "RULE", SourceRef: "SRC-0001",
                ContractBindings: map[string]Binding{
                    "RA": {State: "UNRESOLVED", BasisRefs: []string{"SRC-0001"}},
                    "RAC": {State: "NOT_APPLICABLE", BasisRefs: []string{"SRC-0001"}},
                    "AEC": {State: "DECLARED", BasisRefs: []string{"SRC-0001"}, ContractRef: &Reference{
                        SourceRef: ".ptsip/contracts/compile-sealed-rbac/aec/owner.json",
                        ItemRef: "#", RevisionRef: "git:" + testHead,
                    }},
                },
            }},
        }},
    }
}
func TestAssessExplicitEvidenceAndNoRuntimeGrant(t *testing.T) {
    out, err := Assess(fixture())
    if err != nil { t.Fatal(err) }
    if out.PolicyCount != 1 || out.RuleCount != 1 || out.BindingCount != 3 {
        t.Fatalf("unexpected summary: %+v", out)
    }
    if out.Counts["UNRESOLVED"] != 1 || out.Counts["NOT_APPLICABLE_CANDIDATE"] != 1 ||
        out.Counts["APPLICABLE_CANDIDATE"] != 1 {
        t.Fatalf("wrong candidate counts: %+v", out.Counts)
    }
    if out.EvaluationAuthorized || out.ExecutionAuthorized || out.PolicyMutationAuthorized {
        t.Fatal("applicability cannot grant authority")
    }
}
func TestMigratedDefinitionReferenceIsNotConsumption(t *testing.T) {
    in := fixture()
    b := in.Policies[0].Rules[0].ContractBindings["RA"]
    b.ContractRef = &Reference{
        SourceRef: in.Contracts["RA"].SourceRef, ItemRef: "#", RevisionRef: "git:" + testHead,
    }
    in.Policies[0].Rules[0].ContractBindings["RA"] = b
    out, err := Assess(in)
    if err != nil { t.Fatal(err) }
    for _, row := range out.Decisions {
        if row.ContractKind == "RA" && (row.State != "UNRESOLVED" ||
            row.Reason != "REFERENCE_EXISTS_BUT_CONSUMPTION_UNRESOLVED") {
            t.Fatalf("definition reference incorrectly promoted: %+v", row)
        }
    }
}
func TestFailClosedOnSourceAndRegistryConflicts(t *testing.T) {
    cases := map[string]func(*Input){
        "wrong owner": func(i *Input) { c := i.Contracts["AEC"]; c.SourceRef = ".ptsip/contracts/compile-sealed-rbac/aec/other.json"; i.Contracts["AEC"] = c },
        "mismatched policy": func(i *Input) { i.Policies[0].CatalogStatus = "RETIRED" },
        "duplicate rule": func(i *Input) { i.Policies[0].Rules = append(i.Policies[0].Rules, i.Policies[0].Rules[0]) },
        "missing source": func(i *Input) { i.Policies[0].Items = nil },
        "bad sha": func(i *Input) { i.SourceHead = "latest" },
        "unauthorized kind": func(i *Input) { i.Policies[0].Rules[0].ContractBindings["BAD"] = Binding{State: "UNRESOLVED"} },
    }
    for name, mutate := range cases {
        t.Run(name, func(t *testing.T) {
            in := fixture()
            mutate(&in)
            if _, err := Assess(in); err == nil { t.Fatalf("expected failure: %s", name) }
        })
    }
}
func TestRetiredPolicyIsNotPromoted(t *testing.T) {
    in := fixture()
    in.Policies[0].CatalogStatus = "RETIRED"
    in.Policies[0].DocumentStatus = "RETIRED.Waiting"
    out, err := Assess(in)
    if err != nil { t.Fatal(err) }
    if out.Counts["INELIGIBLE_POLICY"] != 3 { t.Fatal(out.Counts) }
}
