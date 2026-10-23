package schema_check

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func writeSchemaFixture(t *testing.T, root, relative string) {
    t.Helper()
    path := filepath.Join(root, filepath.FromSlash(relative))
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { t.Fatal(err) }
    if err := os.WriteFile(path, []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:test"}`), 0644); err != nil {
        t.Fatal(err)
    }
}

func TestSchemaFilesIncludeApprovedPolicyVersionAndStatus(t *testing.T) {
    root := t.TempDir()
    for _, relative := range approvedSchemaPaths { writeSchemaFixture(t, root, relative) }
    paths, findings := schemaFiles(root)
    if len(findings) != 0 || len(paths) != len(approvedSchemaPaths) {
        t.Fatalf("approved schemas not discovered: paths=%v findings=%v", paths, findings)
    }
    for _, required := range approvedSchemaPaths {
        present := false
        for _, path := range paths {
            rel, err := filepath.Rel(root, path)
            if err != nil { t.Fatal(err) }
            if filepath.ToSlash(rel) == required { present = true }
        }
        if !present { t.Fatalf("approved schema omitted: %s", required) }
    }
}

func TestMissingPolicyVersionSchemaFailsClosed(t *testing.T) {
    root := t.TempDir()
    for _, relative := range approvedSchemaPaths {
        if strings.Contains(relative, "/Policy_version/") { continue }
        writeSchemaFixture(t, root, relative)
    }
    _, findings := schemaFiles(root)
    found := false
    for _, finding := range findings {
        if finding.File == ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json" && finding.Problem == "MISSING_APPROVED_SCHEMA" {
            found = true
        }
    }
    if !found { t.Fatalf("missing policy version schema not reported: %v", findings) }
}

func TestEmptyStatusSchemaRootFailsClosed(t *testing.T) {
    root := t.TempDir()
    writeSchemaFixture(t, root, ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json")
    _, findings := schemaFiles(root)
    found := false
    for _, finding := range findings {
        if finding.File == ".ptsip/contracts/schemas" && strings.HasPrefix(finding.Problem, "SCHEMA_ROOT_UNAVAILABLE") {
            found = true
        }
    }
    if !found { t.Fatalf("missing status schema root not reported: %v", findings) }
}
