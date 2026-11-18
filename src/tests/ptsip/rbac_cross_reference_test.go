package contracts_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const (
	raID        = "R-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	racID       = "ID-RAC-aaaaaaaaaaaaaaaa"
	secondRACID = "ID-RAC-bbbbbbbbbbbbbbbb"
	aecID       = "ID-AEC-EEEEEEEEEEEEEEEE"
	fileID      = "profiles-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func rbacDocuments() object {
	return object{
		"policy": object{"Policy_Rules": object{"rules": array{object{"RA_id": raID}}}},
		"ra": object{"RA_list": object{"entries": array{object{
			"RA_id": raID, "RA_title": "Review a request", "RA_description": "Review assignment and exercise conditions.",
			"AEC_ids": array{object{"Exercise review": aecID}}, "AEC_name": array{"Exercise review"},
			"RAC_ids":  array{object{"Primary assignment": racID}, object{"Secondary assignment": secondRACID}},
			"RAC_name": array{"Primary assignment", "Secondary assignment"},
		}}}},
		"rac": object{
			"file_name": "RAC-list.ptsip.yaml", "file_id": fileID, "format_version": "v.1", "file_title": "Assignment conditions",
			"RAC_list": array{
				object{"RAC_id": racID, "RAC_name": "Primary assignment", "RAC_description": "Initial reviewer is assigned.", "value": object{"example": array{"primary", 1}}},
				object{"RAC_id": secondRACID, "RAC_name": "Secondary assignment", "RAC_description": "Backup reviewer is assigned."},
			},
		},
		"aec": object{"AEC_list": array{object{
			"AEC_id": aecID, "AEC_name": "Exercise review", "AEC_description": "Assigned reviewer acts on a request.",
			"value": array{"example", object{"threshold": nil}},
		}}},
	}
}

func raRecord(documents object) object {
	return documents["ra"].(object)["RA_list"].(object)["entries"].(array)[0].(object)
}

func TestForwardReferencesUseCanonicalOwningListsAndReadOnlyUsage(t *testing.T) {
	v := validator(t)
	documents := rbacDocuments()
	before, err := json.Marshal(documents)
	if err != nil {
		t.Fatal(err)
	}
	result := evaluate(t, v, "validation.rbac.references", documents)
	assertValid(t, result)
	after, err := json.Marshal(documents)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("reference validation mutated the caller's documents")
	}
	data, err := json.Marshal(result.RAUsage)
	if err != nil {
		t.Fatal(err)
	}
	var usage []object
	if err := json.Unmarshal(data, &usage); err != nil {
		t.Fatal(err)
	}
	if len(usage) != 1 || usage[0]["RA_id"] != raID {
		t.Fatalf("unexpected derived RA usage: %s", data)
	}
	want := array{array{"Policy_Rules", "rules", float64(0), "RA_id"}}
	if !reflect.DeepEqual(usage[0]["paths"], want) {
		t.Fatalf("incorrect policy usage projection: %s", data)
	}
	if _, present := raRecord(documents)["policy_usage"]; present {
		t.Fatal("reverse policy usage was written to the RA record")
	}
}

func TestCanonicalNamesMayRepeatAcrossDistinctConditionIDs(t *testing.T) {
	v := validator(t)
	documents := rbacDocuments()
	documents["rac"].(object)["RAC_list"].(array)[1].(object)["RAC_name"] = "Primary assignment"
	raRecord(documents)["RAC_ids"].(array)[1] = object{"Primary assignment": secondRACID}
	raRecord(documents)["RAC_name"].(array)[1] = "Primary assignment"
	assertValid(t, evaluate(t, v, "validation.rbac.references", documents))
}

func TestEmptyRelationshipGroupsAndInlineConditionValuesAreAllowed(t *testing.T) {
	v := validator(t)
	assertValid(t, evaluate(t, v, "validation.rbac.references", object{
		"policy": object{}, "ra": object{"RA_list": object{"entries": array{object{"RA_id": raID}}}},
		"rac": object{"RAC_list": array{}}, "aec": object{"AEC_list": array{}},
	}))
	for _, value := range []any{nil, false, 2, "author value", array{}, object{}, object{"custom": 1}} {
		for _, kind := range []string{"RAC", "AEC"} {
			id := racID
			if kind == "AEC" {
				id = aecID
			}
			result := evaluate(t, v, "validation.rbac."+strings.ToLower(kind), object{
				"document": object{kind + "_list": array{object{kind + "_id": id, "value": value}}},
			})
			assertValid(t, result)
		}
	}
}

func TestRBACListMetadataAndSevenFieldShapes(t *testing.T) {
	v := validator(t)
	racCases := []object{
		{"file_id": racID}, {"file_id": strings.ToUpper(fileID)}, {"file_id": fileID + "\n"},
		{"File_Identity": object{}}, {"file_kind": "RAC"}, {"RAC_list": object{"entries": array{}}},
		{"RAC_list": array{object{"RAC_id": aecID}}},
		{"RAC_list": array{object{"RAC_id": racID, "source_ref": "external.json"}}},
	}
	for index, payload := range racCases {
		result := evaluate(t, v, "validation.rbac.rac", object{"document": payload})
		if result.Valid || !hasCode(result.Diagnostics, "SCHEMA") {
			t.Errorf("RAC case %d accepted unapproved metadata/shape: %+v", index, result)
		}
	}
	raCases := []struct {
		field string
		value any
	}{
		{"RA_id", strings.ToLower(raID)}, {"policy_usage", array{}}, {"RA_purpose", "An eighth field"},
		{"RAC_ids", array{racID}}, {"RAC_ids", array{object{"Primary assignment": aecID}}},
		{"RAC_ids", array{object{"Primary assignment": racID, "Secondary assignment": secondRACID}}},
		{"AEC_name", array{object{"Exercise review": aecID}}},
	}
	for _, test := range raCases {
		documents := rbacDocuments()
		raRecord(documents)[test.field] = test.value
		result := evaluate(t, v, "validation.rbac.ra", object{"document": documents["ra"]})
		if result.Valid || !hasCode(result.Diagnostics, "SCHEMA") {
			t.Errorf("RA %s accepted invalid shape: %+v", test.field, result)
		}
	}
}

