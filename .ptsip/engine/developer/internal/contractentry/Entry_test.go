package contractentry

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func fixture(t *testing.T) string {
    t.Helper()
    root:=t.TempDir()
    put:=func(p string, data any) {
        t.Helper()
        file:=filepath.Join(root,filepath.FromSlash(p))
        if err:=os.MkdirAll(filepath.Dir(file),0755);err!=nil {t.Fatal(err)}
        var raw []byte
        if s,ok:=data.(string);ok {raw=[]byte(s)} else {var err error;raw,err=json.Marshal(data);if err!=nil {t.Fatal(err)}}
        if err:=os.WriteFile(file,raw,0644);err!=nil{t.Fatal(err)}
    }
    put(".ptsip/index.json",map[string]any{"namespaces":map[string]any{"contracts":map[string]any{"status":"ACTIVE","index":"contracts/index.json"}}})
    put(".ptsip/contracts/index.json",map[string]any{"policy":"policy/boundary.json", "release_documentation":"release/Platform/Common/releasenote.json", "authoring":map[string]any{"entrypoint":".ptsip/engine/developer","validators":[]string{"policy-rules","responsibility-authority"},"execution_authorized":false}})
    put(".ptsip/engine/developer/index.json",map[string]any{"policy_authority":"NONE","execution_authorized":false,"validators":map[string]any{"contract-entry":"policy/index.json","policy-check":"policy/index.json"}})
    put(".ptsip/contracts/policy/boundary.json",map[string]any{"domain":"policy","routing_binding":map[string]any{"data":".ptsip/contracts/policy/routing/policy_routes.json"},"rules_binding":map[string]any{"family_source":".ptsip/contracts/policy/identity/root_family.json","directory":".ptsip/contracts/policy/Rules","filename":"{root_family}_rules.json","selection_schema":".ptsip/contracts/schemas/Policy/Policy_Rules/Public_Structure_Rules.schema.json","responsibility_profile":".ptsip/profiles/Responsibility-Authority-list.yaml"}})
    put(".ptsip/contracts/policy/routing/policy_routes.json",map[string]any{"identity_sources":map[string]any{"policy_kind":".ptsip/contracts/policy/identity/policy_kind.json"},"routes":map[string]any{"PTSIP_DEVELOPER_POLICY":map[string]any{"output_root":"developer/policy"}},"schema_refs":map[string]any{"public":".ptsip/contracts/schemas/Policy/PTSIP_Public_Structure.schema.json"}})
    put(".ptsip/contracts/policy/identity/policy_kind.json",map[string]any{"enum":[]string{"PTSIP_DEVELOPER_POLICY"}})
    put(".ptsip/contracts/policy/identity/root_family.json",map[string]any{"enum":[]string{"GOV"}})
    put(".ptsip/contracts/policy/Rules/GOV_rules.json",map[string]any{"root_family":"GOV"})
    put(".ptsip/contracts/schemas/Policy/PTSIP_Public_Structure.schema.json",map[string]any{"$id":"urn:public"})
    put(".ptsip/contracts/schemas/Policy/Policy_Rules/Public_Structure_Rules.schema.json",map[string]any{"$id":"urn:rules"})
    put(".ptsip/profiles/Responsibility-Authority-list.yaml","responsibility: []\n")
    put("developer/policy/GOV/MPD-GOV-0006.yaml", "Policy_Identity:\n  policy_kind: PTSIP_DEVELOPER_POLICY\n  root_family: GOV\nPolicy_Status:\n  status: APPROVED.Applying\n")
    return root
}

func TestPolicySelectionIgnoresUnrelatedMissingReleaseContract(t *testing.T){
    root:=fixture(t)
    got,err:=Resolve(root,"developer/policy/GOV/MPD-GOV-0006.yaml")
    if err!=nil{t.Fatal(err)}
    if got.Validator!="policy-check"||got.RootFamily!="GOV"||got.ExecutionAuthorized||got.MutationAuthorized{t.Fatalf("invalid selection: %+v",got)}
    if got.ContractRefs["family"]!=".ptsip/contracts/policy/Rules/GOV_rules.json"{t.Fatalf("wrong family ref: %+v",got)}
}

func TestMissingSelectedContractFailsClosed(t *testing.T){
    root:=fixture(t)
    if err:=os.Remove(filepath.Join(root,".ptsip/contracts/policy/Rules/GOV_rules.json"));err!=nil{t.Fatal(err)}
    if _,err:=Resolve(root,"developer/policy/GOV/MPD-GOV-0006.yaml");err==nil||!strings.Contains(err.Error(),"MISSING_REGISTERED_FILE") {t.Fatalf("missing selected contract accepted: %v",err)}
}

func TestUnknownAndMismatchedIdentityFailClosed(t *testing.T){
    for _,example:=range []string{
        "Policy_Identity:\n  policy_kind: PTSIP_DEVELOPER_POLICY\n  root_family: UNKNOWN\n",
        "Policy_Identity:\n  policy_kind: PTSIP_SUPPORT_FEATURE\n  root_family: GOV\n",
        "Policy_Identity:\n  policy_kind: PTSIP_DEVELOPER_POLICY\n  root_family: GOV\n  root_family: GOV\n",
        "Policy_Identity:\n  root_family: GOV\n",
        "Policy_Identity: {policy_kind: PTSIP_DEVELOPER_POLICY, root_family: GOV}\n",
    } {
        root:=fixture(t)
        if err:=os.WriteFile(filepath.Join(root,"developer/policy/GOV/MPD-GOV-0006.yaml"),[]byte(example),0644);err!=nil{t.Fatal(err)}
        if _,err:=Resolve(root,"developer/policy/GOV/MPD-GOV-0006.yaml");err==nil{t.Errorf("accepted invalid selector: %q", example)}
    }
}

func TestWrongDomainAndTraversalFailClosed(t *testing.T){
    root:=fixture(t)
    for _,target:=range []string{"../outside.yaml","developer/policy/GOV/../../GOV/MPD-GOV-0006.yaml","src/policy/ARCH/SFP-ARCH-0005.yaml"}{
        if _,err:=Resolve(root,target);err==nil{t.Fatalf("accepted unregistered target: %s",target)}
    }
}

func TestAuthorityEscalationFailsClosed(t *testing.T){
    root:=fixture(t)
    filename:=filepath.Join(root,filepath.FromSlash(".ptsip/engine/developer/index.json"))
    if err:=os.WriteFile(filename,[]byte(`{"policy_authority":"NONE","execution_authorized":true,"validators":{"contract-entry":"policy/index.json","policy-check":"policy/index.json"}}`),0644);err!=nil{t.Fatal(err)}
    if _,err:=Resolve(root,"developer/policy/GOV/MPD-GOV-0006.yaml");err==nil||!strings.Contains(err.Error(),"AUTHORITY_BOUNDARY_CHANGED"){t.Fatalf("accepted authority escalation: %v",err)}
}
