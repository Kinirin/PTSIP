package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/compatibility"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/registry"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
)

const protocol = "ptsip-common-contract/v1"

// The wheel builder binds these to the exact source and manifest it packages.
var sourceRevision, registryDigest string

type record = map[string]any

func runCLI() {
	root := flag.String("resource-root", "", "common_contract directory for source builds")
	flag.Parse()
	if flag.NArg() != 0 || (*root != "" && registryDigest != "") {
		fmt.Fprintln(os.Stderr, "invalid arguments for this build")
		os.Exit(2)
	}
	const maxRequest = 32 * 1024 * 1024
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, maxRequest+1))
	result := record{"status": "INVALID", "problem": "REQUEST_SIZE_OR_READ_FAILURE"}
	if err == nil && len(raw) <= maxRequest {
		value, err := jsondoc.Decode(raw)
		if err != nil {
			result["problem"] = err.Error()
		} else if request := source.Object(value); request != nil && request["format"] == protocol {
			result = dispatch(request, *root)
		} else {
			result["problem"] = "INVALID_REQUEST_FORMAT"
		}
	}
	result["format"] = protocol
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func dispatch(request record, root string) record {
	op := source.String(request["operation"])
	switch op {
	case "lookup", "validate", "compatibility":
		if err := checkContractRequest(request, op); err != nil {
			return record{"status": "INVALID", "problem": err.Error()}
		}
		return contractOperation(request, root, op)
	case "restore-source", "project-support-source", "source-rule-units", "validate-unified":
		return sourceOperation(request, op)
	default:
		return record{"status": "INVALID", "problem": "UNKNOWN_OPERATION"}
	}
}

func checkContractRequest(request record, op string) error {
	if _, ok := request["urn"].(string); !ok {
		return fmt.Errorf("URN_STRING_REQUIRED")
	}
	for key := range request {
		switch key {
		case "format", "operation", "urn":
			continue
		case "instance":
			if op == "validate" {
				continue
			}
		case "schema", "meanings", "extension_id":
			if op == "compatibility" {
				continue
			}
		}
		return fmt.Errorf("UNEXPECTED_REQUEST_FIELD")
	}
	if value, present := request["extension_id"]; present && source.String(value) == "" {
		return fmt.Errorf("EXTENSION_ID_MUST_BE_NONEMPTY_STRING")
	}
	return nil
}

func contractOperation(request record, root, op string) record {
	result := record{
		"status": "UNRESOLVED", "reference_resolution": record{"status": "UNRESOLVED"},
		"structure": record{"status": "NOT_REQUESTED"}, "semantics": record{"status": "NOT_REQUESTED"},
		"change_impact": record{"status": "NOT_ASSESSED"},
	}
	if root == "" {
		executable, err := os.Executable()
		if err == nil {
			executable, err = filepath.EvalSymlinks(executable)
		}
		if err != nil {
			result["reference_resolution"] = record{"status": "UNRESOLVED", "problem": err.Error()}
			return result
		}
		root = filepath.Dir(executable)
	}
	r, err := registry.Open(root, registryDigest)
	if err != nil {
		result["reference_resolution"] = record{"status": "UNRESOLVED", "problem": err.Error()}
		return result
	}
	id := source.String(request["urn"])
	entry, base, err := resolver.Resolve(r, id, nil)
	if err != nil {
		result["reference_resolution"] = record{"status": "UNRESOLVED", "urn": id, "problem": err.Error()}
		return result
	}
	result["contract"] = entry
	result["provenance"] = record{"registry_revision": r.Revision, "registry_sha256": r.Digest, "source_revision": sourceRevision}
	result["reference_resolution"] = record{"status": "RESOLVED", "urn": id}
	switch op {
	case "lookup":
		result["status"] = "RESOLVED"
		result["schema"] = base.Docs[id]
	case "validate":
		instance, present := request["instance"]
		if !present {
			result["status"] = "INVALID"
			result["structure"] = record{"status": "INVALID", "problem": "INSTANCE_REQUIRED"}
			return result
		}
		schema, err := validator.Compile(base)
		if err != nil {
			result["structure"] = record{"status": "UNRESOLVED", "problem": err.Error()}
			return result
		}
		result["status"] = "VALID"
		result["structure"] = record{"status": "VALID"}
		if err := schema.Validate(instance); err != nil {
			result["status"] = "INVALID"
			result["structure"] = record{"status": "INVALID", "problem": err.Error()}
		}
	case "compatibility":
		compare(request, r, entry, base, result)
	}
	return result
}

