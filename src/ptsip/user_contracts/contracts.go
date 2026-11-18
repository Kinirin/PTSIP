package usercontracts

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed templates/*.json
var templates embed.FS

var standardBindings = map[string][]string{
	"evaluation_state.json":  {"Policy_Rules.evaluation.state"},
	"policy_content_id.json": {"Policy_Content.items[].id", "Policy_Rules.rules[].content.content_id[]"},
	"rules_patterns.json":    {"Policy_Rules.rules[].id"},
}

type targetPart struct {
	name   string
	arrays int
}

type targetValue struct {
	value any
	path  []any
}

type selectedContract struct {
	document schemaDocument
	targets  []string
}

var fieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func parseTarget(target string) []targetPart {
	parts := make([]targetPart, 0)
	for _, text := range strings.Split(target, ".") {
		part := targetPart{}
		for strings.HasSuffix(text, "[]") {
			part.arrays++
			text = strings.TrimSuffix(text, "[]")
		}
		if !fieldName.MatchString(text) {
			return nil
		}
		part.name = text
		parts = append(parts, part)
	}
	return parts
}

func targetValues(value any, target string) []targetValue {
	values := []targetValue{{value: value, path: []any{}}}
	parts := parseTarget(target)
	if parts == nil {
		return nil
	}
	for _, part := range parts {
		children := make([]targetValue, 0)
		for _, parent := range values {
			if fields, ok := parent.value.(map[string]any); ok {
				if child, present := fields[part.name]; present {
					childPath := append(append([]any{}, parent.path...), part.name)
					children = append(children, targetValue{value: child, path: childPath})
				}
			}
		}
		for depth := 0; depth < part.arrays; depth++ {
			elements := make([]targetValue, 0)
			for _, child := range children {
				if array, ok := child.value.([]any); ok {
					for index, item := range array {
						itemPath := append(append([]any{}, child.path...), index)
						elements = append(elements, targetValue{value: item, path: itemPath})
					}
				}
			}
			children = elements
		}
		values = children
	}
	return values
}

func (v *Validator) targetExists(target string) bool {
	nodes := []any{v.policy}
	parts := parseTarget(target)
	if parts == nil {
		return false
	}
	for _, part := range parts {
		next := make([]any, 0)
		for _, raw := range nodes {
			node, _ := raw.(map[string]any)
			if ref, ok := node["$ref"].(string); ok {
				node, _ = v.documents[ref].(map[string]any)
			}
			properties, _ := node["properties"].(map[string]any)
			if child, present := properties[part.name]; present {
				next = append(next, child)
			}
		}
		for depth := 0; depth < part.arrays; depth++ {
			items := make([]any, 0)
			for _, raw := range next {
				node, _ := raw.(map[string]any)
				if child, ok := node["items"].(map[string]any); ok {
					items = append(items, child)
				}
				if children, ok := node["prefixItems"].([]any); ok {
					items = append(items, children...)
				}
			}
			next = items
		}
		if len(next) == 0 {
			return false
		}
		nodes = next
	}
	return len(nodes) > 0
}

func matchesTargets(targets, prefixes []string) bool {
	if prefixes == nil {
		return true
	}
	for _, target := range targets {
		for _, prefix := range prefixes {
			if target == prefix || strings.HasPrefix(target, prefix+".") {
				return true
			}
		}
	}
	return false
}

func resultFor(name string, targets []string, status string) ContractResult {
	return ContractResult{
		Name: &name, Targets: append([]string{}, targets...), Status: status, Issues: []Issue{},
	}
}

func reported(name string, targets []string, status, code, message string) ContractResult {
	result := resultFor(name, targets, status)
	result.Issues = append(result.Issues, Issue{Code: code, Path: []any{}, Message: message})
	return result
}

func samePath(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func contractRoot(projectRoot string) (string, error) {
	absolute, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("adopting project root must be a directory: %s", root)
	}
	for ancestor := root; ; ancestor = filepath.Dir(ancestor) {
		cli, cliErr := os.Stat(filepath.Join(ancestor, "src", "ptsip", "CLI.go"))
		config, configErr := os.Stat(filepath.Join(ancestor, "src", "ptsip", "schemas", "config.json"))
		if cliErr == nil && configErr == nil && !cli.IsDir() && !config.IsDir() {
			return "", errors.New("user contracts require an adopting project outside the PTSIP development checkout")
		}
		if filepath.Dir(ancestor) == ancestor {
			break
		}
	}
	for _, part := range []string{".ptsip", "contract", "user"} {
		root = filepath.Join(root, part)
		info, err := os.Lstat(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !samePath(root, resolved) {
			return "", fmt.Errorf("user contract path must stay in the adopting project: %s", root)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("user contract path must be a directory: %s", root)
		}
	}
	return root, nil
}

func createTemplate(filename string) error {
	content, err := templates.ReadFile("templates/" + filepath.Base(filename))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(filename), ".ptsip-user-contract-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Link(file.Name(), filename); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return nil
}

func readUserSchema(filename string) (any, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || !samePath(filename, resolved) {
		return nil, errors.New("contract must be a regular file under the user contract root")
	}
	return readJSON(filename, true)
}

func categoryFields(schema any) ([]string, error) {
	object, _ := schema.(map[string]any)
	roles, _ := object["x-ptsip-category-roles"].(map[string]any)
	allowed := map[string]bool{
		"CATEGORY_NAMESPACE": true, "NAMESPACE_DEFINITION": true,
		"OTHER_NAMESPACE_BOUNDARIES": true, "CATEGORY_RESPONSIBILITIES": true, "EXCLUDED_RESPONSIBILITIES": true,
	}
	problem := errors.New("x-ptsip-category-roles must bind CATEGORY_NAMESPACE and NAMESPACE_DEFINITION to distinct user-defined fields")
	seen := make(map[string]bool)
	for role, raw := range roles {
		field, ok := raw.(string)
		if !allowed[role] || !ok || field == "" || seen[field] {
			return nil, problem
		}
		seen[field] = true
	}
	fields := make([]string, 0, 2)
	for _, role := range []string{"CATEGORY_NAMESPACE", "NAMESPACE_DEFINITION"} {
		field, ok := roles[role].(string)
		if !ok || field == "" {
			return nil, problem
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func declaresConstraints(schema any) bool {
	if boolean, ok := schema.(bool); ok {
		return !boolean
	}
	keywords := strings.Fields("$ref $dynamicRef type enum const multipleOf maximum minimum exclusiveMaximum exclusiveMinimum maxLength minLength pattern format maxItems minItems uniqueItems contains maxContains minContains prefixItems items unevaluatedItems maxProperties minProperties required dependentRequired properties patternProperties additionalProperties propertyNames dependentSchemas unevaluatedProperties allOf anyOf oneOf not if then else")
	object, _ := schema.(map[string]any)
	for _, keyword := range keywords {
		if _, present := object[keyword]; present {
			return true
		}
	}
	return false
}

func (v *Validator) ValidateUserContracts(policy map[string]any, projectRoot string, bootstrap bool, targetPrefixes []string) ([]ContractResult, error) {
	if bootstrap && projectRoot == "" {
		return nil, errors.New("bootstrapping requires an explicit adopting project root")
	}
	root := ""
	if projectRoot != "" {
		var err error
		root, err = contractRoot(projectRoot)
		if err != nil {
			return nil, err
		}
	}
	required := make(map[string][]string)
	for name, targets := range standardBindings {
		for _, target := range targets {
			if len(targetValues(policy, target)) > 0 {
				required[name] = targets
				break
			}
		}
	}
	if bootstrap {
		for _, name := range sortedKeys(required) {
			filename := filepath.Join(root, name)
			if _, err := os.Lstat(filename); errors.Is(err, os.ErrNotExist) {
				if err := createTemplate(filename); err != nil {
					return nil, err
				}
			} else if err != nil {
				return nil, err
			}
		}
	}
	paths := make(map[string]string)
	if root != "" {
		entries, err := os.ReadDir(root)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".json") {
				paths[entry.Name()] = filepath.Join(root, entry.Name())
			}
		}
	}
	names := make(map[string]bool)
	for name := range required {
		names[name] = true
	}
	for name := range paths {
		if _, standard := standardBindings[name]; !standard {
			names[name] = true
		}
	}
	results := make(map[string]ContractResult)
	declared := make(map[string][]string)
	selected := make(map[string]selectedContract)
	documents := make(map[string]schemaDocument)
	documentNames := make(map[string]string)
	versionPresent := len(targetValues(policy, VersionTarget)) > 0
	for _, name := range sortedKeys(paths) {
		targets := required[name]
		schema, err := readUserSchema(paths[name])
		if err != nil {
			if names[name] {
				results[name] = reported(name, targets, "INVALID", "CONTRACT_SYNTAX", err.Error())
			}
			continue
		}
		object, _ := schema.(map[string]any)
		annotation, annotated := object["x-ptsip-target"]
		annotationText, stringAnnotation := annotation.(string)
		bindings, known := standardBindings[name]
		if known && annotated && (!stringAnnotation || !containsString(bindings, annotationText)) {
			if _, needed := required[name]; needed {
				results[name] = reported(name, targets, "UNBOUND", "CONTRACT_BINDING", "Standard contract target cannot be reassigned")
			}
			continue
		}
		if !known && stringAnnotation {
			if targetPrefixes != nil {
				declared[name] = []string{annotationText}
			}
			if v.targetExists(annotationText) && annotationText != "Policy_Rules.rules[].RA_id" {
				targets = []string{annotationText}
			}
		}
		if len(targets) == 1 && targets[0] == VersionTarget && !versionPresent {
			delete(names, name)
			targets = nil
		}
		if len(targets) > 0 && !matchesTargets(targets, targetPrefixes) {
			delete(names, name)
			targets = nil
		}
		if status, message := v.schemaProblem(schema); status != "" {
			if names[name] {
				code := "CONTRACT_SCHEMA"
				if status == "UNSUPPORTED" {
					code = "CONTRACT_DIALECT"
				}
				results[name] = reported(name, targets, status, code, message)
			}
			continue
		}
		document, err := describeSchema(fileURI(paths[name]), schema)
		if err != nil {
			if names[name] {
				results[name] = reported(name, targets, "INVALID", "CONTRACT_REFERENCE", err.Error())
			}
			continue
		}
		documents[document.URI] = document
		documentNames[document.URI] = name
		if len(targets) > 0 {
			selected[name] = selectedContract{document: document, targets: targets}
		} else if names[name] {
			results[name] = reported(name, nil, "UNBOUND", "CONTRACT_BINDING", "Declare an existing policy target with x-ptsip-target")
		}
	}
	owners := make(map[string][]string)
	for uri, document := range documents {
		for alias := range document.Aliases {
			owners[alias] = append(owners[alias], uri)
		}
	}
	ambiguous := make(map[string]string)
	for alias, claimed := range owners {
		if _, retrieval := documents[alias]; retrieval && (len(claimed) != 1 || claimed[0] != alias) {
			claimed = append(claimed, alias)
		}
		if len(claimed) > 1 {
			for _, uri := range claimed {
				ambiguous[uri] = alias
			}
		}
	}
	for uri, alias := range ambiguous {
		name := documentNames[uri]
		entry := selected[name]
		delete(documents, uri)
		delete(selected, name)
		if names[name] {
			results[name] = reported(name, entry.targets, "INVALID", "CONTRACT_REFERENCE", "Ambiguous contract resource ID: "+alias)
		}
	}
	compiler, err := compileDocuments(documents)
	if err != nil {
		return nil, err
	}
	for _, name := range sortedKeys(selected) {
		entry := selected[name]
		values := make([]targetValue, 0)
		for _, target := range entry.targets {
			values = append(values, targetValues(policy, target)...)
		}
		category := []string{}
		if len(entry.targets) == 1 && entry.targets[0] == CategoryTarget && len(values) > 0 {
			category, err = categoryFields(entry.document.Value)
			if err != nil {
				results[name] = reported(name, entry.targets, "INVALID", "CONTRACT_BINDING", err.Error())
				continue
			}
		}
		result := resultFor(name, entry.targets, "UNCONFIGURED")
		if len(values) > 0 {
			schema, err := compileClosure(compiler, entry.document, documents)
			if err != nil {
				code := "CONTRACT_REFERENCE"
				var schemaError *jsonschema.SchemaValidationError
				var patternError *jsonschema.InvalidRegexError
				if errors.As(err, &schemaError) || errors.As(err, &patternError) {
					code = "CONTRACT_SCHEMA"
				}
				results[name] = reported(name, entry.targets, "INVALID", code, err.Error())
				continue
			}
			for _, value := range values {
				result.Issues = append(result.Issues, validationIssues(schema, value.value, "CONTRACT_VALUE", value.path)...)
				for _, field := range category {
					fields, _ := value.value.(map[string]any)
					if _, present := fields[field]; !present {
						result.Issues = append(result.Issues, Issue{Code: "CATEGORY_ROLE", Path: value.path, Message: fmt.Sprintf("Missing category role field: %q", field)})
					}
				}
			}
			if len(category) > 0 || declaresConstraints(entry.document.Value) {
				result.Status = "VALID"
			}
			if len(result.Issues) > 0 {
				result.Status = "INVALID"
			}
		}
		results[name] = result
	}
	for name, targets := range required {
		if _, exists := paths[name]; !exists {
			results[name] = resultFor(name, targets, "UNCONFIGURED")
		}
	}
	reportedResults := make([]ContractResult, 0, len(results)+1)
	categoryBound := false
	for _, result := range results {
		if containsString(result.Targets, CategoryTarget) {
			categoryBound = true
		}
	}
	if len(targetValues(policy, CategoryTarget)) > 0 && !categoryBound && matchesTargets([]string{CategoryTarget}, targetPrefixes) {
		reportedResults = append(reportedResults, ContractResult{Name: nil, Targets: []string{CategoryTarget}, Status: "UNCONFIGURED", Issues: []Issue{}})
	}
	for _, name := range sortedKeys(results) {
		result := results[name]
		if len(result.Targets) == 0 && len(declared[name]) > 0 {
			result.Targets = declared[name]
		}
		if matchesTargets(result.Targets, targetPrefixes) {
			reportedResults = append(reportedResults, result)
		}
	}
	return reportedResults, nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
