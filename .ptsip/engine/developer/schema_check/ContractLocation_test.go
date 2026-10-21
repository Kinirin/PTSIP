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
    info, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
    if err != nil { return fmt.Errorf("%s: %w", name, err) }
    if info.IsDir() { return fmt.Errorf("%s: not a file", name) }
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

    routingPath := ".ptsip/contracts/routing/policy_routes.json"
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
