// Package registry opens the versioned catalog shipped beside the Go executable.
package registry

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

const Manifest = "registry/manifest.json"
const PlanDocumentURN = "urn:ptsip:contract:plan:public-structure"
const PolicyDocumentURN = "urn:ptsip:contract:public-structure-policy:v4"
const PolicyIdentityURN = "urn:ptsip:contract:policy:identity:v4"
const PlanLifecycleURN = "urn:ptsip:contract:plan:lifecycle"

type PlanDependency struct {
	Field    string
	URN      string
	Resource string
}

func PlanDependencies() []PlanDependency {
	return []PlanDependency{
		{"Plan_Identity", "urn:ptsip:contract:plan:identity", "schemas/plan/plan-identity.schema.json"},
		{"Plan_Lifecycle", PlanLifecycleURN, "schemas/plan/plan-lifecycle.schema.json"},
		{"Plan_Unit", "urn:ptsip:contract:plan:unit", "schemas/plan/plan-unit.schema.json"},
	}
}

var urn = regexp.MustCompile(`^urn:ptsip:contract:(public-structure-policy|policy:[a-z][a-z_-]*):v[1-9][0-9]*$`)
var sha = regexp.MustCompile(`^[a-f0-9]{64}$`)
var schemaVersion = regexp.MustCompile(`^SCH\.[1-9][0-9]*\.[0-9]{2}$`)

type Entry struct {
	Field     string            `json:"field"`
	URN       string            `json:"urn"`
	Version   string            `json:"version"`
	Resource  string            `json:"resource"`
	SHA256    string            `json:"sha256"`
	Semantics map[string]string `json:"semantics"`
}