func TestReferencesRequireExactUniqueIDsInCorrectLists(t *testing.T) {
	v := validator(t)
	for _, kind := range []string{"ra", "rac", "aec"} {
		documents := rbacDocuments()
		if kind == "ra" {
			documents[kind] = object{"RA_list": object{"entries": array{}}}
		} else {
			documents[kind] = object{strings.ToUpper(kind) + "_list": array{}}
		}
		result := evaluate(t, v, "validation.rbac.references", documents)
		if result.Valid || !hasCode(result.Diagnostics, "UNKNOWN_ID") {
			t.Errorf("missing owning %s target was accepted: %+v", kind, result)
		}
	}
	documents := rbacDocuments()
	raRecord(documents)["RAC_ids"].(array)[0] = object{"Primary assignment": "ID-RAC-Aaaaaaaaaaaaaaaa"}
	result := evaluate(t, v, "validation.rbac.references", documents)
	if result.Valid || !hasCode(result.Diagnostics, "UNKNOWN_ID") {
		t.Fatalf("case-folded ID was accepted: %+v", result)
	}
	for _, kind := range []string{"RA", "RAC", "AEC"} {
		documents := rbacDocuments()
		if kind == "RA" {
			list := documents["ra"].(object)["RA_list"].(object)
			list["entries"] = append(list["entries"].(array), object{"RA_id": raID, "RA_title": "Different record"})
		} else {
			list := documents[strings.ToLower(kind)].(object)
			entries := list[kind+"_list"].(array)
			copy := clone(t, entries[0].(object))
			copy[kind+"_name"] = "Different canonical name"
			list[kind+"_list"] = append(entries, copy)
		}
		result := evaluate(t, v, "validation.rbac.references", documents)
		if result.Valid || !hasCode(result.Diagnostics, "DUPLICATE_ID") {
			t.Errorf("duplicate %s record was accepted: %+v", kind, result)
		}
	}
}

func TestRelationshipMultiplicityAndDisplayPairing(t *testing.T) {
	v := validator(t)
	documents := rbacDocuments()
	record := raRecord(documents)
	record["RAC_ids"] = append(record["RAC_ids"].(array), object{"Primary assignment": racID})
	record["RAC_name"] = append(record["RAC_name"].(array), "Primary assignment")
	result := evaluate(t, v, "validation.rbac.references", documents)
	if result.Valid || !hasCode(result.Diagnostics, "DUPLICATE_ID") {
		t.Fatalf("duplicate relationship ID was accepted: %+v", result)
	}
	for _, names := range []array{{}, {"Primary assignment"}, {"Secondary assignment", "Primary assignment"}, {"Primary assignment", "Unregistered name"}} {
		documents := rbacDocuments()
		raRecord(documents)["RAC_name"] = names
		result := evaluate(t, v, "validation.rbac.references", documents)
		if result.Valid || !hasCode(result.Diagnostics, "NAME_PAIRING") {
			t.Errorf("display sequence does not match reference sequence: %+v", result)
		}
	}
	documents = rbacDocuments()
	delete(raRecord(documents), "AEC_ids")
	result = evaluate(t, v, "validation.rbac.references", documents)
	if result.Valid || !hasCode(result.Diagnostics, "NAME_PAIRING") {
		t.Fatalf("display names without relationship IDs passed: %+v", result)
	}
	documents = rbacDocuments()
	delete(raRecord(documents), "AEC_name")
	assertValid(t, evaluate(t, v, "validation.rbac.references", documents))
}

func TestCanonicalConditionNameCannotBeSuppliedByRA(t *testing.T) {
	v := validator(t)
	documents := rbacDocuments()
	documents["rac"].(object)["RAC_list"].(array)[0].(object)["RAC_name"] = "Renamed assignment"
	result := evaluate(t, v, "validation.rbac.references", documents)
	if result.Valid || !hasCode(result.Diagnostics, "NAME_MISMATCH") {
		t.Fatalf("stale derived condition name was accepted: %+v", result)
	}
	raRecord(documents)["RAC_ids"].(array)[0] = object{"Renamed assignment": racID}
	raRecord(documents)["RAC_name"].(array)[0] = "Renamed assignment"
	assertValid(t, evaluate(t, v, "validation.rbac.references", documents))
	delete(documents["aec"].(object)["AEC_list"].(array)[0].(object), "AEC_name")
	result = evaluate(t, v, "validation.rbac.references", documents)
	if result.Valid || !hasCode(result.Diagnostics, "NAME_MISMATCH") {
		t.Fatalf("RA supplied a missing authoritative name: %+v", result)
	}
}
