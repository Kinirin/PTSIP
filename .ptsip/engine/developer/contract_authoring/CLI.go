package contract_authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// YAML is serialization only. All schema validation stays in the installed
// PTSIP engine, using its existing Python runtime for YAML support.
const decodeYAML = `import json, sys, yaml
class UniqueLoader(yaml.SafeLoader): pass
def mapping(loader, node, deep=False):
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if not isinstance(key, str): raise ValueError('mapping keys must be strings')
        if key in result: raise ValueError('duplicate mapping key: ' + key)
        result[key] = loader.construct_object(value_node, deep=deep)
    return result
UniqueLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, mapping)
data = yaml.load(sys.stdin.buffer.read().decode('utf-8-sig'), Loader=UniqueLoader)
if not isinstance(data, dict): raise ValueError('document must be an object')
sys.stdout.buffer.write(json.dumps(data, ensure_ascii=False, allow_nan=False).encode('utf-8'))
`

func pythonSerialization(script string, data []byte) ([]byte, error) {
	python := os.Getenv("PTSIP_PYTHON")
	if python == "" {
		python = "python"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-c", script)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	result, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("YAML_SERIALIZATION_FAILED: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return result, nil
}

func ReadDocument(root, relative string) (Document, error) {
	var data []byte
	var err error
	if relative == "-" {
		data, err = io.ReadAll(io.LimitReader(os.Stdin, 4*1024*1024+1))
	} else {
		file, pathErr := LocalFile(root, relative)
		if pathErr != nil {
			return nil, pathErr
		}
		data, err = os.ReadFile(file)
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 4*1024*1024 {
		return nil, fmt.Errorf("INPUT_DOCUMENT_TOO_LARGE")
	}
	encoded, err := pythonSerialization(decodeYAML, data)
	if err != nil {
		return nil, err
	}
	var result Document
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func WriteDocument(root, relative string, doc Document) error {
	if relative == "" || filepath.IsAbs(relative) || strings.ContainsAny(relative, "\\:\x00") ||
		filepath.ToSlash(filepath.Clean(relative)) != relative {
		return fmt.Errorf("INVALID_OUTPUT_PATH: %s", relative)
	}
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("INVALID_OUTPUT_PATH: %s", relative)
		}
	}
	// Check existing ancestors without creating any directories or replacing files.
	parent := filepath.Dir(filepath.FromSlash(relative))
	if parent != "." {
		current := root
		for _, part := range strings.Split(filepath.ToSlash(parent), "/") {
			current = filepath.Join(current, part)
			info, err := os.Lstat(current)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("OUTPUT_PARENT_UNAVAILABLE_OR_SYMLINK: %s", current)
			}
		}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	switch strings.ToLower(filepath.Ext(relative)) {
	case ".json":
		data = append(data, '\n')
	case ".yaml", ".yml":
		data, err = pythonSerialization("import json,sys,yaml; value=json.loads(sys.stdin.buffer.read().decode('utf-8')); sys.stdout.buffer.write(yaml.safe_dump(value,allow_unicode=True,sort_keys=False).encode('utf-8'))", data)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("OUTPUT_MUST_BE_JSON_OR_YAML")
	}
	file, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(relative)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("OUTPUT_NOT_CREATED: %w", err)
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func Run(root, selected, operation, family, input, output string, validate Validator) (any, int) {
	report := Document{"status": "FAIL", "scope": "CONTRACT_STRUCTURE_AND_REFERENCES", "execution_authorized": false}
	reject := func(err error) (any, int) { report["problem"] = err.Error(); return report, 2 }
	if output != "" && operation != "generate" {
		return reject(fmt.Errorf("OUTPUT_REQUIRES_GENERATE_OPERATION"))
	}
	if family != "" && selected != "policy-rules" {
		return reject(fmt.Errorf("FAMILY_ONLY_APPLIES_TO_POLICY_RULES"))
	}
	catalog, err := Load(root)
	if err != nil {
		return reject(err)
	}
	if err := catalog.CheckContracts(validate); err != nil {
		return reject(err)
	}
	if operation == "describe" {
		if selected != "policy-rules" {
			return reject(fmt.Errorf("DESCRIBE_ONLY_APPLIES_TO_POLICY_RULES"))
		}
		if catalog.Contracts[family] == nil {
			return reject(fmt.Errorf("SELECT_REGISTERED_ROOT_FAMILY"))
		}
		report["status"], report["contract"] = "DESCRIBED", catalog.Contracts[family]
		return report, 0
	}
	if operation != "check" && operation != "generate" {
		return reject(fmt.Errorf("UNSUPPORTED_AUTHORING_OPERATION: %s", operation))
	}
	doc := Document{}
	if input == "" && selected == "responsibility-authority" && operation == "check" {
		input = ProfilePath
	}
	if input != "" {
		doc, err = ReadDocument(catalog.Root, input)
		if err != nil {
			return reject(err)
		}
	}
	status := "VALID"
	if selected == "policy-rules" {
		if operation == "generate" {
			doc, status, err = catalog.GenerateRules(validate, doc, family)
		} else {
			if family != "" {
				return reject(fmt.Errorf("CHECK_USES_DOCUMENT_ROOT_FAMILY"))
			}
			err = catalog.CheckRules(validate, doc)
		}
	} else if selected == "responsibility-authority" {
		if operation == "generate" {
			doc, err = catalog.GenerateProfile(validate, doc)
			status = "GENERATED"
		} else {
			err = catalog.CheckProfile(validate, doc)
		}
	} else {
		return reject(fmt.Errorf("UNKNOWN_AUTHORING_INTERFACE: %s", selected))
	}
	if err != nil {
		return reject(err)
	}
	if output != "" && status != "OMITTED" {
		if err := WriteDocument(catalog.Root, output, doc); err != nil {
			return reject(err)
		}
		report["output"] = output
	}
	report["status"], report["document"], report["family_contracts_checked"] = status, doc, len(catalog.Families)
	return report, 0
}