type Diagnostic struct {
	Field    string `json:"field"`
	Resource string `json:"resource"`
	SHA256   string `json:"sha256"`
}
type catalog struct {
	Format      string       `json:"format"`
	Revision    string       `json:"revision"`
	Contracts   []Entry      `json:"contracts"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type Registry struct {
	root        string
	entries     map[string]Entry
	diagnostics map[string]Diagnostic
	Revision    string
	Digest      string
}

func Open(root, expectedDigest string) (*Registry, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	raw, err := readResource(root, Manifest)
	if err != nil {
		return nil, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	if expectedDigest != "" && digest != expectedDigest {
		return nil, fmt.Errorf("STALE_REGISTRY: %s", Manifest)
	}
	if _, err := jsondoc.Decode(raw); err != nil {
		return nil, err
	}
	var data catalog
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if data.Format != "ptsip-common-contract-registry/v1" || data.Revision == "" || len(data.Contracts) == 0 {
		return nil, fmt.Errorf("INVALID_REGISTRY")
	}
	r := &Registry{root: root, entries: make(map[string]Entry), diagnostics: make(map[string]Diagnostic), Revision: data.Revision, Digest: digest}
	resources := make(map[string]bool)
	fields := make(map[string]bool)
	for _, entry := range data.Contracts {
		planRoot := entry.URN == PlanDocumentURN && entry.Field == "Plan_Document" &&
			entry.Resource == "schemas/plan/public-structure-plan.schema.json" && len(entry.Semantics) == 0
		versionMatches := strings.HasSuffix(entry.URN, ":"+entry.Version)
		if entry.URN == PolicyDocumentURN {
			versionMatches = schemaVersion.MatchString(entry.Version)
		}
		policy := urn.MatchString(entry.URN) && versionMatches &&
			entry.Semantics[""] != "" && schemaPath(entry.Resource, "schemas/policy/")
		if entry.Version == "" || entry.Field == "" || !sha.MatchString(entry.SHA256) || !(planRoot || policy) {
			return nil, fmt.Errorf("INVALID_REGISTRY_ENTRY: %s", entry.URN)
		}
		fieldVersion := entry.Field + "/" + entry.Version
		resourceKey := strings.ToLower(entry.Resource)
		if _, exists := r.entries[entry.URN]; exists || resources[resourceKey] || fields[fieldVersion] {
			return nil, fmt.Errorf("AMBIGUOUS_REGISTRY_ENTRY: %s", entry.URN)
		}
		r.entries[entry.URN] = entry
		resources[resourceKey] = true
		fields[fieldVersion] = true
	}
	seen := map[string]bool{}
	for _, d := range data.Diagnostics {
		if d.Field == "" || !sha.MatchString(d.SHA256) ||
			!(schemaPath(d.Resource, "schemas/plan/") || schemaPath(d.Resource, "schemas/policy/")) {
			return nil, fmt.Errorf("INVALID_DIAGNOSTIC_RESOURCE: %s", d.Resource)
		}
		if _, exists := r.diagnostics[d.Field]; exists || seen[strings.ToLower(d.Resource)] {
			return nil, fmt.Errorf("AMBIGUOUS_DIAGNOSTIC_RESOURCE: %s", d.Field)
		}
		r.diagnostics[d.Field] = d
		seen[strings.ToLower(d.Resource)] = true
	}
	return r, nil
}

func schemaPath(resource, prefix string) bool {
	return strings.HasPrefix(resource, prefix) && fs.ValidPath(resource) &&
		path.Clean(resource) == resource && strings.HasSuffix(resource, ".schema.json") &&
		!strings.ContainsAny(resource, "\\:")
}
func (r *Registry) DiagnosticFields() []string {
	result := make([]string, 0, len(r.diagnostics))
	for field := range r.diagnostics {
		result = append(result, field)
	}
	sort.Strings(result)
	return result
}

// InspectDiagnostic reads a declared package schema path but never grants
// official schema registration or a versioned contract identity.
func (r *Registry) InspectDiagnostic(field string) (Diagnostic, map[string]any, error) {
	entry, ok := r.diagnostics[field]
	if !ok {
		return Diagnostic{}, nil, fmt.Errorf("UNREGISTERED_DIAGNOSTIC_FIELD: %s", field)
	}
	raw, err := r.readSchema(entry.Resource, entry.SHA256)
	if err != nil {
		return entry, nil, err
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return entry, nil, fmt.Errorf("SCHEMA_JSON_INVALID: %w", err)
	}
	doc, ok := value.(map[string]any)
	if !ok {
		return entry, nil, fmt.Errorf("SCHEMA_ROOT_NOT_OBJECT: %s", entry.Resource)
	}
	return entry, doc, nil
}
func (r *Registry) Schema(id string) (Entry, map[string]any, error) {
	entry, ok := r.entries[id]
	if !ok {
		return Entry{}, nil, fmt.Errorf("UNREGISTERED_URN: %s", id)
	}
	raw, err := r.readSchema(entry.Resource, entry.SHA256)
	if err != nil {
		return Entry{}, nil, err
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return Entry{}, nil, err
	}
	schema, ok := value.(map[string]any)
	if !ok || schema["$id"] != id {
		return Entry{}, nil, fmt.Errorf("SCHEMA_ID_MISMATCH: %s", id)
	}
	if id == PolicyDocumentURN && schema["version"] != entry.Version {
		return Entry{}, nil, fmt.Errorf("SCHEMA_VERSION_MISMATCH: %s", id)
	}
	if id == PlanDocumentURN {
		properties, _ := schema["properties"].(map[string]any)
		version, _ := properties["SchemaVersion"].(map[string]any)
		if schema["version"] != entry.Version || version["const"] != entry.Version {
			return Entry{}, nil, fmt.Errorf("SCHEMA_VERSION_MISMATCH: %s", id)
		}
	}
	entry.Semantics = maps.Clone(entry.Semantics)
	return entry, schema, nil
}

// Reference resolves the three Plan child contracts only through the public
// Plan root. A child has its own identity and version but no public entry.
func (r *Registry) Reference(root, id string) (map[string]any, error) {
	if root == PlanDocumentURN {
		for _, dependency := range PlanDependencies() {
			if id != dependency.URN {
				continue
			}
			entry, schema, err := r.InspectDiagnostic(dependency.Field)
			if err != nil {
				return nil, err
			}
			version, _ := schema["version"].(string)
			if entry.Resource != dependency.Resource || schema["$id"] != id || version == "" {
				return nil, fmt.Errorf("PLAN_SCHEMA_DEPENDENCY_MISMATCH: %s", id)
			}
			return schema, nil
		}
		return nil, fmt.Errorf("UNREGISTERED_PLAN_SCHEMA_DEPENDENCY: %s", id)
	}
	_, schema, err := r.Schema(id)
	if err == nil && root == PolicyDocumentURN && id == PolicyIdentityURN &&
		!PolicyIdentityVersionMatches(schema, r.entries[root].Version) {
		return nil, fmt.Errorf("POLICY_IDENTITY_SCHEMA_VERSION_MISMATCH: %s", id)
	}
	return schema, err
}

func PolicyIdentityVersionMatches(schema map[string]any, version string) bool {
	properties, _ := schema["properties"].(map[string]any)
	versionField, _ := properties["SchemaVersion"].(map[string]any)
	return versionField["const"] == version
}

func (r *Registry) readSchema(resource, expectedDigest string) ([]byte, error) {
	raw, err := readResource(filepath.Dir(r.root), resource)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != expectedDigest {
		return nil, fmt.Errorf("STALE_SCHEMA: %s", resource)
	}
	return raw, nil
}

func readResource(root, relative string) ([]byte, error) {
	if !fs.ValidPath(relative) || strings.ContainsAny(relative, "\\:") {
		return nil, fmt.Errorf("INVALID_RESOURCE_PATH: %s", relative)
	}
	current := root
	var info fs.FileInfo
	for _, part := range strings.Split(relative, "/") {
		current = filepath.Join(current, part)
		var err error
		info, err = os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("RESOURCE_UNAVAILABLE: %s", relative)
		}
	}
	if !info.Mode().IsRegular() || info.Size() > 16*1024*1024 {
		return nil, fmt.Errorf("INVALID_RESOURCE_FILE: %s", relative)
	}
	resolved, err := filepath.EvalSymlinks(current)
	if err != nil || !strings.EqualFold(resolved, current) {
		return nil, fmt.Errorf("RESOURCE_UNAVAILABLE: %s", relative)
	}
	return os.ReadFile(resolved)
}
