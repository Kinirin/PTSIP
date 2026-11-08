// Package source preserves the original source trees of unified policy documents.
package source

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const Version = "ptsip-unified-policy/v3-candidate"
const MaxBytes = 16 * 1024 * 1024

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var indexPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

type Document = map[string]any

func Object(value any) Document {
	object, _ := value.(map[string]any)
	return object
}

func Array(value any) []any {
	items, _ := value.([]any)
	return items
}

func String(value any) string {
	s, _ := value.(string)
	return s
}

func Decode(raw []byte) (Document, error) {
	if len(raw) > MaxBytes || !utf8.Valid(raw) {
		return nil, fmt.Errorf("INVALID_POLICY_SIZE_OR_ENCODING")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var node yaml.Node
	if err := decoder.Decode(&node); err != nil || len(node.Content) != 1 {
		return nil, fmt.Errorf("INVALID_POLICY_YAML: %v", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("TRAILING_POLICY_DOCUMENT")
	}
	remaining := MaxBytes
	value, err := yamlValue(node.Content[0], 0, &remaining)
	if err != nil {
		return nil, err
	}
	doc := Object(value)
	if doc == nil {
		return nil, fmt.Errorf("POLICY_DOCUMENT_MUST_BE_OBJECT")
	}
	return doc, nil
}

func yamlValue(node *yaml.Node, depth int, remaining *int) (any, error) {
	*remaining -= len(node.Value) + 1
	if depth > 512 || *remaining < 0 {
		return nil, fmt.Errorf("POLICY_DOCUMENT_LIMIT")
	}
	switch node.Kind {
	case yaml.MappingNode:
		doc := Document{}
		for i := 0; i < len(node.Content); i += 2 {
			key, err := yamlValue(node.Content[i], depth+1, remaining)
			name, ok := key.(string)
			if err != nil || !ok {
				return nil, fmt.Errorf("MAPPING_KEY_MUST_BE_STRING")
			}
			if _, exists := doc[name]; exists {
				return nil, fmt.Errorf("DUPLICATE_MAPPING_KEY: %s", name)
			}
			doc[name], err = yamlValue(node.Content[i+1], depth+1, remaining)
			if err != nil {
				return nil, err
			}
		}
		return doc, nil
	case yaml.SequenceNode:
		items := make([]any, len(node.Content))
		for i, child := range node.Content {
			value, err := yamlValue(child, depth+1, remaining)
			if err != nil {
				return nil, err
			}
			items[i] = value
		}
		return items, nil
	case yaml.AliasNode:
		return yamlValue(node.Alias, depth+1, remaining)
	case yaml.ScalarNode:
		if node.Tag == "!!str" && node.Style == 0 {
			switch strings.ToLower(node.Value) {
			case "yes", "on":
				return true, nil
			case "no", "off":
				return false, nil
			}
		}
		switch node.Tag {
		case "!!str", "!!null", "!!bool", "!!int", "!!float":
			var value any
			return valueAfterDecode(node, &value)
		}
	}
	return nil, fmt.Errorf("UNSUPPORTED_POLICY_YAML_TAG: %s", node.Tag)
}

func valueAfterDecode(node *yaml.Node, value *any) (any, error) {
	if err := node.Decode(value); err != nil {
		return nil, err
	}
	if number, ok := (*value).(float64); ok {
		text, err := floatText(number)
		return json.Number(text), err
	}
	return *value, nil
}

func floatText(value float64) (string, error) {
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return "", fmt.Errorf("SOURCE_MUST_BE_FINITE_JSON")
	}
	mode := byte('f')
	if n := math.Abs(value); n != 0 && (n < 1e-4 || n >= 1e16) {
		mode = 'e'
	}
	number := strconv.FormatFloat(value, mode, -1, 64)
	if mode == 'f' && !strings.Contains(number, ".") {
		number += ".0"
	}
	return number, nil
}

func Digest(value any) (string, error) {
	var b strings.Builder
	if err := canonical(&b, value, 0); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String()))), nil
}

// The existing source digest uses Python's sorted, ASCII JSON encoding.
func canonical(b *strings.Builder, value any, depth int) error {
	if depth > 512 {
		return fmt.Errorf("SOURCE_NESTING_LIMIT")
	}
	switch v := value.(type) {
	case map[string]any:
		b.WriteByte('{')
		for i, key := range slices.Sorted(maps.Keys(v)) {
			if i > 0 {
				b.WriteByte(',')
			}
			quote(b, key)
			b.WriteByte(':')
			if err := canonical(b, v[key], depth+1); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := canonical(b, item, depth+1); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case string:
		quote(b, v)
	case float64:
		number, err := floatText(v)
		if err != nil {
			return err
		}
		b.WriteString(number)
	case json.Number:
		if strings.ContainsAny(string(v), ".eE") {
			f, err := v.Float64()
			if err != nil {
				return err
			}
			return canonical(b, f, depth+1)
		}
		if v == "-0" {
			v = "0"
		}
		b.WriteString(string(v))
	case nil, bool, int, int64, uint64:
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		b.Write(raw)
	default:
		return fmt.Errorf("SOURCE_MUST_BE_FINITE_JSON: %T", value)
	}
	return nil
}

func quote(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\b':
			b.WriteString("\\b")
		case '\t':
			b.WriteString("\\t")
		case '\n':
			b.WriteString("\\n")
		case '\f':
			b.WriteString("\\f")
		case '\r':
			b.WriteString("\\r")
		default:
			switch {
			case r >= 0x20 && r <= 0x7e:
				b.WriteRune(r)
			case r <= 0xffff:
				fmt.Fprintf(b, "\\u%04x", r)
			default:
				hi, lo := utf16.EncodeRune(r)
				fmt.Fprintf(b, "\\u%04x\\u%04x", hi, lo)
			}
		}
	}
	b.WriteByte('"')
}
