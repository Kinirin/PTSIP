package binding

import (
	"crypto/sha256"
	"fmt"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const BindingRegistryPath = "developer/bindings/policy-plan-bindings.yaml"
const BindingSchemaPath = "developer/bindings/schemas/policy-plan-bindings.schema.json"

type BindingSnapshot struct {
	Path    string
	Payload Object
	Digest  string
}

func (r *Store) LoadBindingRegistry(required bool) (*BindingSnapshot, error) {
	path, err := r.Path(BindingRegistryPath)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && !required {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("BINDING_REGISTRY_MISSING: %w", err)
	}
	payload, err := r.Read(BindingRegistryPath)
	if err != nil {
		return nil, err
	}
	if err = r.Validate(BindingSchemaPath, payload); err != nil {
		return nil, fmt.Errorf("BINDING_REGISTRY_SCHEMA_INVALID: %w", err)
	}
	// Text snapshots use the same universal-newline contract as the prior registry reader.
	raw = []byte(strings.ReplaceAll(string(raw), "\r\n", "\n"))
	return &BindingSnapshot{BindingRegistryPath, payload, fmt.Sprintf("%x", sha256.Sum256(raw))}, nil
}

func bindingEntries(payload Object) ([]any, error) {
	rows, ok := payload["bindings"].([]any)
	if !ok {
		return nil, fmt.Errorf("BINDING_REGISTRY_INVALID_BINDINGS")
	}
	for _, raw := range rows {
		if Map(raw) == nil {
			return nil, fmt.Errorf("BINDING_REGISTRY_INVALID_ENTRY")
		}
	}
	return rows, nil
}

func (r *Store) ResolveBindings(query Object) (Object, error) {
	allowed := map[string]bool{"binding_id": true, "policy_ref": true, "resolved_plan_id": true, "plan_file_id": true, "plan_ref": true}
	nonempty := false
	for k, v := range query {
		if !allowed[k] {
			return nil, fmt.Errorf("BINDING_QUERY_FIELD_INVALID: %s", k)
		}
		if v != nil {
			if Text(v) == "" {
				return nil, fmt.Errorf("BINDING_QUERY_IDENTITY_INVALID: %s", k)
			}
			nonempty = true
		}
	}
	if !nonempty {
		return nil, fmt.Errorf("BINDING_QUERY_EMPTY")
	}
	snap, err := r.LoadBindingRegistry(true)
	if err != nil {
		return nil, err
	}
	rows, err := bindingEntries(snap.Payload)
	if err != nil {
		return nil, err
	}
	matches := []any{}
	for _, raw := range rows {
		item := Map(raw)
		match := true
		for k, v := range query {
			if v != nil && !reflect.DeepEqual(item[k], v) {
				match = false
			}
		}
		if match {
			matches = append(matches, item)
		}
	}
	out := Object{"status": "UNBOUND", "bindings": matches}
	for k := range allowed {
		out[k] = query[k]
	}
	if len(matches) > 0 {
		out["status"] = "BOUND"
	}
	return out, nil
}

func (r *Store) ValidateBindingRegistry(payload Object) []string {
	if err := r.Validate(BindingSchemaPath, payload); err != nil {
		return []string{err.Error()}
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return []string{err.Error()}
	}
	rows, _ := bindingEntries(payload)
	ids := map[string]bool{}
	relations := map[string]bool{}
	owners := map[string]string{}
	failures := []string{}
	for i, raw := range rows {
		row := Map(raw)
		id := Text(row["binding_id"])
		policy := Text(row["policy_ref"])
		prefix := fmt.Sprintf("bindings[%d]: ", i)
		if ids[id] {
			failures = append(failures, prefix+"duplicate binding_id "+id)
		}
		ids[id] = true
		policyRecord, policyErr := resolver.Policy(policy)
		if policyErr != nil {
			failures = append(failures, prefix+"unknown or noncanonical developer policy "+policy)
		} else {
			for _, section := range Strings(row["policy_sections"]) {
				if _, exists := Map(policyRecord["rules"])[section]; !exists {
					failures = append(failures, prefix+"unknown policy section "+policy+"#"+section)
				}
			}
		}
		if row["planning_state"] != "CREATED" {
			continue
		}
		resolved := Text(row["resolved_plan_id"])
		file := Text(row["plan_file_id"])
		relation := createdRelation(row)
		if relations[relation] {
			failures = append(failures, prefix+"duplicate created relation "+policy+" -> "+resolved)
		}
		relations[relation] = true
		if old := owners[file]; old != "" && old != resolved {
			failures = append(failures, prefix+"plan_file_id reused by different resolved_plan_id values")
		}
		owners[file] = resolved
		path, err := r.Path(Text(row["plan_ref"]))
		if err != nil {
			failures = append(failures, prefix+err.Error())
			continue
		}
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			failures = append(failures, prefix+"plan_ref does not exist: "+Text(row["plan_ref"]))
		}
	}
	return failures
}

