// Package jsondoc decodes finite JSON with exact numbers and unique keys.
package jsondoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

func Decode(raw []byte) (any, error) {
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("INVALID_UTF8")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	value, err := read(d, 0)
	if err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("TRAILING_JSON_DATA")
	}
	return value, nil
}

func read(d *json.Decoder, depth int) (any, error) {
	if depth > 512 {
		return nil, fmt.Errorf("JSON_NESTING_LIMIT")
	}
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		value := make(map[string]any)
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return nil, err
			}
			name := key.(string)
			if _, exists := value[name]; exists {
				return nil, fmt.Errorf("DUPLICATE_MAPPING_KEY: %s", name)
			}
			value[name], err = read(d, depth+1)
			if err != nil {
				return nil, err
			}
		}
		_, err := d.Token()
		return value, err
	case json.Delim('['):
		value := []any{}
		for d.More() {
			item, err := read(d, depth+1)
			if err != nil {
				return nil, err
			}
			value = append(value, item)
		}
		_, err := d.Token()
		return value, err
	default:
		return token, nil
	}
}
