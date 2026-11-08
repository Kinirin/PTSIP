// Package resolver follows only installed, exact-version schema references.
package resolver

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/registry"
)

const ExtensionID = "urn:ptsip:request:extension"

type Graph struct {
	Root string
	Docs map[string]map[string]any
}

func Resolve(r *registry.Registry, id string, extension map[string]any) (registry.Entry, Graph, error) {
	entry, schema, err := r.Schema(id)
	if err != nil {
		return entry, Graph{}, err
	}
	g := Graph{Root: id, Docs: map[string]map[string]any{id: schema}}
	if extension != nil {
		g.Root = ExtensionID
		g.Docs[ExtensionID] = extension
	}
	seen := map[string]bool{}
	visiting := map[string]bool{}
	var visit func(string, string) error
	visit = func(resource, pointer string) error {
		key := resource + "#" + pointer
		if visiting[key] {
			return fmt.Errorf("REFERENCE_CYCLE: %s", key)
		}
		if seen[key] {
			return nil
		}
		visiting[key] = true
		doc, found := g.Docs[resource]
		if !found {
			_, doc, err = r.Schema(resource)
			if err != nil {
				return err
			}
			g.Docs[resource] = doc
		}
		sub, ok := Pointer(doc, pointer)
		if !ok {
			return fmt.Errorf("MISSING_SCHEMA_POINTER: %s", key)
		}
		schema, object := sub.(map[string]any)
		if !object {
			if _, boolean := sub.(bool); !boolean {
				return fmt.Errorf("INVALID_SCHEMA_POINTER: %s", key)
			}
		}
		if _, nested := schema["$id"]; nested && (pointer != "" || resource == ExtensionID) {
			return fmt.Errorf("UNREGISTERED_SCHEMA_ID: %s", key)
		}
		if _, dynamic := schema["$dynamicRef"]; dynamic {
			return fmt.Errorf("UNSUPPORTED_DYNAMIC_REFERENCE: %s", key)
		}
		if raw, exists := schema["$ref"]; exists {
			ref, ok := raw.(string)
			if !ok || ref == "" {
				return fmt.Errorf("INVALID_SCHEMA_REFERENCE: %s", key)
			}
			base, fragment, _ := strings.Cut(ref, "#")
			if base == "" {
				base = resource
			} else if base == ExtensionID {
				return fmt.Errorf("UNREGISTERED_URN: %s", base)
			}
			fragment, err := url.PathUnescape(fragment)
			if err != nil || (fragment != "" && !strings.HasPrefix(fragment, "/")) {
				return fmt.Errorf("UNSUPPORTED_SCHEMA_FRAGMENT: %s", ref)
			}
			if err := visit(base, fragment); err != nil {
				return err
			}
		}
		for _, child := range children(schema) {
			if err := visit(resource, pointer+child); err != nil {
				return err
			}
		}
		delete(visiting, key)
		seen[key] = true
		return nil
	}
	if err := visit(id, ""); err != nil {
		return entry, Graph{}, err
	}
	if err := visit(g.Root, ""); err != nil {
		return entry, Graph{}, err
	}
	return entry, g, nil
}

func children(schema map[string]any) []string {
	var result []string
	for _, key := range []string{"additionalProperties", "unevaluatedProperties", "propertyNames", "items", "contains", "unevaluatedItems", "not", "if", "then", "else", "contentSchema"} {
		if _, exists := schema[key]; exists {
			result = append(result, "/"+key)
		}
	}
	for _, key := range []string{"properties", "patternProperties", "dependentSchemas", "$defs"} {
		entries, _ := schema[key].(map[string]any)
		for name := range entries {
			result = append(result, "/"+key+"/"+Escape(name))
		}
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
		entries, _ := schema[key].([]any)
		for i := range entries {
			result = append(result, "/"+key+"/"+strconv.Itoa(i))
		}
	}
	sort.Strings(result)
	return result
}

func Escape(token string) string {
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}

func Pointer(value any, pointer string) (any, bool) {
	if pointer == "" {
		return value, true
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, false
	}
	for _, token := range strings.Split(pointer[1:], "/") {
		for i := 0; i < len(token); i++ {
			if token[i] == '~' && (i+1 == len(token) || (token[i+1] != '0' && token[i+1] != '1')) {
				return nil, false
			}
		}
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		switch node := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = node[token]
			if !ok {
				return nil, false
			}
		case []any:
			i, err := strconv.Atoi(token)
			if err != nil || i < 0 || i >= len(node) || strconv.Itoa(i) != token {
				return nil, false
			}
			value = node[i]
		default:
			return nil, false
		}
	}
	return value, true
}
