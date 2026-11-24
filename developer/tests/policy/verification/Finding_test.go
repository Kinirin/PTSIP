package schema_check_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
)

const schemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

var approvedSchemaPaths = []string{
	publicPolicyStatusPath,
	policyVersionPath,
	policyStatusVocabularyPath,
	".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json",
}

func schemaFiles(root string) ([]string, []verification.Finding) {
	inspection := verification.InspectCorpus(root)
	var paths []string
	var problems []verification.Finding
	for _, finding := range inspection.Findings {
		if finding.Status == "FAIL" {
			problems = append(problems, finding)
		} else {
			paths = append(paths, filepath.Join(root, filepath.FromSlash(finding.File)))
		}
	}
	return paths, problems
}

func writeCorpusJSON(t *testing.T, root, relative string, document any) {
	t.Helper()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func corpusFinding(t *testing.T, root, name string) verification.Finding {
	t.Helper()
	for _, finding := range verification.InspectCorpus(root).Findings {
		if finding.File == name {
			return finding
		}
	}
	t.Fatalf("no corpus finding for %s", name)
	return verification.Finding{}
}

func writeSchemaFixture(t *testing.T, root, relative string) {
	t.Helper()
	writeCorpusJSON(t, root, relative, map[string]any{"$schema": schemaDraft202012, "$id": "urn:example:test:" + relative})
}

var otherCurrentSchemaFixtures = []string{
	".ptsip/contracts/schemas/release/scan-fixture.schema.json",
	".ptsip/contracts/compile-sealed-rbac/scan-fixture.schema.json",
}

func writeOtherCurrentSchemaFixtures(t *testing.T, root string) {
	t.Helper()
	for _, relative := range otherCurrentSchemaFixtures {
		writeSchemaFixture(t, root, relative)
	}
}

func TestSchemaFilesIncludeApprovedPolicyVersionAndStatus(t *testing.T) {
	root := t.TempDir()
	writeOtherCurrentSchemaFixtures(t, root)
	for _, relative := range approvedSchemaPaths {
		writeSchemaFixture(t, root, relative)
	}
	paths, findings := schemaFiles(root)
	if len(findings) != 0 || len(paths) != len(approvedSchemaPaths)+len(otherCurrentSchemaFixtures) {
		t.Fatalf("approved schemas not discovered: paths=%v findings=%v", paths, findings)
	}
	for _, required := range append(append([]string{}, approvedSchemaPaths...), otherCurrentSchemaFixtures...) {
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
	writeOtherCurrentSchemaFixtures(t, root)
	for _, relative := range approvedSchemaPaths {
		if relative == policyVersionPath {
			continue
		}
		writeSchemaFixture(t, root, relative)
	}
	_, findings := schemaFiles(root)
	found := false
	for _, finding := range findings {
		if finding.File == ".ptsip/contracts/policy/Policy_lifecycle/version/Policy_version.json" && finding.Problem == "MISSING_APPROVED_SCHEMA" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing policy version schema not reported: %v", findings)
	}
}

func TestMissingCurrentPolicySchemaRootFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeOtherCurrentSchemaFixtures(t, root)
	_, findings := schemaFiles(root)
	found := false
	for _, finding := range findings {
		if finding.File == ".ptsip/contracts/policy" && strings.HasPrefix(finding.Problem, "SCHEMA_ROOT_UNAVAILABLE") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing current policy contract root not reported: %v", findings)
	}
}

func TestSchemaFilesExcludeStoppedContractRoots(t *testing.T) {
	root := t.TempDir()
	writeOtherCurrentSchemaFixtures(t, root)
	for _, relative := range approvedSchemaPaths {
		writeSchemaFixture(t, root, relative)
	}
	for _, relative := range []string{
		".ptsip/contracts/schemas/Policy/PTSIP_Public_Structure.schema.json",
		".ptsip/contracts/schemas/Policy/Policy_Rules/abortively/GOV_rules.schema.json",
		".ptsip/contracts/schemas/compile-sealed-rbac/responsibility.schema.json",
	} {
		writeSchemaFixture(t, root, relative)
	}
	paths, findings := schemaFiles(root)
	if len(findings) != 0 {
		t.Fatalf("historical copies entered current validation: %v", findings)
	}
	if len(paths) != len(approvedSchemaPaths)+len(otherCurrentSchemaFixtures) {
		t.Fatalf("historical copies were discovered: %v", paths)
	}
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		relative = filepath.ToSlash(relative)
		if strings.HasPrefix(relative, ".ptsip/contracts/schemas/Policy/") ||
			strings.HasPrefix(relative, ".ptsip/contracts/schemas/compile-sealed-rbac/") {
			t.Fatalf("stopped contract root entered current scan: %s", relative)
		}
	}
}

