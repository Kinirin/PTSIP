package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const engineIndex = "engines/index.json"
const transportID = "urn:ptsip:engine-transport:json-stdio:v1"

type engineEntry struct {
	Kind       string   `json:"kind"`
	ID         string   `json:"engine_id"`
	Version    string   `json:"version"`
	Capability string   `json:"capability_id"`
	Contract   string   `json:"contract_ref"`
	Manifest   string   `json:"manifest_ref"`
	Binding    string   `json:"execution_binding_ref"`
	Operations []string `json:"operations"`
}

type indexDocument struct {
	Schema     string        `json:"schema_version"`
	Class      string        `json:"policy_class"`
	Transport  string        `json:"transport_contract_ref"`
	Entries    []engineEntry `json:"official_engine_entries"`
	Federation struct {
		Owner     string `json:"owner_id"`
		Execution string `json:"execution_registry_ref"`
	} `json:"federation"`
}

type sourceRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

type engineBinding struct {
	EngineID        string    `json:"engine_id"`
	EngineVersion   string    `json:"engine_version"`
	CapabilityID    string    `json:"capability_id"`
	ContractVersion string    `json:"contract_version"`
	Entrypoint      string    `json:"entrypoint"`
	Transport       sourceRef `json:"transport_ref"`
	Permission      *bool     `json:"permission_grant"`
}

