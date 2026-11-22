package main

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

const unifiedPolicyPath = "CNTR/SFP-CNTR-0009.yaml"

func unifiedPolicyFixture(t *testing.T) (string, []byte, record) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "src", "policy", filepath.FromSlash(unifiedPolicyPath)))
	if err != nil {
		t.Fatal(err)
	}
	document, err := source.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	return root, raw, document
}

func TestUnifiedPolicySourceRoundTrip(t *testing.T) {
	root, raw, document := unifiedPolicyFixture(t)
	restored := dispatch(record{"format": protocol, "operation": "restore-source", "document": document}, "")
	if restored["status"] != "VALID" {
		t.Fatalf("source restoration failed: %v", restored)
	}
	original := source.Object(restored["source"])
	digest, err := source.Digest(original)
	content := source.Object(document["Policy_Content"])
	if err != nil || digest != source.Object(content["source_document"])["sha256"] {
		t.Fatalf("restored source digest does not match recorded digest: %s, %v", digest, err)
	}
	if !strings.Contains(source.String(source.Object(original["Policy_Identity"])["title"]), "고객 기본 계약") {
		t.Fatal("restored source lost the localized policy title")
	}
	project := record{"format": protocol, "operation": "project-support-source", "raw": string(raw), "path": unifiedPolicyPath}
	if report := dispatch(project, ""); report["status"] != "VALID" || !reflect.DeepEqual(report["source"], original) {
		t.Fatalf("projected source differs from restored source: %v", report)
	}
	project["path"] = "CNTR/wrong.yaml"
	if report := dispatch(project, ""); report["status"] != "FAIL" || !strings.Contains(source.String(report["problem"]), "SUPPORT_PROJECTION_PATH_MISMATCH") {
		t.Fatalf("misplaced policy projection was accepted: %v", report)
	}

	report := dispatch(record{"format": protocol, "operation": "validate-unified", "repository_root": root, "document": document, "restore": true}, "")
	if report["status"] != "VALID" || !reflect.DeepEqual(report["source"], original) ||
		report["scope"] != "STRUCTURE_SOURCE_INTEGRITY_AND_DECLARED_REFERENCES" || source.Object(report["evaluation_registration"])["status"] != "UNREGISTERED" {
		t.Fatalf("unexpected unified source validation: %v", report)
	}
	for _, key := range []string{"execution_authorized", "evaluation_authorized", "mutation_authorized", "publication_authorized"} {
		if report[key] != false {
			t.Fatalf("source validation unexpectedly granted %s: %v", key, report)
		}
	}
}

func TestUnifiedPolicySourceRejectsInvalidPartitions(t *testing.T) {
	for _, tc := range []struct{ name, problem string }{
		{"digest", "SOURCE_DIGEST_MISMATCH"},
		{"duplicate", "INVALID_OR_DUPLICATE_SOURCE_ITEM"},
		{"overlap", "DUPLICATE_OR_OVERLAPPING_SOURCE_POINTER"},
		{"invalid escape", "INVALID_SOURCE_POINTER"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, document := unifiedPolicyFixture(t)
			content := source.Object(document["Policy_Content"])
			if tc.name == "digest" {
				source.Object(content["source_document"])["sha256"] = strings.Repeat("0", 64)
			} else {
				items := content["items"].([]any)
				item := maps.Clone(source.Object(items[0]))
				if tc.name != "duplicate" {
					item["id"] = "SRC-9999"
				}
				if tc.name == "invalid escape" {
					item["source_pointer"] = "/~2"
				}
				content["items"] = append(items, item)
			}
			report := dispatch(record{"format": protocol, "operation": "restore-source", "document": document}, "")
			if report["status"] != "FAIL" || !strings.Contains(source.String(report["problem"]), tc.problem) {
				t.Fatalf("want %s; got %v", tc.problem, report)
			}
		})
	}
}

func TestUnifiedPolicySourceRejectsUnregisteredItem(t *testing.T) {
	root, _, document := unifiedPolicyFixture(t)
	source.Object(document["Policy_Evaluation"])["source_refs"] = []any{"SRC-9999"}
	report := dispatch(record{"format": protocol, "operation": "validate-unified", "repository_root": root, "document": document}, "")
	if report["status"] != "FAIL" || !strings.Contains(source.String(report["problem"]), "UNRESOLVED_POLICY_SOURCE_ITEM") || report["execution_authorized"] != false {
		t.Fatalf("unregistered source item not rejected: %v", report)
	}
}

func TestUnifiedPolicySourceRejectsDuplicateYAMLKeys(t *testing.T) {
	report := dispatch(record{"format": protocol, "operation": "project-support-source", "raw": "duplicate: 1\nduplicate: 2\n", "path": unifiedPolicyPath}, "")
	if report["status"] != "FAIL" || !strings.Contains(source.String(report["problem"]), "DUPLICATE_MAPPING_KEY") {
		t.Fatalf("duplicate YAML key not rejected: %v", report)
	}
}
