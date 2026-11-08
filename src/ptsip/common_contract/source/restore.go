package source

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
)

type partition struct {
	children map[string]*partition
	value    any
	leaf     bool
}

func tokens(pointer string) ([]string, error) {
	if pointer == "" || pointer == "#" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("INVALID_SOURCE_POINTER")
	}
	parts := strings.Split(pointer[1:], "/")
	for i, token := range parts {
		for j := 0; j < len(token); j++ {
			if token[j] == '~' && (j+1 == len(token) || (token[j+1] != '0' && token[j+1] != '1')) {
				return nil, fmt.Errorf("INVALID_SOURCE_POINTER_ESCAPE")
			}
		}
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
	}
	return parts, nil
}

func Restore(doc Document) (Document, error) {
	if doc["schema_version"] != Version {
		return nil, fmt.Errorf("UNSUPPORTED_UNIFIED_POLICY_VERSION")
	}
	content := Object(doc["Policy_Content"])
	items := Array(content["items"])
	if len(items) == 0 {
		return nil, fmt.Errorf("SOURCE_ITEMS_REQUIRED")
	}
	var tree partition
	ids := map[string]bool{}
	for _, raw := range items {
		item := Object(raw)
		id := String(item["id"])
		if id == "" || ids[id] {
			return nil, fmt.Errorf("INVALID_OR_DUPLICATE_SOURCE_ITEM: %s", id)
		}
		ids[id] = true
		value, present := item["value"]
		if !present {
			return nil, fmt.Errorf("SOURCE_ITEM_VALUE_REQUIRED")
		}
		pointer, ok := item["source_pointer"].(string)
		if !ok {
			return nil, fmt.Errorf("INVALID_SOURCE_POINTER")
		}
		parts, err := tokens(pointer)
		if err != nil || len(parts) > 512 {
			return nil, fmt.Errorf("INVALID_SOURCE_POINTER: %s", pointer)
		}
		node := &tree
		for _, part := range parts {
			if node.leaf {
				return nil, fmt.Errorf("OVERLAPPING_SOURCE_POINTERS")
			}
			if node.children == nil {
				node.children = map[string]*partition{}
			}
			if node.children[part] == nil {
				node.children[part] = &partition{}
			}
			node = node.children[part]
		}
		if node.leaf || len(node.children) != 0 {
			return nil, fmt.Errorf("DUPLICATE_OR_OVERLAPPING_SOURCE_POINTER")
		}
		node.leaf, node.value = true, value
	}
	value, err := tree.rebuild()
	if err != nil {
		return nil, err
	}
	restored := Object(value)
	if restored == nil {
		return nil, fmt.Errorf("RESTORED_SOURCE_MUST_BE_OBJECT")
	}
	expected := String(Object(content["source_document"])["sha256"])
	if !digestPattern.MatchString(expected) {
		return nil, fmt.Errorf("SOURCE_DIGEST_REQUIRED")
	}
	actual, err := Digest(restored)
	if err != nil {
		return nil, err
	}
	if actual != expected {
		return nil, fmt.Errorf("SOURCE_DIGEST_MISMATCH")
	}
	return restored, nil
}

func (p *partition) rebuild() (any, error) {
	if p.leaf {
		return clone(p.value), nil
	}
	numeric := true
	for key := range p.children {
		numeric = numeric && indexPattern.MatchString(key)
	}
	if numeric {
		result := make([]any, len(p.children))
		for i := range result {
			child := p.children[strconv.Itoa(i)]
			if child == nil {
				return nil, fmt.Errorf("NONCONTIGUOUS_SOURCE_ARRAY")
			}
			var err error
			result[i], err = child.rebuild()
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	result := Document{}
	for _, key := range slices.Sorted(maps.Keys(p.children)) {
		var err error
		result[key], err = p.children[key].rebuild()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func clone(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(Document, len(v))
		for key, item := range v {
			out[key] = clone(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = clone(item)
		}
		return out
	default:
		return value
	}
}

func Identity(source Document) Document {
	if id := Object(source["Policy_Identity"]); id != nil {
		return Document{"id": id["id"], "policy_kind": id["policy_kind"], "root_family": id["root_family"], "title": id["title"]}
	}
	policy := Object(source["policy"])
	return Document{"id": policy["id"], "policy_kind": source["policy_class"], "root_family": source["responsibility_family"], "title": policy["title"]}
}

func Status(value any) any {
	if doc := Object(value); doc != nil {
		if status, exists := doc["status"]; exists {
			return status
		}
		if status, exists := Object(doc["constant"])["status"]; exists {
			return status
		}
		for _, key := range []string{"Policy_Lifecycle", "Policy_Status", "policy"} {
			if nested := Object(doc[key]); nested != nil {
				return Status(nested)
			}
		}
	}
	return nil
}

func ProjectSupport(doc Document, relative string) (Document, error) {
	// Validate native documents during the build, then let the caller retain their raw bytes.
	if doc["schema_version"] != Version {
		return nil, nil
	}
	restored, err := Restore(doc)
	if err != nil {
		return nil, err
	}
	id := Identity(restored)
	declared := Object(doc["Policy_Identity"])
	if len(declared) != len(id) || id["policy_kind"] != "PTSIP_SUPPORT_FEATURE" {
		return nil, fmt.Errorf("SUPPORT_PROJECTION_IDENTITY_MISMATCH: %s", relative)
	}
	for key, value := range id {
		if String(value) == "" || value != declared[key] {
			return nil, fmt.Errorf("SUPPORT_PROJECTION_IDENTITY_MISMATCH: %s", relative)
		}
	}
	content := Object(doc["Policy_Content"])
	expected := String(id["root_family"]) + "/" + String(id["id"]) + ".yaml"
	if relative != expected || !Relative(relative) || Object(content["source_document"])["path"] != "src/policy/"+expected {
		return nil, fmt.Errorf("SUPPORT_PROJECTION_PATH_MISMATCH: %s", relative)
	}
	pointer := "/policy"
	if _, exists := restored["Policy_Identity"]; exists {
		pointer = "/Policy_Status"
		if _, exists := restored["Policy_Lifecycle"]; exists {
			pointer = "/Policy_Lifecycle"
		}
	}
	lifecycle := Object(doc["Policy_Lifecycle"])
	status := String(Status(restored))
	ref := String(lifecycle["source_ref"])
	for _, item := range Array(content["items"]) {
		item := Object(item)
		if ref != "" && item["id"] == ref && item["source_pointer"] == pointer &&
			status != "" && lifecycle["status"] == status {
			return restored, nil
		}
	}
	return nil, fmt.Errorf("SUPPORT_PROJECTION_LIFECYCLE_MISMATCH: %s", relative)
}

func Relative(relative string) bool {
	return relative != "" && relative != "." && relative != ".." && !path.IsAbs(relative) &&
		path.Clean(relative) == relative && !strings.HasPrefix(relative, "../") &&
		!strings.ContainsAny(relative, "\\:\x00\r\n") &&
		!strings.HasPrefix(strings.ToLower(relative), "developer/temporary_record/")
}
