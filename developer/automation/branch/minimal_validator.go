package branch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ValidateMinimalCreationInput checks only required, user-supplied fields.
// It does not impose PTSIP naming patterns or authorize Git operations.
func ValidateMinimalCreationInput(data []byte) error {
	fields, err := decodeObject(data)
	if err != nil {
		return err
	}
	for _, name := range []string{"branch_name", "base_branch"} {
		raw, present := fields[name]
		if !present {
			return fmt.Errorf("missing required field %q", name)
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
			return fmt.Errorf("field %q must be a nonempty string", name)
		}
	}
	return nil
}

// ValidateMinimalBranchRoster validates a main-owned registry without inferring
// branch activity from naming conventions, Git refs, or commit age.
func ValidateMinimalBranchRoster(data []byte) error {
	fields, err := decodeObject(data)
	if err != nil {
		return err
	}
	var format string
	if raw, ok := fields["schema_version"]; !ok || json.Unmarshal(raw, &format) != nil || format != "ptsip-branch-roster/v1" {
		return fmt.Errorf("schema_version must be ptsip-branch-roster/v1")
	}
	activeSet := map[string]bool{}
	for _, name := range []string{"active_branches", "exception_branches"} {
		raw, present := fields[name]
		if !present || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("field %q must be an array", name)
		}
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil {
			return fmt.Errorf("field %q must be an array of strings: %w", name, err)
		}
		seen := map[string]struct{}{}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("field %q contains a blank branch", name)
			}
			if _, duplicated := seen[value]; duplicated {
				return fmt.Errorf("field %q contains duplicate %q", name, value)
			}
			seen[value] = struct{}{}
			if name == "active_branches" { activeSet[value] = true } else if !activeSet[value] { return fmt.Errorf("exception branch %q is not active", value) }
		}
	}
	return nil
}

func decodeObject(data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("expected a JSON object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return nil, fmt.Errorf("unexpected trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return fields, nil
}
