package schema_check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSchemaFixture(t *testing.T, root, relative string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:test"}`), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaFilesIncludeApprovedPolicyVersionAndStatus(t *testing.T) {
	root := t.TempDir()
	for _, relative := range approvedSchemaPaths {
		writeSchemaFixture(t, root, relative)
	}
	paths, findings := schemaFiles(root)
	if len(findings) != 0 || len(paths) != len(approvedSchemaPaths) {
		t.Fatalf("approved schemas not discovered: paths=%v findings=%v", paths, findings)
	}
	for _, required := range approvedSchemaPaths {
		present := false
		for _, path := range paths {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				t.Fatal(err)
			}
			if filepath.ToSlash(rel) == required {
				present = true
			}
		}
		if !present {
			t.Fatalf("approved schema omitted: %s", required)
		}
	}
}

func TestMissingPolicyVersionSchemaFailsClosed(t *testing.T) {
	root := t.TempDir()
	for _, relative := range approvedSchemaPaths {
		if strings.Contains(relative, "/Policy_version/") {
			continue
		}
		writeSchemaFixture(t, root, relative)
	}
	_, findings := schemaFiles(root)
	found := false
	for _, finding := range findings {
		if finding.File == ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json" && finding.Problem == "MISSING_APPROVED_SCHEMA" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing policy version schema not reported: %v", findings)
	}
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
	if !found {
		t.Fatalf("missing status schema root not reported: %v", findings)
	}
}

func TestSchemaScanSeparatesJSONDataFromSchemas(t *testing.T) {
	for _, file := range []string{
		".ptsip/contracts/schemas/compile-sealed-rbac/candidate-index.json",
		".ptsip/contracts/schemas/Policy_Rules/root-family/ARCH_rules.schemas.json",
		".ptsip/contracts/schemas/compile-sealed-rbac/binding.schema.json",
	} {
		wantSchema := !strings.HasSuffix(file, "candidate-index.json")
		if isSchemaJSON(file) != wantSchema {
			t.Fatalf("wrong schema classification: %s", file)
		}
	}
}

func TestApprovedRBACScaffoldsRemainFailClosed(t *testing.T) {
	root := routingRepositoryRoot(t)
	for file := range approvedFailClosedRBACScaffolds {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		if !validateFailClosedRBACScaffold(file, doc) {
			t.Errorf("approved RBAC scaffold must remain fail closed: %s", file)
		}
	}
	file := ".ptsip/contracts/schemas/compile-sealed-rbac/activation.schema.json"
	base := map[string]any{
		"$schema":  schemaDraft202012,
		"title":    "Unresolved RBAC scaffold",
		"$comment": "SCAFFOLD_ONLY: Reject all instances",
		"type":     "object",
		"not":      map[string]any{},
	}
	if !validateFailClosedRBACScaffold(file, base) {
		t.Fatal("expected valid closed scaffold")
	}
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"guard_removed", func(v map[string]any) { delete(v, "not") }},
		{"guard_weakened", func(v map[string]any) { v["not"] = map[string]any{"type": "string"} }},
		{"guard_null", func(v map[string]any) { v["not"] = nil }},
		{"wrong_type", func(v map[string]any) { v["type"] = "array" }},
		{"marker_missing", func(v map[string]any) { v["$comment"] = "unresolved" }},
		{"extra_field", func(v map[string]any) { v["properties"] = map[string]any{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := map[string]any{}
			for k, v := range base {
				candidate[k] = v
			}
			tc.mutate(candidate)
			if validateFailClosedRBACScaffold(file, candidate) {
				t.Fatal("unsafe scaffold shape accepted")
			}
		})
	}
	if validateFailClosedRBACScaffold(".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json", base) {
		t.Fatal("unapproved scaffold path accepted")
	}
}

func TestFamilyRulesStorageRejectsUnroutedMisplacedFiles(t *testing.T) {
	root := t.TempDir()
	for _, relative := range approvedSchemaPaths {
		writeSchemaFixture(t, root, relative)
	}
	good := ".ptsip/contracts/policy/Rules/GOV_rules.json"
	writeSchemaFixture(t, root, good)
	_, prior := schemaFiles(root)
	if len(prior) != 0 {
		t.Fatalf("canonical Family rules schema rejected: %v", prior)
	}
	for _, tc := range []struct{ path, diagnosticFile, problem string }{
		{".ptsip/contracts/schemas/Policy_Rules/root-family/GOV_rules.schema.json",
			".ptsip/contracts/schemas/Policy_Rules/root-family/GOV_rules.schema.json",
			"RETIRED_FAMILY_RULES_SCHEMA"},
		{".ptsip/contracts/schemas/Policy_Rules/GOV_rules.schema.json",
			".ptsip/contracts/schemas/Policy_Rules/GOV_rules.schema.json",
			"RETIRED_FAMILY_RULES_SCHEMA"},
		{".ptsip/contracts/schemas/Policy_Rules/abortively/GOV_rules.schema.json",
			".ptsip/contracts/schemas/Policy_Rules/abortively",
			"DISALLOWED_POLICY_RULES_ARCHIVE_DIRECTORY"},
	} {
		writeSchemaFixture(t, root, tc.path)
		_, findings := schemaFiles(root)
		found := false
		for _, row := range findings {
			if row.File == tc.diagnosticFile && row.Problem == tc.problem && row.Status == "FAIL" {
				found = true
			}
		}
		if !found {
			t.Fatalf("misplaced Family schema was not rejected: %s findings=%v", tc.path, findings)
		}
	}
}

func TestPolicyRulesArchiveDirectoryIsForbidden(t *testing.T) {
	root := t.TempDir()
	for _, relative := range approvedSchemaPaths {
		writeSchemaFixture(t, root, relative)
	}
	_, clean := schemaFiles(root)
	if len(clean) != 0 {
		t.Fatalf("canonical fixture unexpectedly rejected: %v", clean)
	}
	forbidden := ".ptsip/contracts/schemas/Policy_Rules/abortively"
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(forbidden)), 0755); err != nil {
		t.Fatal(err)
	}
	_, findings := schemaFiles(root)
	found := false
	for _, f := range findings {
		if f.File == forbidden && f.Problem == "DISALLOWED_POLICY_RULES_ARCHIVE_DIRECTORY" && f.Status == "FAIL" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unnecessary Policy_Rules archive must be rejected: %v", findings)
	}
}