// ReplaceBindingRegistry performs a compare-and-swap under an exclusive lock and
// installs a synced temporary file only after caller-owned validation succeeds.
func (r *Store) ReplaceBindingRegistry(payload Object, expectedDigest string, validator func(Object) []string) (*BindingSnapshot, error) {
	if validator == nil {
		return nil, fmt.Errorf("BINDING_REGISTRY_VALIDATOR_REQUIRED")
	}
	if failures := validator(payload); len(failures) > 0 {
		return nil, fmt.Errorf("BINDING_REGISTRY_VALIDATION_FAILED: %s", strings.Join(failures, "; "))
	}
	path, err := r.Path(BindingRegistryPath)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("BINDING_REGISTRY_LOCKED: %w", err)
	}
	lock.Close()
	defer os.Remove(path + ".lock")
	current, err := r.LoadBindingRegistry(false)
	if err != nil {
		return nil, err
	}
	digest := ""
	if current != nil {
		digest = current.Digest
	}
	if digest != expectedDigest {
		return nil, fmt.Errorf("BINDING_REGISTRY_STALE")
	}
	rendered, err := yaml.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err = WriteFileAtomically(path, rendered); err != nil {
		return nil, err
	}
	return r.LoadBindingRegistry(true)
}

// WriteFileAtomically installs synced bytes without changing domain authority.
func WriteFileAtomically(path string, raw []byte) error {
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

func (r *Store) ReconcileBindings(apply bool) (Object, error) {
	snap, err := r.LoadBindingRegistry(true)
	if err != nil {
		return nil, err
	}
	failures := r.ValidateBindingRegistry(snap.Payload)
	out := Object{"status": "INVALID", "changed": false, "applied": false, "binding_count": len(List(snap.Payload["bindings"])), "failures": failures}
	if len(failures) > 0 {
		return out, nil
	}
	rows := append([]any{}, List(snap.Payload["bindings"])...)
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := Text(Map(rows[i])["binding_id"]), Text(Map(rows[j])["binding_id"])
		a, _ := strconv.Atoi(strings.SplitN(strings.TrimPrefix(left, "PPB-"), "-", 2)[0])
		b, _ := strconv.Atoi(strings.SplitN(strings.TrimPrefix(right, "PPB-"), "-", 2)[0])
		if a == b {
			return left < right
		}
		return a < b
	})
	canonical := Object{"schema_version": snap.Payload["schema_version"], "registry_role": snap.Payload["registry_role"], "schema_ref": snap.Payload["schema_ref"], "bindings": rows}
	changed := !reflect.DeepEqual(canonical, snap.Payload)
	status := "CURRENT"
	if changed {
		status = "WOULD_RECONCILE"
	}
	if changed && apply {
		if _, err = r.ReplaceBindingRegistry(canonical, snap.Digest, r.ValidateBindingRegistry); err != nil {
			return nil, err
		}
		status = "RECONCILED"
	}
	out["status"] = status
	out["changed"] = changed
	out["applied"] = changed && apply
	return out, nil
}
