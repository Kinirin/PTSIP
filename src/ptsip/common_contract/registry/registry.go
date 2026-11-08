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
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

const Manifest = "registry/manifest.json"

var urn = regexp.MustCompile(`^urn:ptsip:(common_policy|common:plan):[a-z][a-z_]*:v[1-9][0-9]*$`)
var sha = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Entry struct {
	Field     string            `json:"field"`
	URN       string            `json:"urn"`
	Version   string            `json:"version"`
	Resource  string            `json:"resource"`
	SHA256    string            `json:"sha256"`
	Semantics map[string]string `json:"semantics"`
}

type catalog struct {
	Format    string  `json:"format"`
	Revision  string  `json:"revision"`
	Contracts []Entry `json:"contracts"`
}

type Registry struct {
	root     string
	entries  map[string]Entry
	Revision string
	Digest   string
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
	r := &Registry{root: root, entries: make(map[string]Entry), Revision: data.Revision, Digest: digest}
	resources := make(map[string]bool)
	fields := make(map[string]bool)
	for _, entry := range data.Contracts {
		if !urn.MatchString(entry.URN) || !strings.HasSuffix(entry.URN, ":"+entry.Version) ||
			entry.Version == "" || entry.Field == "" || !sha.MatchString(entry.SHA256) || entry.Semantics[""] == "" ||
			!strings.HasPrefix(entry.Resource, "common_contract/schemas/") || !fs.ValidPath(entry.Resource) ||
			!strings.HasSuffix(entry.Resource, ".schema.json") || path.Clean(entry.Resource) != entry.Resource {
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
	return r, nil
}

func (r *Registry) Schema(id string) (Entry, map[string]any, error) {
	entry, ok := r.entries[id]
	if !ok {
		return Entry{}, nil, fmt.Errorf("UNREGISTERED_URN: %s", id)
	}
	raw, err := readResource(r.root, strings.TrimPrefix(entry.Resource, "common_contract/"))
	if err != nil {
		return Entry{}, nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != entry.SHA256 {
		return Entry{}, nil, fmt.Errorf("STALE_SCHEMA: %s", entry.Resource)
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return Entry{}, nil, err
	}
	schema, ok := value.(map[string]any)
	if !ok || schema["$id"] != id {
		return Entry{}, nil, fmt.Errorf("SCHEMA_ID_MISMATCH: %s", id)
	}
	entry.Semantics = maps.Clone(entry.Semantics)
	return entry, schema, nil
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
