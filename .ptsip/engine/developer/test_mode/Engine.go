// Package test_mode validates registry structure through the installed PTSIP
// engine and checks explicit references. It does not resolve changed-file modes.
package test_mode

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path"
    "path/filepath"
    "sort"
    "strings"

    schema_check "github.com/Kinirin/PTSIP/.ptsip/engine/developer/schema_check"
)

const registryRef = ".github/test_modes.yaml"
const profileRef = ".ptsip/profiles/main.ptsip.yaml"
const schemaRef = "developer/contracts/test-mode-control-plane/registry-structure.schema.json"

type Report struct {
    Status string `json:"status"`
    Scope string `json:"scope"`
    StructureEngine string `json:"structure_engine"`
    RegistryConformance string `json:"registry_conformance"`
    FullIndependence string `json:"full_independence"`
    Findings []string `json:"findings"`
}

func yamlRecord(root, relative string) (map[string]any, error) {
    raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
    if err != nil { return nil, err }
    python := os.Getenv("PTSIP_PYTHON")
    if python == "" { python = "python" }
    cmd := exec.Command(python, "-c", "import json,sys,yaml; print(json.dumps(yaml.safe_load(sys.stdin.read())))")
    cmd.Stdin = bytes.NewReader(raw)
    output, err := cmd.CombinedOutput()
    if err != nil { return nil, fmt.Errorf("YAML parse %s: %w: %s", relative, err, output) }
    var record map[string]any
    if err = json.Unmarshal(output, &record); err != nil { return nil, err }
    if record == nil { return nil, fmt.Errorf("YAML root %s not an object", relative) }
    return record, nil
}
func items(v any) []any { a, _ := v.([]any); return a }
func object(v any) map[string]any { m, _ := v.(map[string]any); return m }
func stringValue(v any) string { s, _ := v.(string); return s }
func hasRole(c map[string]any, role string) bool {
    for _, v := range items(c["roles"]) { if stringValue(v) == role { return true } }
    return false
}
func safeTarget(root, relative string) (string, error) {
    if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "\\") {
        return "", fmt.Errorf("unsafe execution target %q", relative)
    }
    for _, piece := range strings.Split(relative, "/") {
        if piece == "" || piece == "." || piece == ".." { return "", fmt.Errorf("unsafe execution target %q", relative) }
    }
    if filepath.Clean(relative) != filepath.FromSlash(relative) { return "", fmt.Errorf("noncanonical target %q", relative) }
    targetPath := filepath.Join(root, filepath.FromSlash(relative))
    st, err := os.Stat(targetPath)
    if err != nil { return "", fmt.Errorf("unavailable target %q: %w", relative, err) }
    if !st.IsDir() && !st.Mode().IsRegular() { return "", fmt.Errorf("unsupported target %q", relative) }
    return targetPath, nil
}
func selectorCoversFile(pattern, relative string) bool {
    if pattern == relative { return true }
    if strings.HasSuffix(pattern, "/**") {
        prefix := strings.TrimSuffix(pattern, "/**")
        return relative == prefix || strings.HasPrefix(relative, prefix+"/")
    }
    matched, err := path.Match(pattern, relative)
    return err == nil && matched
}
// This verifies declared execution-target ownership, not changed-file mode selection.
func ownedTarget(root, target, targetPath string, component map[string]any) bool {
    patterns := items(component["include"])
    for _, p := range patterns { if selectorCoversFile(stringValue(p), target) { return true } }
    st, err := os.Stat(targetPath)
    if err != nil || !st.IsDir() { return false }
    found := false
    _ = filepath.WalkDir(targetPath, func(candidate string, entry os.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if entry.IsDir() { return nil }
        relative, err := filepath.Rel(root, candidate)
        if err != nil { return err }
        relative = filepath.ToSlash(relative)
        for _, p := range patterns {
            if selectorCoversFile(stringValue(p), relative) { found = true; return filepath.SkipAll }
        }
        return nil
    })
    return found
}
func contractErrors(root string, registry, profile map[string]any) []string {
    findings := []string{}
    components := map[string]map[string]any{}
    for _, v := range items(profile["components"]) {
        c := object(v); id := stringValue(c["id"])
        if id == "" { findings = append(findings, "PROFILE_COMPONENT_WITHOUT_ID"); continue }
        if _, ok := components[id]; ok { findings = append(findings, "DUPLICATE_PROFILE_COMPONENT:"+id) }
        components[id] = c
    }
    grouped := map[string]int{}
    for group, members := range object(registry["groups"]) {
        if group == "" { findings = append(findings, "EMPTY_GROUP_ID") }
        for _, raw := range items(members) {
            id := stringValue(raw)
            if id == "" { findings = append(findings, "EMPTY_GROUP_MODE") }
            grouped[id]++
        }
    }
    seen := map[string]bool{}
    for _, v := range items(registry["modes"]) {
        mode := object(v); id := stringValue(mode["id"]); comp := stringValue(mode["component_ref"])
        if id == "" || seen[id] { findings = append(findings, "DUPLICATE_OR_EMPTY_MODE:"+id) }
        seen[id] = true
        if grouped[id] != 1 { findings = append(findings, "MODE_GROUP_MEMBERSHIP:"+id) }
        c, ok := components[comp]
        if !ok || !hasRole(c,"VERIFICATION") {
            findings = append(findings, "MISSING_VERIFICATION_OWNER:"+id)
            continue
        }
        execution := object(mode["execution"]); targets := []string{}
        for _, key := range []string{"pytest","go"} {
            for _, raw := range items(execution[key]) { targets = append(targets, stringValue(raw)) }
        }
        for _, key := range []string{"contract_ref","runner_ref"} {
            if v, ok := execution[key]; ok { targets = append(targets,stringValue(v)) }
        }
        if len(targets)==0 { findings = append(findings,"NO_TARGETS:"+id); continue }
        for _, target := range targets {
            full, err := safeTarget(root,target)
            if err != nil { findings = append(findings,"INVALID_TARGET:"+id+":"+err.Error()); continue }
            if !ownedTarget(root,target,full,c) { findings = append(findings,"TARGET_NOT_PROFILE_OWNED:"+id+":"+target) }
        }
    }
    for id, count := range grouped {
        if count != 1 || !seen[id] { findings = append(findings, "UNMATCHED_GROUP_MEMBER:"+id) }
    }
    if def:=stringValue(object(registry["defaults"])["workflow_mode"]); !seen[def] {
        findings = append(findings, "INVALID_DEFAULT_MODE:"+def)
    }
    sort.Strings(findings)
    return findings
}
func Run(root string) (Report, int) {
    report := Report{ Status:"FAIL", Scope:"TEST_MODE_REGISTRY_STRUCTURE_AND_REFERENCES_ONLY", StructureEngine:"INSTALLED_PTSIP_VALIDATION", RegistryConformance:"NOT_ASSERTED", FullIndependence:"NOT_ASSERTED", Findings:[]string{} }
    reg, err := yamlRecord(root,registryRef)
    if err != nil { report.Findings=append(report.Findings,err.Error()); return report,2 }
    prof, err := yamlRecord(root,profileRef)
    if err != nil { report.Findings=append(report.Findings,err.Error()); return report,2 }
    raw, err := os.ReadFile(filepath.Join(root,schemaRef))
    if err != nil { report.Findings=append(report.Findings,err.Error()); return report,2 }
    var schema any
    if err=json.Unmarshal(raw,&schema); err!=nil { report.Findings=append(report.Findings,err.Error()); return report,2 }
    valid, err := schema_check.ValidateInstance(schema,reg)
    if err!=nil { report.Findings=append(report.Findings,"OFFICIAL_ENGINE_ERROR:"+err.Error()); return report,2 }
    if !valid { report.Findings=append(report.Findings,"OFFICIAL_ENGINE_SCHEMA_REJECTION"); return report,2 }
    report.Findings=contractErrors(root,reg,prof)
    if len(report.Findings)>0 { return report,2 }
    report.Status="REGISTRY_CHECK_PASS"
    report.RegistryConformance="STRUCTURE_AND_DECLARED_REFERENCES_VALIDATED"
    return report,0
}
