package schema_check

// Checks only this repository's owner-selected contract storage route.
// It does not define a universal PTSIP contract directory requirement.
import (
    "encoding/json"
    "errors"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func checkCanonicalContractFile(root, name string) error {
    if !strings.HasPrefix(name, ".ptsip/contracts/") ||
        strings.Contains(name, "\\") ||
        strings.Contains(name, "..") ||
        filepath.ToSlash(filepath.Clean(name)) != name {
        return fmt.Errorf("unapproved or unsafe contract reference %q", name)
    }
    base, err := filepath.EvalSymlinks(filepath.Join(root, ".ptsip", "contracts"))
    if err != nil { return fmt.Errorf("canonical contract root: %w", err) }
    target, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(name)))
    if err != nil { return fmt.Errorf("%s: %w", name, err) }
    relative, err := filepath.Rel(base, target)
    if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
        return fmt.Errorf("contract reference escapes owner-selected root: %s", name)
    }
    info, err := os.Stat(target)
    if err != nil { return fmt.Errorf("%s: %w", name, err) }
    if !info.Mode().IsRegular() { return fmt.Errorf("%s: not a regular contract file", name) }
    return nil
}

func TestCanonicalDeveloperContractRoutes(t *testing.T) {
    root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
    if err != nil { t.Fatal(err) }
    old := filepath.Join(root, "developer", "contracts")
    if _, err := os.Stat(old); !errors.Is(err, os.ErrNotExist) {
        t.Fatalf("legacy contract root remains active or inaccessible: %v", err)
    }

    canonical := filepath.Join(root, ".ptsip", "contracts")
    count := 0
    err = filepath.WalkDir(canonical, func(path string, entry fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" { return nil }
        bytes, err := os.ReadFile(path)
        if err != nil { return err }
        if !json.Valid(bytes) { return fmt.Errorf("invalid contract JSON: %s", path) }
        count++
        return nil
    })
    if err != nil { t.Fatal(err) }
    if count < 57 {
        t.Fatalf("contract migration coverage incomplete: %d JSON files (expected at least 55 migrated plus 2 release)", count)
    }

    routingPath := ".ptsip/contracts/policy/routing/policy_routes.json"
    if err := checkCanonicalContractFile(root, routingPath); err != nil { t.Fatal(err) }
    raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(routingPath)))
    if err != nil { t.Fatal(err) }
    var routing struct {
        IdentitySources map[string]string `json:"identity_sources"`
        Routes map[string]map[string]struct {
            SchemaRefs map[string]string `json:"schema_refs"`
        } `json:"routes"`
    }
    if err := json.Unmarshal(raw, &routing); err != nil { t.Fatal(err) }
    if len(routing.Routes["PTSIP_DEVELOPER_POLICY"]) != 14 {
        t.Fatalf("expected 14 owner-selected root families, got %d", len(routing.Routes["PTSIP_DEVELOPER_POLICY"]))
    }

    for _, source := range []string{"policy_kind", "root_family"} {
        if err := checkCanonicalContractFile(root, routing.IdentitySources[source]); err != nil { t.Fatal(err) }
    }
    rootFamilyPath := routing.IdentitySources["root_family"]
    vocabBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rootFamilyPath)))
    if err != nil { t.Fatal(err) }
    var vocab struct { Enum []string `json:"enum"` }
    if err := json.Unmarshal(vocabBytes, &vocab); err != nil { t.Fatal(err) }
    if len(vocab.Enum) != 14 { t.Fatalf("expected 14 root-family vocabulary values, got %d", len(vocab.Enum)) }
    for _, family := range vocab.Enum {
        entry, ok := routing.Routes["PTSIP_DEVELOPER_POLICY"][family]
        if !ok { t.Fatalf("missing declared family route: %s", family) }
        for _, key := range []string{"public", "authority", "rules"} {
            if err := checkCanonicalContractFile(root, entry.SchemaRefs[key]); err != nil {
                t.Fatalf("%s/%s: %v", family, key, err)
            }
        }
    }
}