func readDocument[T any](root, relative string, dst *T) (string, error) {
	if !fs.ValidPath(relative) || relative == "." || strings.ContainsAny(relative, "\\:") {
		return "", fmt.Errorf("invalid resource path %q", relative)
	}
	filename := filepath.Join(root, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return "", err
	}
	inside, err := filepath.Rel(root, resolved)
	if err != nil || filepath.ToSlash(inside) != relative {
		return "", fmt.Errorf("resource path is an alias: %s", relative)
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("invalid resource file: %s", relative)
	}
	const max = 4 << 20
	raw, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return "", err
	}
	if len(raw) > max {
		return "", fmt.Errorf("resource size limit exceeded: %s", relative)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return "", fmt.Errorf("%s: %w", relative, err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func readEngineResource[T any](root, reference string, dst *T) (string, error) {
	relative, ok := strings.CutPrefix(reference, "src/ptsip/")
	if !ok || !strings.HasPrefix(relative, "engines/") {
		return "", fmt.Errorf("engine resource outside its package: %q", reference)
	}
	return readDocument(root, relative, dst)
}

func bindingFor(root string, index indexDocument, entry engineEntry) (engineBinding, error) {
	var registry struct {
		Owner      string `json:"owner_id"`
		Permission *bool  `json:"execution_permission"`
		Bindings   []struct {
			EngineID string `json:"engine_id"`
			Ref      string `json:"binding_ref"`
		} `json:"bindings"`
	}
	var result engineBinding
	if _, err := readEngineResource(root, index.Federation.Execution, &registry); err != nil {
		return result, err
	}
	if registry.Owner == "" || registry.Owner != index.Federation.Owner || registry.Permission == nil || *registry.Permission {
		return result, errors.New("inconsistent execution registry owner or permission")
	}
	count := 0
	for _, row := range registry.Bindings {
		if row.EngineID == entry.ID {
			count++
			if row.Ref != entry.Binding {
				return result, errors.New("execution binding reference differs from the index")
			}
		}
	}
	if count != 1 {
		return result, errors.New("missing or ambiguous execution registration")
	}
	var file struct {
		Bindings []engineBinding `json:"bindings"`
	}
	if _, err := readEngineResource(root, entry.Binding, &file); err != nil {
		return result, err
	}
	if len(file.Bindings) != 1 {
		return result, errors.New("missing or ambiguous native binding")
	}
	result = file.Bindings[0]
	if result.EngineID != entry.ID || result.EngineVersion != entry.Version ||
		result.CapabilityID != entry.Capability || result.ContractVersion == "" ||
		result.Permission == nil || *result.Permission || result.Transport.Path != index.Transport {
		return result, errors.New("inconsistent native binding")
	}
	var transport struct {
		ID         string `json:"contract_id"`
		Version    string `json:"version"`
		Request    string `json:"request_encoding"`
		Response   string `json:"response_encoding"`
		Permission *bool  `json:"execution_permission_from_binding"`
	}
	digest, err := readEngineResource(root, result.Transport.Path, &transport)
	if err != nil {
		return result, err
	}
	if transport.ID != transportID || transport.ID != result.Transport.ID ||
		transport.Version != "1.0.0" || transport.Version != result.Transport.Version ||
		transport.Request != "ONE_UTF8_JSON_OBJECT_ON_STDIN_EOF" ||
		transport.Response != "ONE_UTF8_JSON_OBJECT_ON_STDOUT" ||
		digest != result.Transport.SHA256 || transport.Permission == nil || *transport.Permission {
		return result, errors.New("inconsistent native transport")
	}
	return result, nil
}

func route(root, scripts string, index indexDocument, operation string) error {
	var selected *engineEntry
	for _, entry := range index.Entries {
		for _, declared := range entry.Operations {
			if operation != declared {
				continue
			}
			if selected != nil {
				return fmt.Errorf("ambiguous operation owner: %s", operation)
			}
			selected = &entry
		}
	}
	if selected == nil {
		return fmt.Errorf("unregistered engine operation: %s", operation)
	}
	entry := *selected
	if entry.ID == "" || entry.Version == "" || entry.Kind == "" || entry.Capability == "" {
		return errors.New("incomplete engine identity")
	}
	binding, err := bindingFor(root, index, entry)
	if err != nil {
		return err
	}
	var contract struct {
		ID         string `json:"contract_id"`
		Engine     string `json:"engine_id"`
		Owner      string `json:"semantic_owner"`
		Version    string `json:"version"`
		Operations []struct {
			ID string `json:"operation_id"`
		} `json:"operations"`
	}
	digest, err := readEngineResource(root, entry.Contract, &contract)
	if err != nil {
		return err
	}
	count := 0
	for _, row := range contract.Operations {
		if row.ID == operation {
			count++
		}
	}
	if contract.ID != entry.Capability || contract.Engine != entry.ID || contract.Owner != entry.Kind ||
		contract.Version != binding.ContractVersion || count != 1 {
		return errors.New("operation does not resolve to the bound engine contract")
	}
	var manifest struct {
		ID           string      `json:"engine_id"`
		Origin       string      `json:"origin"`
		Version      string      `json:"implementation_version"`
		Capabilities []sourceRef `json:"capabilities"`
	}
	if _, err := readEngineResource(root, entry.Manifest, &manifest); err != nil {
		return err
	}
	expected := sourceRef{ID: contract.ID, Version: contract.Version, Path: entry.Contract, SHA256: digest}
	count = 0
	for _, capability := range manifest.Capabilities {
		if capability == expected {
			count++
		}
	}
	if manifest.ID != entry.ID || manifest.Origin != "OFFICIAL" || manifest.Version != entry.Version || count != 1 {
		return errors.New("operation contract differs from its engine manifest")
	}

	name := binding.Entrypoint
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\:") {
		return errors.New("invalid registered executable name")
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		name += ".exe"
	}
	path := filepath.Join(scripts, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s executable unavailable in the selected directory", entry.Kind)
	}
	// Read only the routing field. The engine owns the envelope, input and
	// operation validation and receives the original bytes.
	const limit = 1 << 20
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, limit+1))
	if err != nil {
		return err
	}
	if len(raw) > limit {
		return errors.New("request exceeds engine transport limit")
	}
	var envelope struct {
		Operation string `json:"operation_id"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("invalid routing envelope: %w", err)
	}
	if envelope.Operation != operation {
		return errors.New("request operation differs from the selected route")
	}
	cmd := exec.Command(path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(raw), os.Stdout, os.Stderr
	return cmd.Run()
}

func listOperations(index indexDocument) error {
	type operation struct {
		ID    string `json:"operation_id"`
		Owner string `json:"engine"`
	}
	rows := make([]operation, 0)
	for _, entry := range index.Entries {
		for _, id := range entry.Operations {
			rows = append(rows, operation{ID: id, Owner: entry.Kind})
		}
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Schema     string      `json:"schema_version"`
		Operations []operation `json:"operations"`
		Permission bool        `json:"execution_authorized"`
	}{Schema: "ptsip-registered-operations/v1", Operations: rows})
}

const usage = "Usage: ptsip help | operations <ptsip-root> | engine <ptsip-root> <scripts-dir> <operation-id>\nEngine commands pass one complete versioned JSON request from stdin to the registered native CLI."

func run(args []string) error {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		_, err := fmt.Fprintln(os.Stdout, usage)
		return err
	}
	if !(len(args) == 2 && args[0] == "operations") && !(len(args) == 4 && args[0] == "engine") {
		return errors.New(usage)
	}
	root, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	var index indexDocument
	if _, err := readDocument(root, engineIndex, &index); err != nil {
		return err
	}
	if index.Schema != "ptsip-official-engine-index/v1" || index.Class != "PTSIP_SUPPORT_FEATURE" {
		return errors.New("unsupported engine index")
	}
	if args[0] == "operations" {
		return listOperations(index)
	}
	scripts, err := filepath.Abs(args[2])
	if err != nil {
		return err
	}
	return route(root, scripts, index, args[3])
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		var childExit *exec.ExitError
		if errors.As(err, &childExit) && childExit.ExitCode() >= 0 {
			os.Exit(childExit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
