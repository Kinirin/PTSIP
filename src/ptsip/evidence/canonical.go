package evidence

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

// CanonicalJSON serializes deterministic JSON for contracts bound to evidence.
func CanonicalJSON(value any) (string, error) { return canonicalJSON(value) }

// StableDigest hashes the canonical evidence JSON representation of value.
func StableDigest(value any) (string, error) { return digestJSON(value) }

// Evidence v1 hashes compact, key-sorted UTF-8 JSON. This encoding is part of
// evidence identity, including the distinction between integer and float values.
func canonicalJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	document, err := jsondoc.Decode(raw)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	err = writeCanonical(&out, document)
	return out.String(), err
}

func writeCanonical(out *strings.Builder, value any) error {
	switch v := value.(type) {
	case map[string]any:
		out.WriteByte('{')
		for i, key := range slices.Sorted(maps.Keys(v)) {
			if i > 0 {
				out.WriteByte(',')
			}
			quoteString(out, key)
			out.WriteByte(':')
			if err := writeCanonical(out, v[key]); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	case []any:
		out.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := writeCanonical(out, item); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case string:
		quoteString(out, v)
	case json.Number:
		number := string(v)
		if strings.ContainsAny(number, ".eE") {
			n, err := v.Float64()
			if err != nil {
				return err
			}
			mode := byte('f')
			if abs := math.Abs(n); abs != 0 && (abs < 1e-4 || abs >= 1e16) {
				mode = 'e'
			}
			number = strconv.FormatFloat(n, mode, -1, 64)
			if mode == 'f' && !strings.Contains(number, ".") {
				number += ".0"
			}
		} else if number == "-0" {
			number = "0"
		}
		out.WriteString(number)
	case bool:
		out.WriteString(strconv.FormatBool(v))
	case nil:
		out.WriteString("null")
	default:
		return fmt.Errorf("evidence value is not JSON: %T", v)
	}
	return nil
}

func quoteString(out *strings.Builder, value string) {
	out.WriteByte('"')
	const hex = "0123456789abcdef"
	for _, r := range value {
		switch r {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case '\b':
			out.WriteString(`\b`)
		case '\t':
			out.WriteString(`\t`)
		case '\n':
			out.WriteString(`\n`)
		case '\f':
			out.WriteString(`\f`)
		case '\r':
			out.WriteString(`\r`)
		default:
			if r < 0x20 {
				out.WriteString(`\u00`)
				out.WriteByte(hex[r>>4])
				out.WriteByte(hex[r&0x0f])
			} else {
				out.WriteRune(r)
			}
		}
	}
	out.WriteByte('"')
}
