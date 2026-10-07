package machine

import (
	"encoding/json"
	"fmt"
)

func decodeJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return token, nil
	}
	switch delimiter {
	case '{':
		value := Object{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, fmt.Errorf("JSON object key must be a string")
			}
			if _, exists := value[key]; exists {
				return nil, Fail("DUPLICATE_JSON_KEY", key)
			}
			child, err := decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			value[key] = child
		}
		if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
			return nil, fmt.Errorf("invalid JSON object closing token: %v", err)
		}
		return value, nil
	case '[':
		value := []any{}
		for decoder.More() {
			child, err := decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			value = append(value, child)
		}
		if token, err := decoder.Token(); err != nil || token != json.Delim(']') {
			return nil, fmt.Errorf("invalid JSON array closing token: %v", err)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}