func TestCanonicalContractReferenceRejectsUnsafeOrMissing(t *testing.T) {
    root := t.TempDir()
    for _, ref := range []string{
        "developer/contracts/identity/root_family.json",
        ".ptsip/contracts/../outside.json",
        ".ptsip/contracts/missing.json",
        "/absolute/contract.json",
    } {
        if checkCanonicalContractFile(root, ref) == nil {
            t.Fatalf("must reject missing or noncanonical reference: %s", ref)
        }
    }
}
func TestPolicyContractResponsibilityMigration(t *testing.T) {
    root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
    if err != nil { t.Fatal(err) }
    required := []string{
        ".ptsip/contracts/policy/boundary.json",
        ".ptsip/contracts/policy/identity/policy_kind.json",
        ".ptsip/contracts/policy/identity/root_family.json",
        ".ptsip/contracts/policy/routing/policy_routes.json",
        ".ptsip/contracts/policy/routing/Policy_Routing.schema.json",
        ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json",
        ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json",
        ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json",
        ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json",
        ".ptsip/contracts/policy/authority/Authority_Major_Category.json",
        ".ptsip/contracts/policy/authority/Authority_Minor_Category.json",
        ".ptsip/contracts/policy/authority/Authority_Micro_Category.json",
        ".ptsip/contracts/policy/authority/abortively/v1_Authority_Major_Category.json",
        ".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_ACTIVATE.json",
        ".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_DRAFT.json",
        ".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_INTERRUPTION.json",
    }
    for _, name := range required {
        if err := checkCanonicalContractFile(root, name); err != nil { t.Fatalf("%s: %v", name, err) }
    }
    obsolete := []string{
        ".ptsip/contracts/identity/policy_kind.json",
        ".ptsip/contracts/identity/root_family.json",
        ".ptsip/contracts/routing/policy_routes.json",
        ".ptsip/contracts/schemas/Policy_Identity/Routing/Policy_Routing.schema.json",
        ".ptsip/contracts/lifecycle/Policy_status/Policy_status_APPROVED.json",
        ".ptsip/contracts/lifecycle/Policy_version/Policy_version.schema.json",
        ".ptsip/contracts/lifecycle/policy_status.json",
        ".ptsip/contracts/lifecycle/Plan_status/Plan_status_ACTIVATE.json",
        ".ptsip/contracts/lifecycle/Plan_status/Plan_status_DRAFT.json",
        ".ptsip/contracts/lifecycle/Plan_status/Plan_status_INTERRUPTION.json",
        ".ptsip/contracts/Authority/Authority_Major_Category.json",
    }
    for _, name := range obsolete {
        if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !errors.Is(err, os.ErrNotExist) {
            t.Fatalf("obsolete policy contract path remains or is inaccessible: %s: %v", name, err)
        }
    }
}

func TestPlanStatusContractVocabulary(t *testing.T) {
    root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
    if err != nil { t.Fatal(err) }
    for _, tc := range []struct {
        filename string
        status string
        id string
    }{
        {"Plan_status_ACTIVATE.json", "ACTIVATE", "urn:ptsip:contract:constant:plan_status_activate:v1"},
        {"Plan_status_DRAFT.json", "DRAFT", "urn:ptsip:contract:constant:plan_status_draft:v1"},
        {"Plan_status_INTERRUPTION.json", "INTERRUPTION", "urn:ptsip:contract:constant:plan_status_interruption:v1"},
    } {
        relative := ".ptsip/contracts/policy/lifecycle/Plan_status/" + tc.filename
        raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
        if err != nil { t.Fatal(err) }
        var schema struct {
            ID string `json:"$id"`
            Required []string `json:"required"`
            Properties struct {
                Status struct { Const string `json:"const"` } `json:"status"`
            } `json:"properties"`
            AdditionalProperties bool `json:"additionalProperties"`
        }
        if err := json.Unmarshal(raw, &schema); err != nil { t.Fatal(err) }
        if schema.ID != tc.id || len(schema.Required) != 1 || schema.Required[0] != "status" ||
            schema.Properties.Status.Const != tc.status || schema.AdditionalProperties {
            t.Fatalf("%s: Plan status semantics changed during relocation", relative)
        }
    }
}
