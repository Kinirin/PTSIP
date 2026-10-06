package planning

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Object = map[string]any

// Repository is the minimal control-plane surface Planning Automation consumes.
// The owning implementation remains outside the planning domain; Planning only
// depends on these mechanical repository operations.
type Repository interface {
	Read(string) (Object, error)
	Path(string) (string, error)
	Scope(string) (string, error)
	Validate(string, any) error
	AtomicWrite(string, []byte, *string) error
	DispatchOperation(string, string, map[string]string, []string) (any, error)
	RootDir() string
}

func Map(value any) Object  { object, _ := value.(map[string]any); return object }
func Text(value any) string { text, _ := value.(string); return text }
func List(value any) []any  { list, _ := value.([]any); return list }

func Strings(value any) []string {
	if input, ok := value.([]string); ok {
		return input
	}
	result := []string{}
	for _, item := range List(value) {
		result = append(result, Text(item))
	}
	return result
}

func Has(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func CanonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

func SHA256(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func ppGit(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("GIT_OPERATION_FAILED: %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func ppOptional(text string) any {
	if text == "" {
		return nil
	}
	return text
}

func ppYAML(raw []byte, label string, required bool) (Object, error) {
	if raw == nil {
		if required {
			return nil, fmt.Errorf("REQUIRED_AUTHORITY_INPUT_MISSING: %s", label)
		}
		return nil, nil
	}
	var value any
	if err := yaml.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("AUTHORITY_INPUT_INVALID: %s: %w", label, err)
	}
	normalized, err := normalize(value)
	if err != nil {
		return nil, err
	}
	if Map(normalized) == nil {
		return nil, fmt.Errorf("AUTHORITY_INPUT_INVALID: %s must be mapping", label)
	}
	return Map(normalized), nil
}

func normalize(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			v[key] = normalized
		}
	case map[any]any:
		out := Object{}
		for key, child := range v {
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("non-string mapping key")
			}
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			out[name] = normalized
		}
		return out, nil
	case []any:
		for i, child := range v {
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			v[i] = normalized
		}
	}
	return value, nil
}

func planningAtomicWrite(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ptsip-write-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(raw); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}
