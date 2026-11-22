package contracts_test

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/evidence"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestGoEvidenceNormalizationPreservesContract(t *testing.T) {
	snapshot := evidence.SnapshotBinding{
		RepositoryRoot: "C:/기록", StatusFingerprint: "status", TrackedContentFingerprint: "content",
	}
	generation := evidence.SourceGenerationBinding{
		ProfilePath: "profile.yaml", Version: "pp.3.00", SpecificationRevision: "rev", ContentSHA256: "digest",
	}
	context, err := evidence.BuildEvaluationContext(snapshot, &generation)
	if err != nil || context.EvaluationID != "evaluation:6ef528fe8845b8b1aee3be5c" {
		t.Fatalf("evaluation identity changed: %#v, %v", context, err)
	}
	profilePath, line, reason, channel := "프로필.yaml", 3, "not available", "repo"
	qualifiers := map[string]json.RawMessage{"kind": json.RawMessage(`"검사"`)}
	observed := evidence.Fact{Subject: "경로", Predicate: "dependency", Qualifiers: qualifiers,
		Value:  json.RawMessage(`{"文字":["<한글>","\u2028",1.0,1e-5,1e16,-0.0]}`),
		Origin: evidence.Origin{Provenance: "OBSERVED", Adapter: "probe", EvidenceID: "fact1", BasisIDs: []string{"β", "a", "β"}}}
	declared := evidence.Fact{Subject: "경로", Predicate: "dependency", Qualifiers: qualifiers, Value: json.RawMessage("false"),
		Origin: evidence.Origin{Provenance: "DECLARED", Adapter: "manifest", EvidenceID: "fact2", SourcePath: &profilePath, Line: &line}}
	inferred := evidence.Fact{Subject: "component", Predicate: "class",
		Origin: evidence.Origin{Provenance: "INFERRED", Adapter: "probe", EvidenceID: "fact3"}}
	facts := []evidence.Fact{declared, observed, observed, inferred}
	channels := []evidence.Channel{
		{ID: "repo", Status: "PRODUCED", EvidenceCount: 3},
		{ID: "external", Status: "NOT_ANALYZED", Reason: &reason},
	}
	issues := []evidence.Issue{{Code: "B", Message: "계약", Channel: &channel, EvidenceIDs: []string{"β"}}, {Code: "A", Message: "확인"}}
	result, err := evidence.Normalize(facts, context, channels, issues)
	if err != nil {
		t.Fatal(err)
	}
	// IDs and digests captured from the existing v1 implementation.
	if result.DeterministicDigest != "9a002a69927723a747a341f460cc946482acf478f56b3507dfaf85c1f868f9a4" ||
		result.RecordCount != 2 || result.ConflictCount != 1 {
		t.Fatalf("normalized content differs: %#v", result)
	}
	first, second := result.Records[0], result.Records[1]
	if first.SemanticID != "evidence:d9423f3cc1032f5642a1b7ef" || first.Status != "CONSISTENT" ||
		first.Assertions[0].ID != "assertion:ca7d18090af5bd85a93a1bd3" ||
		second.SemanticID != "evidence:ef7a03243ac92b0e3717b0c3" || second.Status != "CONFLICT" || second.Authority != "EVIDENCE_ONLY" ||
		second.Assertions[0].ID != "assertion:a780925a5d00d0b0b1f16502" || second.Assertions[1].ID != "assertion:948f8fd3ba7eebc050920a04" ||
		len(second.Assertions[1].Origins) != 1 || !slices.Equal(second.Assertions[1].Origins[0].BasisIDs, []string{"a", "β"}) {
		t.Fatalf("records differ: %#v", result.Records)
	}
	slices.Reverse(facts)
	slices.Reverse(channels)
	slices.Reverse(issues)
	reordered, err := evidence.Normalize(facts, context, channels, issues)
	if err != nil || !reflect.DeepEqual(reordered, result) {
		t.Fatalf("reordering inputs changed the result: %#v, %v", reordered, err)
	}

	rawSchema, err := os.ReadFile("../../ptsip/specdata/ptsip-normalized-evidence.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsondoc.Decode(rawSchema)
	if err != nil {
		t.Fatal(err)
	}
	id := document.(map[string]any)["$id"].(string)
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(id, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(value); err != nil {
		t.Fatalf("Go output violates existing evidence format: %v", err)
	}

	emptyContext, err := evidence.BuildEvaluationContext(snapshot, nil)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := evidence.Normalize(nil, emptyContext, nil, nil)
	if err != nil || empty.DeterministicDigest != "94b67861ee5d1220523a0818c294216615531e8ce7abbc54cf5e43300b0ff9e3" {
		t.Fatalf("empty output changed: %#v, %v", empty, err)
	}
	observed.Value = json.RawMessage(`{"duplicate":1,"duplicate":2}`)
	if _, err := evidence.Normalize([]evidence.Fact{observed}, context, nil, nil); err == nil || !strings.Contains(err.Error(), "DUPLICATE_MAPPING_KEY") {
		t.Fatalf("ambiguous evidence value accepted: %v", err)
	}
}