func compare(request record, r *registry.Registry, entry registry.Entry, base resolver.Graph, result record) {
	extension := source.Object(request["schema"])
	if extension == nil {
		result["status"] = "INVALID"
		result["structure"] = record{"status": "INVALID", "problem": "EXTENSION_SCHEMA_REQUIRED"}
		return
	}
	meanings := map[string]string{}
	if raw, exists := request["meanings"]; exists {
		values := source.Object(raw)
		if values == nil {
			result["status"] = "INVALID"
			result["problem"] = "MEANINGS_MUST_BE_OBJECT"
			return
		}
		for pointer, value := range values {
			meaning := source.String(value)
			if meaning == "" {
				result["status"] = "INVALID"
				result["problem"] = "MEANING_ID_MUST_BE_NONEMPTY_STRING"
				return
			}
			meanings[pointer] = meaning
		}
	}
	evidence, _ := json.Marshal(record{"schema": extension, "meanings": meanings})
	digest := fmt.Sprintf("%x", sha256.Sum256(evidence))
	extensionID := source.String(request["extension_id"])
	if extensionID == "" {
		extensionID = "urn:ptsip:extension:sha256:" + digest
	}
	result["extension"] = record{"id": extensionID, "sha256": digest}
	_, graph, err := resolver.Resolve(r, entry.URN, extension)
	if err != nil {
		result["reference_resolution"] = record{"status": "UNRESOLVED", "urn": entry.URN, "problem": err.Error()}
		return
	}
	if _, err := validator.Compile(base); err != nil {
		result["structure"] = record{"status": "UNRESOLVED", "problem": err.Error()}
		return
	}
	if _, err := validator.Compile(graph); err != nil {
		result["status"] = "INVALID"
		result["structure"] = record{"status": "INVALID", "problem": err.Error()}
		return
	}
	result["structure"] = record{"status": "VALID"}
	semantic := compatibility.Evaluate(entry, base.Docs[entry.URN], extension, meanings)
	result["status"] = semantic.Status
	result["semantics"] = semantic
	result["change_impact"] = record{"status": semantic.Status, "scope": "AFFECTED_EXTENSION_ONLY", "base_urn": entry.URN}
}

func sourceOperation(request record, op string) record {
	doc := source.Object(request["document"])
	if op == "project-support-source" {
		var err error
		doc, err = source.Decode([]byte(source.String(request["raw"])))
		if err != nil {
			return fail(err)
		}
	}
	if doc == nil {
		return fail(fmt.Errorf("POLICY_DOCUMENT_MUST_BE_OBJECT"))
	}
	result := record{"status": "VALID"}
	var err error
	switch op {
	case "restore-source":
		result["source"], err = source.Restore(doc)
	case "project-support-source":
		result["source"], err = source.ProjectSupport(doc, source.String(request["path"]))
	case "source-rule-units":
		result["units"], err = source.RuleUnits(doc)
	case "validate-unified":
		root := source.String(request["repository_root"])
		if !filepath.IsAbs(root) {
			return fail(fmt.Errorf("ABSOLUTE_REPOSITORY_ROOT_REQUIRED"))
		}
		verifier, err := source.NewVerifier(root)
		if err != nil {
			return fail(err)
		}
		result, original, err := verifier.Check(doc)
		if err != nil {
			return fail(err)
		}
		if request["restore"] == true {
			result["source"] = original
		}
		return result
	}
	if err != nil {
		return fail(err)
	}
	return result
}

func fail(err error) record {
	return record{"status": "FAIL", "problem": err.Error(), "execution_authorized": false, "mutation_authorized": false}
}
