// Package compatibility proves preservation for an explicitly supported schema fragment.
package compatibility

import (
	"maps"
	"slices"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/registry"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
)

type Finding struct {
	Pointer   string `json:"pointer"`
	Code      string `json:"code"`
	Base      any    `json:"base,omitempty"`
	Extension any    `json:"extension,omitempty"`
}

type Result struct {
	Status   string    `json:"status"`
	Scope    string    `json:"scope"`
	Findings []Finding `json:"findings"`
}

type domain struct {
	typeName string
	values   []string // nil denotes an unbounded domain.
}

type shape struct {
	properties map[string]domain
	required   []string
	open       bool
}

// Evaluate assumes both schemas have already passed structural compilation.
// It proves the intersection and inclusion of independent optional/required
// properties constrained by a single type, string enum or string constant.
func Evaluate(entry registry.Entry, base, extension map[string]any, meanings map[string]string) Result {
	result := Result{Status: "SUPPORTED", Scope: "DECLARED_MEANING_AND_OBJECT_PROPERTY_CONSTRAINTS", Findings: []Finding{}}
	add := func(pointer, code string, expected, actual any, conflict bool) {
		result.Findings = append(result.Findings, Finding{pointer, code, expected, actual})
		if conflict {
			result.Status = "INCOMPATIBLE"
		} else if result.Status == "SUPPORTED" {
			result.Status = "UNRESOLVED"
		}
	}
	for _, pointer := range slices.Sorted(maps.Keys(entry.Semantics)) {
		expected := entry.Semantics[pointer]
		actual := meanings[pointer]
		if actual == "" {
			add(pointer, "MEANING_EVIDENCE_MISSING", expected, nil, false)
		} else if expected != actual {
			add(pointer, "MEANING_CONFLICT", expected, actual, true)
		}
	}
	baseShape, baseOK := objectShape(base)
	constraints, composed := directConjunction(extension, entry.URN)
	extShape, extensionOK := objectShape(constraints)
	if !baseOK || !extensionOK {
		add("", "UNSUPPORTED_SCHEMA_COMPARISON", nil, nil, false)
		return result
	}
	for _, pointer := range slices.Sorted(maps.Keys(meanings)) {
		if _, baseMeaning := entry.Semantics[pointer]; !baseMeaning {
			value, exists := resolver.Pointer(constraints, pointer)
			_, property := value.(map[string]any)
			if !exists || !property || slices.Contains(slices.Collect(maps.Values(entry.Semantics)), meanings[pointer]) {
				add(pointer, "UNPROVEN_ADDITIONAL_MEANING", nil, meanings[pointer], false)
			}
		}
	}
	for _, field := range slices.Sorted(maps.Keys(baseShape.properties)) {
		pointer := "/properties/" + resolver.Escape(field)
		baseDomain := baseShape.properties[field]
		extDomain, present := extShape.properties[field]
		if !present && !extShape.open {
			add(pointer, "COMMON_FIELD_FORBIDDEN", true, false, true)
			continue
		}
		if present && !overlaps(baseDomain, extDomain) {
			add(pointer, "CONTRADICTORY_CONSTRAINT", baseDomain.evidence(), extDomain.evidence(), true)
		} else if !composed && !subset(extDomain, baseDomain) {
			add(pointer, "WEAKENED_CONSTRAINT", baseDomain.evidence(), extDomain.evidence(), true)
		}
	}
	for _, field := range baseShape.required {
		if !composed && !slices.Contains(extShape.required, field) {
			add("/required", "REQUIRED_FIELD_WEAKENED", field, nil, true)
		}
	}
	return result
}

func directConjunction(schema map[string]any, id string) (map[string]any, bool) {
	all, ok := schema["allOf"].([]any)
	if !ok || len(all) != 2 {
		return schema, false
	}
	for key := range schema {
		switch key {
		case "allOf", "$schema", "title", "description", "$comment":
		default:
			return schema, false
		}
	}
	for i, item := range all {
		ref, _ := item.(map[string]any)
		if len(ref) == 1 && ref["$ref"] == id {
			branch, ok := all[1-i].(map[string]any)
			if ok {
				return branch, true
			}
		}
	}
	return schema, false
}

func objectShape(schema map[string]any) (shape, bool) {
	result := shape{properties: map[string]domain{}, open: true}
	if schema["type"] != "object" {
		return result, false
	}
	for key, value := range schema {
		switch key {
		case "$schema", "$id", "title", "description", "$comment", "type":
		case "properties":
			properties, _ := value.(map[string]any)
			for field, child := range properties {
				property, ok := child.(map[string]any)
				if !ok {
					return result, false
				}
				constraint, ok := propertyDomain(property)
				if !ok {
					return result, false
				}
				result.properties[field] = constraint
			}
		case "required":
			fields, _ := value.([]any)
			for _, field := range fields {
				result.required = append(result.required, field.(string))
			}
			slices.Sort(result.required)
		case "additionalProperties":
			open, ok := value.(bool)
			if !ok {
				return result, false
			}
			result.open = open
		default:
			return result, false
		}
	}
	for _, field := range result.required {
		if _, present := result.properties[field]; !present && !result.open {
			return result, false
		}
	}
	return result, true
}

func propertyDomain(schema map[string]any) (domain, bool) {
	d := domain{}
	var constant *string
	for key, value := range schema {
		switch key {
		case "title", "description", "$comment":
		case "type":
			var ok bool
			d.typeName, ok = value.(string)
			if !ok {
				return d, false
			}
		case "enum":
			d.values = []string{}
			for _, item := range value.([]any) {
				text, ok := item.(string)
				if !ok {
					return d, false
				}
				d.values = append(d.values, text)
			}
			slices.Sort(d.values)
		case "const":
			text, ok := value.(string)
			if !ok {
				return d, false
			}
			constant = &text
		default:
			return d, false
		}
	}
	if constant != nil {
		if d.values == nil || slices.Contains(d.values, *constant) {
			d.values = []string{*constant}
		} else {
			d.values = []string{}
		}
	}
	if d.values != nil && !typeIncludes(d.typeName, "string") {
		d.values = []string{}
	}
	return d, d.values == nil || len(d.values) != 0
}

func (d domain) evidence() map[string]any {
	return map[string]any{"type": d.typeName, "allowed_strings": d.values}
}

func typeIncludes(outer, inner string) bool {
	return outer == "" || outer == inner || (outer == "number" && inner == "integer")
}

func (d domain) allowsString(value string) bool {
	return typeIncludes(d.typeName, "string") && (d.values == nil || slices.Contains(d.values, value))
}

func overlaps(a, b domain) bool {
	if a.values == nil && b.values == nil {
		return typeIncludes(a.typeName, b.typeName) || typeIncludes(b.typeName, a.typeName)
	}
	if a.values == nil {
		a, b = b, a
	}
	return slices.ContainsFunc(a.values, b.allowsString)
}

func subset(inner, outer domain) bool {
	if inner.values == nil {
		return outer.values == nil && typeIncludes(outer.typeName, inner.typeName)
	}
	for _, value := range inner.values {
		if !outer.allowsString(value) {
			return false
		}
	}
	return true
}
