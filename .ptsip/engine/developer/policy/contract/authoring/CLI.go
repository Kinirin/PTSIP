package authoring

import (
	"fmt"
	"path/filepath"
	"strings"

	legacy "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/check"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
)

func ReadDocument(root, relative string) (Document, error) {
	return legacy.ReadDocument(root, relative)
}

func LocalFile(root, relative string) (string, error) {
	return legacy.LocalFile(root, relative)
}

func WriteDocument(root, relative string, doc Document) error {
	return legacy.WriteDocument(root, relative, doc)
}

func Run(root, selected, operation, family, input, output string, validate Validator) (any, int) {
	report := Document{"status": "FAIL", "scope": "CONTRACT_STRUCTURE_AND_REFERENCES", "execution_authorized": false}
	reject := func(err error) (any, int) { report["problem"] = err.Error(); return report, 2 }
	if selected == "responsibility-authority" {
		return legacy.Run(root, selected, operation, family, input, output, validate)
	}
	if selected != "policy-rules" {
		return reject(fmt.Errorf("UNKNOWN_AUTHORING_INTERFACE: %s", selected))
	}
	if output != "" && operation != "generate" {
		return reject(fmt.Errorf("OUTPUT_REQUIRES_GENERATE_OPERATION"))
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return reject(err)
	}
	if operation == "describe" {
		boundary, err := ReadDocument(absolute, BoundaryPath)
		if err != nil {
			return reject(err)
		}
		if _, declared := boundary["unified_binding"]; declared {
			selection, err := entry.SelectUnifiedPolicy(absolute, "PTSIP_DEVELOPER_POLICY", family)
			if err != nil {
				return reject(err)
			}
			schema, err := ReadDocument(absolute, selection.ContractRefs["public_schema"])
			if err != nil {
				return reject(err)
			}
			report["status"], report["selection"], report["contract"] = "DESCRIBED", selection, schema
			report["structural_state"], report["normative_approval"] = "CANDIDATE", false
			return report, 0
		}
	}
	doc := Document{}
	if input != "" {
		doc, err = ReadDocument(absolute, input)
		if err != nil {
			return reject(err)
		}
		if entry.IsUnifiedPolicy(doc) {
			return runUnifiedAuthoring(absolute, operation, family, output, doc)
		}
	}
	catalog, err := Load(absolute)
	if err != nil {
		return reject(err)
	}
	if err := catalog.CheckContracts(validate); err != nil {
		return reject(err)
	}
	if operation == "describe" {
		if catalog.Contracts[family] == nil {
			return reject(fmt.Errorf("SELECT_REGISTERED_ROOT_FAMILY"))
		}
		selection, err := entry.SelectPolicy(catalog.Root, "PTSIP_DEVELOPER_POLICY", family)
		if err != nil {
			return reject(err)
		}
		report["selection"] = selection
		report["status"], report["contract"] = "DESCRIBED", catalog.Contracts[family]
		return report, 0
	}
	if operation == "lookup-intention" {
		return reject(fmt.Errorf("LOOKUP_INTENTION_REQUIRES_AUTHORIZATION_ID"))
	}
	if operation != "check" && operation != "generate" {
		return reject(fmt.Errorf("UNSUPPORTED_AUTHORING_OPERATION: %s", operation))
	}
	status := "VALID"
	if len(doc) != 0 || operation != "generate" || family != "" {
		selection, err := entry.SelectPolicyDocument(catalog.Root, doc)
		if err != nil {
			return reject(err)
		}
		report["selection"] = selection
		if output != "" && (!strings.HasPrefix(output, selection.OutputRoot+"/") ||
			(strings.ToLower(filepath.Ext(output)) != ".yaml" && strings.ToLower(filepath.Ext(output)) != ".yml")) {
			return reject(fmt.Errorf("POLICY_OUTPUT_OUTSIDE_REGISTERED_ROUTE: %s", output))
		}
	}
	if operation == "generate" {
		doc, status, err = catalog.GenerateRules(validate, doc, family)
	} else {
		if family != "" {
			return reject(fmt.Errorf("CHECK_USES_DOCUMENT_ROOT_FAMILY"))
		}
		err = catalog.CheckRules(validate, doc)
	}
	if err != nil {
		return reject(err)
	}
	if output != "" && status != "OMITTED" {
		if err := WriteDocument(catalog.Root, output, doc); err != nil {
			return reject(err)
		}
		report["output"] = output
	}
	report["status"], report["document"], report["family_contracts_checked"] = status, doc, len(catalog.Families)
	return report, 0
}

func runUnifiedAuthoring(root, operation, family, output string, doc Document) (any, int) {
	report := Document{"status": "FAIL", "scope": "STRUCTURE_SOURCE_INTEGRITY_AND_DECLARED_REFERENCES",
		"execution_authorized": false, "mutation_authorized": false, "lifecycle_transition_authorized": false,
		"policy_conformance": "NOT_ASSERTED", "normative_approval": false, "structural_state": "CANDIDATE"}
	reject := func(err error) (any, int) { report["problem"] = err.Error(); return report, 2 }
	if operation != "check" && operation != "generate" {
		return reject(fmt.Errorf("UNSUPPORTED_AUTHORING_OPERATION: %s", operation))
	}
	if operation == "check" && family != "" {
		return reject(fmt.Errorf("CHECK_USES_DOCUMENT_ROOT_FAMILY"))
	}
	selection, err := entry.SelectPolicyDocument(root, doc)
	if err != nil {
		return reject(err)
	}
	report["selection"] = selection
	if family != "" && family != selection.RootFamily {
		return reject(fmt.Errorf("ROOT_FAMILY_CONFLICT: %s/%s", family, selection.RootFamily))
	}
	validation, err := check.CheckUnifiedDocument(root, doc)
	if err != nil {
		return reject(err)
	}
	for key, value := range validation {
		report[key] = value
	}
	if output != "" {
		if !strings.HasPrefix(output, selection.OutputRoot+"/") ||
			(strings.ToLower(filepath.Ext(output)) != ".yaml" && strings.ToLower(filepath.Ext(output)) != ".yml") {
			return reject(fmt.Errorf("POLICY_OUTPUT_OUTSIDE_REGISTERED_ROUTE: %s", output))
		}
		if err := WriteDocument(root, output, doc); err != nil {
			return reject(err)
		}
		report["output"] = output
	}
	report["status"] = "VALID"
	if operation == "generate" {
		report["status"] = "GENERATED"
	}
	report["document"] = doc
	return report, 0
}
