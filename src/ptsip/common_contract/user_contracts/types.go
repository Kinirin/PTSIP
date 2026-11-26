package usercontracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	Dialect        = "https://json-schema.org/draft/2020-12/schema"
	PolicyKey      = "ptsip.policy.v4"
	VersionTarget  = "Policy_Identity.Version"
	CategoryTarget = "Policy_Identity.Policy_Category"
)

type Issue struct {
	Code    string `json:"code"`
	Path    []any  `json:"path"`
	Message string `json:"message"`
}

type ContractResult struct {
	Name    *string  `json:"name"`
	Targets []string `json:"targets"`
	Status  string   `json:"status"`
	Issues  []Issue  `json:"issues"`
}

// ParseJSON accepts one UTF-8 JSON value and rejects duplicate object keys.
func ParseJSON(data []byte) (any, error) {
	if !utf8.Valid(data) {
		return nil, errors.New("JSON must contain valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := decodeValue(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("expected exactly one JSON value")
		}
		return nil, err
	}
	return value, nil
}

func decodeValue(decoder *json.Decoder) (any, error) {
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
		object := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("expected JSON object key")
			}
			if _, exists := object[key]; exists {
				return nil, fmt.Errorf("duplicate JSON key: %q", key)
			}
			value, err := decodeValue(decoder)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return object, nil
	case '[':
		array := make([]any, 0)
		for decoder.More() {
			value, err := decodeValue(decoder)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return array, nil
	default:
		return nil, errors.New("unexpected JSON closing delimiter")
	}
}

func validationIssues(schema *jsonschema.Schema, value any, code string, prefix []any) []Issue {
	result := make([]Issue, 0)
	err := schema.Validate(value)
	if err == nil {
		return result
	}
	var validation *jsonschema.ValidationError
	if !errors.As(err, &validation) {
		return []Issue{{Code: code, Path: append([]any{}, prefix...), Message: err.Error()}}
	}
	var collect func(*jsonschema.ValidationError)
	collect = func(problem *jsonschema.ValidationError) {
		if len(problem.Causes) > 0 {
			for _, cause := range problem.Causes {
				collect(cause)
			}
			return
		}
		location := append([]any{}, prefix...)
		current := value
		for _, part := range problem.InstanceLocation {
			switch node := current.(type) {
			case []any:
				index, parseErr := strconv.Atoi(part)
				if parseErr == nil && index >= 0 && index < len(node) {
					location = append(location, index)
					current = node[index]
					continue
				}
			case map[string]any:
				current = node[part]
			}
			location = append(location, part)
		}
		result = append(result, Issue{Code: code, Path: location, Message: problem.Error()})
	}
	collect(validation)
	return result
}
