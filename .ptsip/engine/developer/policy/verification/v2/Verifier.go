// Package v2 validates the registered policy v2 structure with the official
// PTSIP VALIDATION engine. It grants no policy, lifecycle or execution authority.
package v2

import (
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "sort"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
    "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractcheck"
)

const registryPath = ".ptsip/contracts/policy/candidate.index.json"
const publicID = "urn:ptsip:candidate:contract:policy:public-structure:v2"
const rulesID = "urn:ptsip:candidate:contract:policy:rules:v2"

func object(v any) map[string]any { m, _ := v.(map[string]any); return m }
func stringValue(v any) string { s, _ := v.(string); return s }

// Check checks structural conformance and registered schema resource identities.
// Unresolved fields remain unresolved; a successful result is not approval.
func Check(root string, document authoring.Document, validate authoring.Validator) error {
    registryFile, err := authoring.LocalFile(root, registryPath)
    if err != nil { return err }
    raw, err := os.ReadFile(registryFile)
    if err != nil { return err }
    var registry map[string]any
    if err := json.Unmarshal(raw, &registry); err != nil { return fmt.Errorf("INVALID_V2_REGISTRY: %w", err) }
    if registry["active_registration"] != true || registry["active_policy_cutover"] != "STRUCTURE_V2_ACTIVE" {
        return fmt.Errorf("V2_POLICY_STRUCTURE_NOT_ACTIVE")
    }
    refs := object(registry["schema_refs"])
    if stringValue(refs["public"]) != publicID || stringValue(refs["rules"]) != rulesID {
        return fmt.Errorf("V2_SCHEMA_REGISTRATION_MISMATCH")
    }

    artifacts, ok := registry["artifacts"].([]any)
    if !ok { return fmt.Errorf("V2_SCHEMA_ARTIFACTS_MISSING") }
    sources := make([]contractcheck.Source, 0, len(artifacts))
    known := map[string]bool{}
    for _, item := range artifacts {
        entry := object(item)
        if stringValue(entry["kind"]) != "SCHEMA" { continue }
        id, p := stringValue(entry["schema_id"]), stringValue(entry["path"])
        if id == "" || p == "" || known[id] { return fmt.Errorf("DUPLICATE_OR_MISSING_V2_SCHEMA: %s", id) }
        if len(p) < len(".ptsip/contracts/policy/") || p[:len(".ptsip/contracts/policy/")] != ".ptsip/contracts/policy/" {
            return fmt.Errorf("V2_SCHEMA_OUTSIDE_POLICY_CONTRACT_ROOT: %s", p)
        }
        known[id] = true
        sources = append(sources, contractcheck.Source{Path:p, ID:id})
    }
    external := object(registry["external_resources"])
    if len(external) == 0 { return fmt.Errorf("V2_EXTERNAL_RESOURCES_MISSING") }
    names := make([]string, 0, len(external))
    for id := range external { names = append(names, id) }
    sort.Strings(names)
    for _, id := range names {
        entry := object(external[id])
        p, expected := stringValue(entry["path"]), stringValue(entry["sha256"])
        if known[id] || p == "" || len(expected) != 64 { return fmt.Errorf("INVALID_V2_EXTERNAL_RESOURCE: %s", id) }
        name, err := authoring.LocalFile(root, p)
        if err != nil { return err }
        raw, err := os.ReadFile(name)
        if err != nil { return err }
        // Git can check out the same committed JSON with CRLF on Windows.
        // Pin the platform-independent LF representation; do not alter or
        // relax schema validation of the original on-disk document.
        digest := sha256.Sum256(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n")))
        if hex.EncodeToString(digest[:]) != expected {
            return fmt.Errorf("V2_EXTERNAL_RESOURCE_DIGEST_MISMATCH: %s", id)
        }
        known[id] = true
        sources = append(sources, contractcheck.Source{Path:p, ID:id})
    }
    if !known[publicID] || !known[rulesID] { return fmt.Errorf("V2_PUBLIC_RULES_SCHEMAS_MISSING") }
    resources, err := contractcheck.Load(root, sources...)
    if err != nil { return err }
    var public map[string]any
    for _, item := range resources {
        doc, _ := item.(map[string]any)
        if doc["$id"] == publicID { public = doc; break }
    }
    if public == nil { return fmt.Errorf("V2_PUBLIC_SCHEMA_UNRESOLVED") }
    if err := contractcheck.Validate(validate, public, document, resources); err != nil { return err }
    return checkIndexedReferences(document)
}
