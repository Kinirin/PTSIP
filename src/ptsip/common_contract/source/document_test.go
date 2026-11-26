package source

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

func TestInstalledPlanPolicyReferenceOwnership(t *testing.T) {
	product := t.TempDir()
	policies := filepath.Join(product, "support", "policy")
	if err := os.MkdirAll(policies, 0o755); err != nil {
		t.Fatal(err)
	}
	const index = `schema_version: ptsip-support-feature-policy-index/v1
policy_class: PTSIP_SUPPORT_FEATURE
policies:
  - id: first
    path: first.yaml
  - id: second
    path: second.yaml
  - id: duplicate-entry
    path: first.yaml
  - id: duplicate-entry
    path: second.yaml
  - id: wrong-owner
    path: first.yaml
`
	for name, contents := range map[string]string{
		"index.yaml":  index,
		"first.yaml":  `{"Policy_Identity":{"id":"first"},"Policy_Rules":{"rules":[{"id":"standalone"},{"id":"duplicate"},{"id":"duplicate"}]}}`,
		"second.yaml": `{"Policy_Identity":{"id":"second"},"Policy_Rules":{"contracts":[{"rules":[{"id":"federated"}]}]}}`,
	} {
		if err := os.WriteFile(filepath.Join(policies, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	check := func(reference string) *PlanPolicyReferenceError {
		return CheckPlanPolicyReferences(product, false, Document{"policy_refs": []any{reference}})
	}
	for _, reference := range []string{"first#standalone", "second#federated"} {
		if err := check(reference); err != nil {
			t.Errorf("exact owner binding rejected: %v", err)
		}
	}
	for reference, problem := range map[string]string{
		"first#duplicate":           "found 2",
		"duplicate-entry#federated": "found 2",
		"wrong-owner#standalone":    "does not declare policy",
		"second#standalone":         "found 0",
	} {
		if err := check(reference); err == nil || !strings.Contains(err.Error(), problem) {
			t.Errorf("expected %q for %q, got %v", problem, reference, err)
		}
	}
}

func TestCanonicalSourceGolden(t *testing.T) {
	raw, err := os.ReadFile("../testdata/canonical_source.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jsondoc.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	fixture := Object(decoded)
	expected := String(fixture["sha256"])
	for _, original := range []any{fixture["original"], yamlOriginal(t)} {
		got, err := Digest(original)
		if err != nil || got != expected {
			t.Fatalf("source digest = %q, %v; want %q", got, err, expected)
		}
		document := Document{
			"schema_version": Version,
			"Policy_Content": Document{
				"source_document": Document{"sha256": expected},
				"items":           []any{Document{"id": "SRC-0001", "source_pointer": "", "value": original}},
			},
		}
		restored, err := Restore(document)
		if err != nil || !reflect.DeepEqual(restored, original) {
			t.Fatalf("restored source = %#v, %v", restored, err)
		}
	}
}

func yamlOriginal(t *testing.T) Document {
	t.Helper()
	doc, err := Decode([]byte(`unicode: "\uC815\uCC45 \U0001F600"
decimal: 1.0
negative_zero: -0.0
small: 1.0e-6
large_int: 9007199254740993
nested: {"a/b": {"~": [null, true, 1]}}
`))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}
