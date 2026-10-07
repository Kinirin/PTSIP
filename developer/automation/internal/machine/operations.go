package machine

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type OperationHandler func(*Repository, string, map[string]string, []string) (any, error)

var operationHandlers = map[string]OperationHandler{}

func RegisterOperations(namespace string, handler OperationHandler) {
	if namespace == "" || handler == nil || operationHandlers[namespace] != nil {
		panic("duplicate or invalid Go operation registration: " + namespace)
	}
	operationHandlers[namespace] = handler
}

func (r *Repository) DispatchOperation(namespace, command string, options map[string]string, args []string) (any, error) {
	handler := operationHandlers[namespace]
	if handler == nil {
		return nil, fmt.Errorf("no registered native Go implementation for %s", namespace)
	}
	return handler(r, command, options, args)
}

type OperationError struct{ Code, Message string }

func (e *OperationError) Error() string                      { return e.Code + ": " + e.Message }
func Fail(code, message string) error                        { return &OperationError{Code: code, Message: message} }
func BoolOption(options map[string]string, name string) bool { return options["--"+name] == "true" }

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
func LFDigest(data []byte) string {
	return SHA256(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
}

// AtomicWrite preserves existing bytes on validation/CAS/write failures. Paths stay in the repository.
func (r *Repository) AtomicWrite(relative string, content []byte, expectedDigest *string) error {
	target, err := r.Path(relative)
	if err != nil {
		return err
	}
	if expectedDigest != nil {
		existing, readErr := os.ReadFile(target)
		if readErr != nil && !os.IsNotExist(readErr) {
			return readErr
		}
		digest := ""
		if readErr == nil {
			digest = SHA256(existing)
		}
		if digest != *expectedDigest {
			return Fail("STALE_SOURCE", relative+" changed before mutation")
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".ptsip-go-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if expectedDigest != nil {
		existing, readErr := os.ReadFile(target)
		digest := ""
		if readErr != nil && !os.IsNotExist(readErr) {
			return readErr
		}
		if readErr == nil {
			digest = SHA256(existing)
		}
		if digest != *expectedDigest {
			return Fail("STALE_SOURCE", relative+" changed during mutation")
		}
	}
	return os.Rename(temporary.Name(), target)
}
func (r *Repository) WriteJSON(relative string, value any, expected *string) error {
	data, err := CanonicalJSON(value)
	if err != nil {
		return err
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, data, "", "  "); err != nil {
		return err
	}
	return r.AtomicWrite(relative, append(formatted.Bytes(), '\n'), expected)
}
func (r *Repository) WriteYAML(relative string, value any, expected *string) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return r.AtomicWrite(relative, data, expected)
}
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
func UniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
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
func NormalizeReference(value string) string { return strings.ReplaceAll(value, "\\", "/") }
