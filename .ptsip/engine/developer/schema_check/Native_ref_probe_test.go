//go:build ptsip_probe

// Diagnostic only: the installed PTSIP VALIDATION engine is the sole executor.
// External-ref limitations are reported, never promoted to policy conformance.
package schema_check

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

type nativeProbeOutput struct {
    Status string `json:"status"`
    Error any `json:"error"`
    Output struct {
        Valid bool `json:"valid"`
        Diagnostics []string `json:"diagnostics"`
        Verdict string `json:"ptsip_conformance_verdict"`
    } `json:"output"`
}

func readApprovedProbeSchema(t *testing.T, rel string) map[string]any {
    t.Helper()
    path := filepath.Join("..","..","..","..", filepath.FromSlash(rel))
    contents, err := os.ReadFile(path)
    if err != nil { t.Fatal(err) }
    var schema map[string]any
    if err := json.Unmarshal(contents,&schema); err != nil { t.Fatal(err) }
    return schema
}

func callRealEngine(t *testing.T, engine Engine, label string, schema, instance any) (nativeProbeOutput, error) {
    t.Helper()
    request := map[string]any{
        "engine_id":engine.EngineID, "engine_version":engine.EngineVersion,
        "contract_id":engine.ContractID, "contract_version":engine.ContractVersion,
        "operation_id":"validation.schema",
        "input":map[string]any{"schema":schema,"instance":instance},
    }
    payload, err := json.Marshal(request)
    if err != nil { t.Fatal(err) }
    cmd := exec.Command(engine.Executable)
    cmd.Stdin = bytes.NewReader(payload)
    output, err := cmd.CombinedOutput()
    var response nativeProbeOutput
    jsonErr := json.Unmarshal(output, &response)
    clipped := strings.TrimSpace(string(output))
    if len(clipped)>2400 {clipped=clipped[:2400]+"...[truncated]" }
    t.Logf("ENGINE_PROBE case=%s exit_error=%v parse_error=%v native_result=%s",label,err,jsonErr,clipped)
    if jsonErr!=nil {return response,fmt.Errorf("native response not JSON: %v",jsonErr)}
    return response,err
}

func TestInstalledPTSIP038b3InlineAndExternalRefExperiment(t *testing.T) {
    engine,err:=installed()
    if err!=nil {t.Fatalf("installed official engine unavailable: %v",err)}
    if engine.PackageVersion!="0.3.8b3" {t.Fatalf("required PTSIP 0.3.8b3, got %s",engine.PackageVersion)}
    t.Logf("ENGINE_PROBE installed=%s engine_version=%s operation=validation.schema",engine.PackageVersion,engine.EngineVersion)
    version:=readApprovedProbeSchema(t,".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json")
    publicStatus:=readApprovedProbeSchema(t,".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json")
    constant:=readApprovedProbeSchema(t,".ptsip/contracts/schemas/Policy_Status/policy_status_constant.schema.json")

    valid,err:=callRealEngine(t,engine,"approved_version_valid",version,map[string]any{"version":"1.0"})
    if err!=nil||valid.Status!="PASS"||!valid.Output.Valid||valid.Output.Verdict!="NOT_ASSERTED" {
        t.Errorf("inline approved version positive check unexpected: status=%s valid=%v err=%v",valid.Status,valid.Output.Valid,err)
    }
    invalid,err:=callRealEngine(t,engine,"approved_version_required_missing",version,map[string]any{})
    if err!=nil||invalid.Status!="PASS"||invalid.Output.Valid||invalid.Output.Verdict!="NOT_ASSERTED" {
        t.Errorf("inline approved version negative check unexpected: status=%s valid=%v err=%v",invalid.Status,invalid.Output.Valid,err)
    }
    invalidType,err:=callRealEngine(t,engine,"approved_version_wrong_type",version,map[string]any{"version":123})
    if err!=nil||invalidType.Status!="PASS"||invalidType.Output.Valid||invalidType.Output.Verdict!="NOT_ASSERTED" {
        t.Errorf("inline approved version type check unexpected: status=%s valid=%v err=%v",invalidType.Status,invalidType.Output.Valid,err)
    }

    statusValid:=map[string]any{"status":"DRAFT","substatus":"Default","contract":"urn:ptsip:contract:policy:arch:identity:status"}
    statusMissingContract:=map[string]any{"status":"DRAFT","substatus":"Default"}
    // In this contract, external refs to lifecycle status contracts are intentional.
    _,_ = callRealEngine(t,engine,"real_status_external_ref_valid",constant,statusValid)
    _,_ = callRealEngine(t,engine,"real_status_external_ref_missing_contract",constant,statusMissingContract)
    envelope:=map[string]any{"Schema":map[string]any{"Policy_Status_version":map[string]any{"version":"1.0"}},"constant":statusValid}
    _,_ = callRealEngine(t,engine,"real_public_status_external_ref",publicStatus,envelope)
    missingVersion:=map[string]any{"Schema":map[string]any{"Policy_Status_version":map[string]any{}},"constant":statusValid}
    _,_ = callRealEngine(t,engine,"real_public_status_missing_nested_version",publicStatus,missingVersion)
    externalSchema:=map[string]any{"type":"object","properties":map[string]any{"linked":map[string]any{"$ref":version["$id"]}}}
    _,_ = callRealEngine(t,engine,"synthetic_external_ref_present_in_repository_only",externalSchema,map[string]any{"linked":map[string]any{}})
    // Do not assert external-ref PASS or FAIL: this probe reports the actual
    // native response while preserving the current schema/authority boundaries.
}
