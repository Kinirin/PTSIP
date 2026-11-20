package contracts_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func profileMetadataInput(version string) object {
	input := object{
		"profile_contract_version": version,
		"specification_source":     "https://github.com/Kinirin/PTSIP",
		"specification_revision":   "3c47816770d194ae42f98faedc911d980db0e62a",
		"schema_ref":               "https://raw.githubusercontent.com/Kinirin/PTSIP/main/schemas/ptsip-profile-" + version + ".schema.json",
	}
	if version == "pp.3.00" {
		input["schema_ref"] = "urn:ptsip:storage:contract:profile:official"
	}
	if version == "pp.1.01" {
		input["specification_family"] = "0.3.7-draft"
	}
	return input
}

func projectedProfileMetadata(t *testing.T, input object) object {
	t.Helper()
	response, err := runProfileOperation(t, "projection", "projection.profile.metadata", input)
	if err != nil || response["status"] != "PASS" {
		t.Fatalf("native metadata projection failed: %v, %v", response, err)
	}
	output, ok := response["output"].(object)
	if !ok {
		t.Fatalf("native metadata projection lacks typed output: %v", response)
	}
	if response["normative_authority"] != false || len(output) != 4 {
		t.Fatalf("projection introduced authority or an undeclared output field: %v", response)
	}
	var header object
	if err := yaml.Unmarshal([]byte(output["serialized_header"].(string)), &header); err != nil {
		t.Fatalf("serialized header is not valid YAML: %v", err)
	}
	if !reflect.DeepEqual(header, object{"ptsip": output["metadata"]}) {
		t.Fatalf("serialized YAML does not preserve projected metadata: %v, %v", header, output)
	}
	if output["schema_ref"] != input["schema_ref"] {
		t.Fatalf("projection changed the caller-selected schema: %v", output)
	}
	return output
}

func TestProjectionProfileMetadataNative(t *testing.T) {
	for _, version := range []string{"pp.1.01", "pp.1.02", "pp.2.00", "pp.3.00"} {
		t.Run(version, func(t *testing.T) {
			input := profileMetadataInput(version)
			before := clone(t, input)
			output := projectedProfileMetadata(t, input)
			metadata := output["metadata"].(object)
			if !reflect.DeepEqual(input, before) {
				t.Fatal("read-only projection mutated its input")
			}
			if !reflect.DeepEqual(output["evidence_refs"], array{}) {
				t.Fatalf("projection invented evidence: %v", output)
			}
			if metadata["version"] != version {
				t.Fatalf("projection changed the exact contract version: %v", metadata)
			}
			specification := metadata["specification"].(object)
			if specification["source"] != input["specification_source"] || specification["revision"] != input["specification_revision"] {
				t.Fatalf("projection changed explicit Specification identity: %v", specification)
			}
			if version == "pp.1.01" {
				if len(metadata) != 2 || specification["family"] != input["specification_family"] {
					t.Fatalf("pp.1.01 metadata gained later-version fields: %v", metadata)
				}
			} else {
				if metadata["revision"] != "Rev.0001" || metadata["profile_role"] != "PROJECT" || len(specification) != 2 {
					t.Fatalf("baseline lineage or Specification identity changed: %v", metadata)
				}
			}

			if version != "pp.1.01" {
				input["profile_role"] = "DISTRIBUTED_EXAMPLE"
			}
			output = projectedProfileMetadata(t, input)
			canonicalPath := filepath.Join(productRoot(t), "schemas", "storage", "profile", "history", "project-profile-"+version+".schema.json")
			if version == "pp.3.00" {
				canonicalPath = filepath.Join(productRoot(t), "schemas", "storage", "profile", "project-profile.schema.json")
			}
			canonical := readObject(t, canonicalPath)
			headerSchema := canonical["properties"].(object)["ptsip"]
			validation, err := runProfileOperation(t, "validation", "validation.schema", object{"schema": headerSchema, "instance": output["metadata"]})
			if err != nil || validation["status"] != "PASS" {
				t.Fatalf("independent native header validation failed: %v, %v", validation, err)
			}
			verdict := validation["output"].(object)
			if verdict["valid"] != true || verdict["ptsip_conformance_verdict"] != "NOT_ASSERTED" {
				t.Fatalf("projected header violates its selected field contract or claims authority: %v", validation)
			}
		})
	}

	t.Run("explicit-lineage-and-safe-serialization", func(t *testing.T) {
		input := profileMetadataInput("pp.2.00")
		input["user_revision"] = "Rev.0042"
		input["profile_role"] = "DISTRIBUTED_EXAMPLE"
		input["specification_source"] = "한글 source: \"quoted\"\n---\nextra: [false, null]\\path"
		input["evidence_refs"] = array{"caller-selected-schema-evidence"}
		output := projectedProfileMetadata(t, input)
		metadata := output["metadata"].(object)
		if metadata["revision"] != input["user_revision"] || metadata["profile_role"] != input["profile_role"] || !reflect.DeepEqual(output["evidence_refs"], input["evidence_refs"]) {
			t.Fatalf("projection changed explicit caller values: %v", output)
		}
		if !strings.HasSuffix(output["serialized_header"].(string), "\n") {
			t.Fatal("serialized header lacks its complete final line")
		}
	})

	invalid := []struct {
		name   string
		change func(object)
	}{
		{"unknown-contract", func(input object) { input["profile_contract_version"] = "pp.1.00" }},
		{"noncanonical-contract", func(input object) { input["profile_contract_version"] = "pp.02.00" }},
		{"missing-source", func(input object) { delete(input, "specification_source") }},
		{"empty-source", func(input object) { input["specification_source"] = "" }},
		{"missing-specification-revision", func(input object) { delete(input, "specification_revision") }},
		{"mutable-specification-revision", func(input object) { input["specification_revision"] = "main" }},
		{"empty-selected-schema", func(input object) { input["schema_ref"] = "" }},
		{"missing-selected-schema", func(input object) { delete(input, "schema_ref") }},
		{"null-user-revision", func(input object) { input["user_revision"] = nil }},
		{"noncanonical-user-revision", func(input object) { input["user_revision"] = "rev.0042" }},
		{"user-revision-overflow", func(input object) { input["user_revision"] = "Rev.10000" }},
		{"zero-user-revision", func(input object) { input["user_revision"] = "Rev.0000" }},
		{"undeclared-role", func(input object) { input["profile_role"] = "ADMIN" }},
		{"fabricated-family", func(input object) { input["specification_family"] = "guessed" }},
		{"missing-historical-family", func(input object) { input["profile_contract_version"] = "pp.1.01" }},
		{"later-role-in-historical-contract", func(input object) {
			input["profile_contract_version"] = "pp.1.01"
			input["specification_family"] = "0.3.7-draft"
			input["profile_role"] = "PROJECT"
		}},
		{"unexpected-field", func(input object) { input["authority_grant"] = true }},
		{"invalid-evidence-list", func(input object) { input["evidence_refs"] = "not-an-array" }},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			input := profileMetadataInput("pp.2.00")
			test.change(input)
			response, err := runProfileOperation(t, "projection", "projection.profile.metadata", input)
			if err == nil || response["status"] != "FAIL" || response["output"] != nil {
				t.Fatalf("invalid native metadata input passed: %v, %v", response, err)
			}
		})
	}
}
