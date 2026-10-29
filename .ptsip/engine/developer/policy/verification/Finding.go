// Findings and limited schema-corpus inspection for the developer tool.
// A limited check is never a JSON Schema metaschema or policy-conformance PASS.
package schema_check

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Finding struct {
	File           string   `json:"file"`
	Status         string   `json:"status"`
	Problem        string   `json:"problem,omitempty"`
	UnverifiedRefs []string `json:"unverified_refs,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}
type Report struct {
	Status                    string        `json:"status"`
	Scope                     string        `json:"scope"`
	Engine                    string        `json:"engine"`
	PackageVersion            string        `json:"package_version,omitempty"`
	Total                     int           `json:"total"`
	Failed                    int           `json:"failed"`
	UnverifiedReferences      int           `json:"unverified_references"`
	UnresolvedScaffolds       int           `json:"unresolved_scaffolds"`
	NonSchemaJSONFiles        int           `json:"non_schema_json_files"`
	PolicyConformance         string        `json:"policy_conformance"`
	FullJSONSchemaConformance string        `json:"full_json_schema_conformance"`
	StatusProbe               *StatusProbe  `json:"status_probe,omitempty"`
	RoutingProbe              *RoutingProbe `json:"routing_probe,omitempty"`
	Findings                  []Finding     `json:"findings"`
}

func collectRefs(v any, refs *[]string) {
	switch x := v.(type) {
	case map[string]any:
		for key, value := range x {
			if key == "$ref" {
				if s, ok := value.(string); ok {
					*refs = append(*refs, s)
				}
			}
			collectRefs(value, refs)
		}
	case []any:
		for _, item := range x {
			collectRefs(item, refs)
		}
	}
}

// approvedSchemaPaths lists current Policy_Status and Policy_version sources.
// Missing current contracts must fail the schema scan.
var approvedSchemaPaths = []string{
	".ptsip/contracts/schemas/Policy/Policy_Status/Public_Policy_Status.schema.json",
	".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json",
}

const schemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

// These six files are user-approved unresolved, fail-closed RBAC scaffolds.
// Recognizing their exact shape does not activate or validate RBAC instances.
var approvedFailClosedRBACScaffolds = map[string]bool{
	".ptsip/contracts/schemas/compile-sealed-rbac/responsibility.schema.json":  true,
	".ptsip/contracts/schemas/compile-sealed-rbac/condition-atom.schema.json":  true,
	".ptsip/contracts/schemas/compile-sealed-rbac/permission-atom.schema.json": true,
	".ptsip/contracts/schemas/compile-sealed-rbac/binding.schema.json":         true,
	".ptsip/contracts/schemas/compile-sealed-rbac/sealed-contract.schema.json": true,
	".ptsip/contracts/schemas/compile-sealed-rbac/activation.schema.json":      true,
}

func isSchemaJSON(path string) bool {
	return strings.HasSuffix(path, ".schema.json") || strings.HasSuffix(path, ".schemas.json")
}

func validateFailClosedRBACScaffold(path string, obj map[string]any) bool {
	if !approvedFailClosedRBACScaffolds[path] || len(obj) != 5 {
		return false
	}
	if obj["$schema"] != schemaDraft202012 || obj["type"] != "object" {
		return false
	}
	title, titleOK := obj["title"].(string)
	comment, commentOK := obj["$comment"].(string)
	guard, guardOK := obj["not"].(map[string]any)
	return titleOK && title != "" && commentOK &&
		strings.Contains(comment, "SCAFFOLD_ONLY") &&
		strings.Contains(comment, "Reject all instances") &&
		guardOK && len(guard) == 0
}

var schemaScanRoots = []string{
	".ptsip/contracts/schemas",
	".ptsip/contracts/policy/lifecycle/Policy_version",
}

// Family authoring schemas have been replaced by data contracts. Their
// reappearance anywhere below Policy_Rules is an unresolved old dependency.
func isFamilyRulesAuthoringSchema(relative string) bool {
	if !strings.HasPrefix(relative, ".ptsip/contracts/schemas/Policy/Policy_Rules/") {
		return false
	}
	idx := strings.LastIndex(relative, "/")
	file := relative[idx+1:]
	for _, suffix := range []string{"_rules.schema.json", "_rules.schemas.json"} {
		if !strings.HasSuffix(file, suffix) {
			continue
		}
		family := strings.TrimSuffix(file, suffix)
		if family == "" {
			return false
		}
		for _, letter := range family {
			if !((letter >= 'A' && letter <= 'Z') || (letter >= '0' && letter <= '9')) {
				return false
			}
		}
		return true
	}
	return false
}

func schemaFiles(root string) ([]string, []Finding) {
	paths := []string{}
	findings := []Finding{}
	found := map[string]bool{}
	for _, relativeRoot := range schemaScanRoots {
		folder := filepath.Join(root, filepath.FromSlash(relativeRoot))
		count := 0
		err := filepath.WalkDir(folder, func(path string, item fs.DirEntry, walkError error) error {
			if walkError != nil {
				return walkError
			}
			if item.IsDir() {
				relative, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				if filepath.ToSlash(relative) == ".ptsip/contracts/schemas/Policy/Policy_Rules/abortively" {
					findings = append(findings, Finding{File: filepath.ToSlash(relative), Status: "FAIL", Problem: "DISALLOWED_POLICY_RULES_ARCHIVE_DIRECTORY"})
					return filepath.SkipDir
				}
			}
			if !item.IsDir() && strings.HasSuffix(item.Name(), ".json") {
				relative, relError := filepath.Rel(root, path)
				if relError != nil {
					return relError
				}
				relative = filepath.ToSlash(relative)
				if isFamilyRulesAuthoringSchema(relative) {
					findings = append(findings, Finding{File: relative, Status: "FAIL", Problem: "RETIRED_FAMILY_RULES_SCHEMA"})
				}
				found[relative] = true
				paths = append(paths, path)
				count++
			}
			return nil
		})
		if err != nil {
			findings = append(findings, Finding{File: relativeRoot, Status: "FAIL", Problem: "SCHEMA_ROOT_UNAVAILABLE: " + err.Error()})
		} else if count == 0 {
			findings = append(findings, Finding{File: relativeRoot, Status: "FAIL", Problem: "SCHEMA_ROOT_EMPTY"})
		}
	}
	for _, relative := range approvedSchemaPaths {
		if !found[relative] {
			findings = append(findings, Finding{File: relative, Status: "FAIL", Problem: "MISSING_APPROVED_SCHEMA"})
		}
	}
	sort.Strings(paths)
	return paths, findings
}

func Run(root string) (Report, int) {
	report := Report{Status: "FAIL", Scope: "LIMITED_SCHEMA_JSON_AND_ENGINE_SHAPE_CHECK",
		Engine: "INSTALLED_PTSIP_VALIDATION", PolicyConformance: "NOT_ASSERTED",
		FullJSONSchemaConformance: "NOT_ASSERTED", Findings: []Finding{}}
	e, err := installed()
	if err != nil {
		report.Failed++
		report.Findings = append(report.Findings, Finding{File: "<installed-VALIDATION>", Status: "FAIL", Problem: err.Error()})
		return report, 2
	}
	report.PackageVersion = e.PackageVersion
	paths, problems := schemaFiles(root)
	report.Failed += len(problems)
	report.Findings = append(report.Findings, problems...)
	ids := map[string]string{}
	for _, path := range paths {
		relative, _ := filepath.Rel(root, path)
		relative = filepath.ToSlash(relative)
		f := Finding{File: relative, Status: "LIMITED_CHECK_PASS"}
		report.Total++
		data, readError := os.ReadFile(path)
		var doc any
		if readError == nil {
			readError = json.Unmarshal(data, &doc)
		}
		if readError != nil {
			f.Status = "FAIL"
			f.Problem = "INVALID_JSON: " + readError.Error()
		} else if obj, ok := doc.(map[string]any); !ok {
			f.Status = "FAIL"
			f.Problem = "JSON_SCHEMA_ROOT_NOT_OBJECT"
		} else if !isSchemaJSON(relative) {
			// A JSON data index is not a JSON Schema and cannot be required to have $id.
			f.Status = "NON_SCHEMA_JSON_SYNTAX_ONLY"
			report.NonSchemaJSONFiles++
		} else if approvedFailClosedRBACScaffolds[relative] {
			if !validateFailClosedRBACScaffold(relative, obj) {
				f.Status = "FAIL"
				f.Problem = "RBAC_SCAFFOLD_NO_LONGER_FAIL_CLOSED"
			} else {
				f.Status = "UNRESOLVED_FAIL_CLOSED_SCAFFOLD"
				f.Warnings = append(f.Warnings, "RBAC_FIELDS_UNRESOLVED_NO_INSTANCE_AUTHORIZED")
				report.UnresolvedScaffolds++
			}
		} else {
			if id, valid := obj["$id"].(string); valid && id != "" {
				if prior, found := ids[id]; found {
					f.Status = "FAIL"
					f.Problem = "DUPLICATE_SCHEMA_ID_WITH: " + prior
				} else {
					ids[id] = relative
				}
			}
			if len(obj) <= 2 {
				f.Warnings = append(f.Warnings, "UNCONSTRAINED_OR_PLACEHOLDER_SCHEMA")
			}
			collectRefs(obj, &f.UnverifiedRefs)
			sort.Strings(f.UnverifiedRefs)
			report.UnverifiedReferences += len(f.UnverifiedRefs)
			if f.Status != "FAIL" {
				if error := nativeCheck(e, doc); error != nil {
					f.Status = "FAIL"
					f.Problem = error.Error()
				}
			}
		}
		if f.Status == "FAIL" {
			report.Failed++
		}
		report.Findings = append(report.Findings, f)
	}
	// Status shape probe is independent of the limited document-shape scan.
	// It checks the flat shared Policy_Status schema through the installed engine.
	// Full policy document conformance remains outside this validator.
	if len(problems) == 0 {
		probe := probeApprovedFlatPolicyStatus(root, e)
		report.StatusProbe = &probe
		if probe.Status != "PASS" {
			report.Failed++
		}
		routing := probePolicyRouting(root, e)
		report.RoutingProbe = &routing
		if routing.Status != "PASS" {
			report.Failed++
			report.Findings = append(report.Findings, Finding{File: policyRoutingData, Status: "FAIL", Problem: routing.Problem})
		}
	}
	if report.Failed == 0 {
		report.Status = "LIMITED_CHECK_PASS"
	}
	if report.Failed > 0 {
		return report, 2
	}
	return report, 0
}