func TestSchemaScanSeparatesJSONDataFromSchemas(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{
		".ptsip/contracts/compile-sealed-rbac/candidate-index.json",
		".ptsip/contracts/policy/Policy_Rules/root-family/ARCH_rules.schemas.json",
		".ptsip/contracts/compile-sealed-rbac/binding.schema.json",
	} {
		writeSchemaFixture(t, root, file)
		wantSchema := !strings.HasSuffix(file, "candidate-index.json")
		classifiedAsSchema := corpusFinding(t, root, file).Status != "NON_SCHEMA_JSON_SYNTAX_ONLY"
		if classifiedAsSchema != wantSchema {
			t.Fatalf("wrong schema classification: %s", file)
		}
	}
}

func TestApprovedRBACScaffoldsRemainFailClosed(t *testing.T) {
	root := routingRepositoryRoot(t)
	for _, name := range []string{"responsibility", "condition-atom", "permission-atom", "binding", "sealed-contract", "activation"} {
		file := ".ptsip/contracts/compile-sealed-rbac/" + name + ".schema.json"
		if corpusFinding(t, root, file).Status != "UNRESOLVED_FAIL_CLOSED_SCAFFOLD" {
			t.Errorf("approved RBAC scaffold must remain fail closed: %s", file)
		}
	}
	file := ".ptsip/contracts/compile-sealed-rbac/activation.schema.json"
	base := map[string]any{
		"$schema":  schemaDraft202012,
		"title":    "Unresolved RBAC scaffold",
		"$comment": "SCAFFOLD_ONLY: Reject all instances",
		"type":     "object",
		"not":      map[string]any{},
	}
	fixture := t.TempDir()
	writeCorpusJSON(t, fixture, file, base)
	if corpusFinding(t, fixture, file).Status != "UNRESOLVED_FAIL_CLOSED_SCAFFOLD" {
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
			writeCorpusJSON(t, fixture, file, candidate)
			if corpusFinding(t, fixture, file).Problem != "RBAC_SCAFFOLD_NO_LONGER_FAIL_CLOSED" {
				t.Fatal("unsafe scaffold shape accepted")
			}
		})
	}
	writeCorpusJSON(t, fixture, publicPolicyStatusPath, base)
	if corpusFinding(t, fixture, publicPolicyStatusPath).Status == "UNRESOLVED_FAIL_CLOSED_SCAFFOLD" {
		t.Fatal("unapproved scaffold path accepted")
	}
}

func TestFamilyRulesStorageRejectsUnroutedMisplacedFiles(t *testing.T) {
	root := t.TempDir()
	writeOtherCurrentSchemaFixtures(t, root)
	for _, relative := range approvedSchemaPaths {
		writeSchemaFixture(t, root, relative)
	}
	good := ".ptsip/contracts/policy/Policy_Rules/Family_Selection.json"
	writeSchemaFixture(t, root, good)
	_, prior := schemaFiles(root)
	if len(prior) != 0 {
		t.Fatalf("canonical Family selection rejected: %v", prior)
	}
	for _, tc := range []struct{ path, diagnosticFile, problem string }{
		{".ptsip/contracts/policy/Policy_Rules/root-family/GOV_rules.schema.json",
			".ptsip/contracts/policy/Policy_Rules/root-family/GOV_rules.schema.json",
			"RETIRED_FAMILY_RULES_SCHEMA"},
		{".ptsip/contracts/policy/Policy_Rules/GOV_rules.schema.json",
			".ptsip/contracts/policy/Policy_Rules/GOV_rules.schema.json",
			"RETIRED_FAMILY_RULES_SCHEMA"},
		{".ptsip/contracts/policy/Policy_Rules/abortively/GOV_rules.schema.json",
			".ptsip/contracts/policy/Policy_Rules/abortively",
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
	writeOtherCurrentSchemaFixtures(t, root)
	for _, relative := range approvedSchemaPaths {
		writeSchemaFixture(t, root, relative)
	}
	_, clean := schemaFiles(root)
	if len(clean) != 0 {
		t.Fatalf("canonical fixture unexpectedly rejected: %v", clean)
	}
	forbidden := ".ptsip/contracts/policy/Policy_Rules/abortively"
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
