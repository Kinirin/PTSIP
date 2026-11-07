// Package review collects explicitly selected historical evidence and inventories
// declared fields. It is an authoring aid, not a schema validator or authority.
package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractentry"
)

const Format = "ptsip-non-normative-policy-review/v1"

type Finding struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type Target struct {
	Path             string                   `json:"path"`
	Document         any                      `json:"document"`
	Selection        *contractentry.Selection `json:"selection,omitempty"`
	SelectionProblem string                   `json:"selection_problem,omitempty"`
	FieldFindings    []Finding                `json:"field_findings"`
}

type Source struct {
	Path    string `json:"path"`
	Pointer string `json:"pointer"`
	Value   any    `json:"value"`
}

type Packet struct {
	Format             string            `json:"format"`
	Status             string            `json:"status"`
	SchemaValidation   string            `json:"schema_validation"`
	SemanticSelection  string            `json:"semantic_selection"`
	MutationAuthorized bool              `json:"mutation_authorized"`
	Files              map[string]string `json:"file_sha256"`
	Targets            []Target          `json:"targets"`
	Sources            []Source          `json:"sources"`
}

func digest(root, path string) (string, error) {
	file, err := authoring.LocalFile(root, path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	if len(data) > 4*1024*1024 {
		return "", fmt.Errorf("REVIEW_INPUT_TOO_LARGE: %s", path)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Pointer implements exact JSON Pointer lookup; no fuzzy names, wildcards,
// source_scope expansion or semantic relevance selection is performed.
func Pointer(value any, pointer string) (any, error) {
	if pointer == "" {
		return value, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("INVALID_POINTER: %s", pointer)
	}
	for _, token := range strings.Split(pointer[1:], "/") {
		for i := 0; i < len(token); i++ {
			if token[i] == '~' {
				if i+1 >= len(token) || (token[i+1] != '0' && token[i+1] != '1') {
					return nil, fmt.Errorf("INVALID_POINTER_ESCAPE")
				}
				i++
			}
		}
		key := strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		switch node := value.(type) {
		case map[string]any:
			var found bool
			value, found = node[key]
			if !found {
				return nil, fmt.Errorf("POINTER_NOT_FOUND: %s", pointer)
			}
		case []any:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(node) || strconv.Itoa(index) != key {
				return nil, fmt.Errorf("INVALID_POINTER_INDEX: %s", key)
			}
			value = node[index]
		default:
			return nil, fmt.Errorf("POINTER_NOT_FOUND: %s", pointer)
		}
	}
	return value, nil
}

func child(path, key string) string {
	return path + "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

// Inventory compares literal properties/items only. Open objects and complex
// schema branches remain explicitly uninspected. Findings are review requests,
// including fields JSON Schema may permit via additionalProperties.
func Inventory(schema map[string]any, value any, path string) []Finding {
	findings := []Finding{}
	switch value.(type) {
	case map[string]any, []any:
	default:
		return findings // This is field inventory, not scalar/schema validation.
	}
	for _, key := range []string{"$ref", "allOf", "anyOf", "oneOf", "if", "patternProperties", "dependentSchemas"} {
		if _, exists := schema[key]; exists {
			return append(findings, Finding{path, "UNINSPECTED_SCHEMA"})
		}
	}
	switch node := value.(type) {
	case map[string]any:
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			return append(findings, Finding{path, "OPEN_OBJECT_REQUIRES_REVIEW"})
		}
		if required, ok := schema["required"].([]any); ok {
			for _, raw := range required {
				if key, ok := raw.(string); ok {
					if _, exists := node[key]; !exists {
						findings = append(findings, Finding{child(path, key), "MISSING_DECLARED_FIELD"})
					}
				}
			}
		}
		keys := make([]string, 0, len(node))
		for key := range node {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			sub, declared := properties[key].(map[string]any)
			if !declared {
				findings = append(findings, Finding{child(path, key), "UNDECLARED_FIELD"})
				continue
			}
			findings = append(findings, Inventory(sub, node[key], child(path, key))...)
		}
	case []any:
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return append(findings, Finding{path, "UNINSPECTED_ARRAY"})
		}
		for i, item := range node {
			findings = append(findings, Inventory(items, item, child(path, strconv.Itoa(i)))...)
		}
	}
	return findings
}

func Build(root string, targets, sources []string) (Packet, error) {
	packet := Packet{Format: Format, Status: "NON_NORMATIVE_REVIEW", SchemaValidation: "NOT_PERFORMED",
		SemanticSelection: "HUMAN_OR_AGENT_SELECTED_NOT_VERIFIED", Files: map[string]string{}, Targets: []Target{}, Sources: []Source{}}
	if len(targets) == 0 {
		return packet, fmt.Errorf("EXPLICIT_TARGET_REQUIRED")
	}
	record := func(path string) error {
		hash, err := digest(root, path)
		if err == nil {
			if before, seen := packet.Files[path]; seen && before != hash {
				return fmt.Errorf("INPUT_CHANGED_DURING_REVIEW: %s", path)
			}
			packet.Files[path] = hash
		}
		return err
	}
	read := func(path string) (map[string]any, error) {
		if err := record(path); err != nil {
			return nil, err
		}
		doc, err := authoring.ReadDocument(root, path)
		if err != nil {
			return nil, err
		}
		if err := record(path); err != nil {
			return nil, err
		}
		return doc, nil
	}
	// These are the existing selector's bootstrap inputs, not a new route.
	for _, path := range []string{".ptsip/index.json", ".ptsip/contracts/index.json", ".ptsip/engine/developer/index.json"} {
		if err := record(path); err != nil {
			return packet, err
		}
	}
	for _, path := range targets {
		doc, err := read(path)
		if err != nil {
			return packet, err
		}
		target := Target{Path: path, Document: doc, FieldFindings: []Finding{}}
		selection, err := contractentry.Resolve(root, path)
		if err != nil {
			target.SelectionProblem = err.Error()
		} else {
			target.Selection = &selection
			for name, ref := range selection.ContractRefs {
				if name == "family_fields" {
					continue // Registered schema URN + fragment, not a local file.
				}
				if err := record(ref); err != nil {
					return packet, err
				}
			}
			if authoring.IsUnifiedPolicy(doc) {
				schema, err := read(selection.ContractRefs["public_schema"])
				if err != nil {
					return packet, err
				}
				target.FieldFindings = Inventory(schema, doc, "")
				packet.Targets = append(packet.Targets, target)
				continue
			}
			rulesSchema, err := read(selection.ContractRefs["selection_schema"])
			if err != nil {
				return packet, err
			}
			resourceID, fragment, ok := strings.Cut(selection.ContractRefs["family_fields"], "#")
			if !ok || rulesSchema["$id"] != resourceID {
				return packet, fmt.Errorf("SELECTED_SCHEMA_REFERENCE_UNRESOLVED")
			}
			if _, err := Pointer(rulesSchema, fragment); err != nil {
				return packet, err
			}
			registry, err := read(selection.ContractRefs["v2_registry"])
			if err != nil {
				return packet, err
			}
			// Capture the registered resource files, without resolving a URN as
			// a filesystem path or treating historical evidence as a contract.
			artifacts, ok := registry["artifacts"].([]any)
			if !ok {
				return packet, fmt.Errorf("V2_SCHEMA_ARTIFACTS_MISSING")
			}
			for _, raw := range artifacts {
				item, _ := raw.(map[string]any)
				if item["kind"] != "SCHEMA" {
					continue
				}
				ref, _ := item["path"].(string)
				if err := authoring.CheckPolicyContractPath(ref); err != nil {
					return packet, err
				}
				if err := record(ref); err != nil {
					return packet, err
				}
			}
			external, _ := registry["external_resources"].(map[string]any)
			for _, raw := range external {
				item, _ := raw.(map[string]any)
				ref, _ := item["path"].(string)
				if err := authoring.CheckPolicyContractPath(ref); err != nil {
					return packet, err
				}
				if err := record(ref); err != nil {
					return packet, err
				}
			}
			if rules, present := doc["Policy_Rules"]; present {
				target.FieldFindings = Inventory(rulesSchema, rules, "/Policy_Rules")
			} else {
				target.FieldFindings = append(target.FieldFindings, Finding{"/Policy_Rules", "MISSING_DECLARED_FIELD"})
			}
		}
		packet.Targets = append(packet.Targets, target)
	}
	seen := map[string]bool{}
	for _, spec := range sources {
		path, pointer, ok := strings.Cut(spec, "#")
		if !ok || !strings.HasPrefix(pointer, "/") || seen[spec] {
			return packet, fmt.Errorf("EXPLICIT_UNIQUE_SOURCE_SECTION_REQUIRED: %s", spec)
		}
		seen[spec] = true
		doc, err := read(path)
		if err != nil {
			return packet, err
		}
		value, err := Pointer(doc, pointer)
		if err != nil {
			return packet, fmt.Errorf("%s: %w", path, err)
		}
		packet.Sources = append(packet.Sources, Source{path, pointer, value})
	}
	// Catch changes to any captured input while the packet was assembled.
	if err := packet.Verify(root); err != nil {
		return packet, err
	}
	return packet, nil
}

// Verify checks the captured file bytes only, not relevance, authenticity,
// translation fidelity, approval, or completeness of a contract graph.
func (p Packet) Verify(root string) error {
	if p.Format != Format || len(p.Files) == 0 {
		return fmt.Errorf("INVALID_REVIEW_SNAPSHOT")
	}
	paths := make([]string, 0, len(p.Files))
	for path := range p.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		want := p.Files[path]
		if bytes, err := hex.DecodeString(want); err != nil || len(bytes) != sha256.Size {
			return fmt.Errorf("INVALID_REVIEW_DIGEST: %s", path)
		}
		got, err := digest(root, path)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("REVIEW_INPUT_CHANGED: %s", path)
		}
	}
	return nil
}

func ReadSnapshot(root, path string) (Packet, error) {
	file, err := authoring.LocalFile(root, path)
	if err != nil {
		return Packet{}, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return Packet{}, err
	}
	if len(data) > 16*1024*1024 {
		return Packet{}, fmt.Errorf("REVIEW_SNAPSHOT_TOO_LARGE")
	}
	var packet Packet
	err = json.Unmarshal(data, &packet)
	return packet, err
}
