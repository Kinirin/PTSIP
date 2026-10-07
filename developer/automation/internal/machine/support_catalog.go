package machine

import (
	"path/filepath"
	"strings"
)

func (r *Repository) ProductContract(id string, requireActive bool) (Object, error) {
	identity, err := r.Read("src/vpms/contracts/identity-registry.json")
	if err != nil {
		return nil, err
	}
	resources := []string{"src/vpms/contracts/identity-registry.json"}
	for _, path := range Strings(identity["schema_resources"]) {
		if strings.Contains(path, "..") || strings.Contains(path, "\\") || filepath.IsAbs(path) {
			return nil, Fail("UNSAFE_REGISTERED_PATH", path)
		}
		resources = append(resources, "src/vpms/contracts/"+path)
	}
	catalog, err := r.Read("src/vpms/contracts/index.json")
	if err != nil {
		return nil, err
	}
	if err := r.ValidateResourceSet("src/vpms/contracts/schemas/catalog.schema.json", catalog, resources); err != nil {
		return nil, err
	}
	entry := Map(Map(catalog["contracts"])[id])
	if entry == nil {
		return nil, Fail("UNKNOWN_CONTRACT_ID", id)
	}
	relative := Text(entry["path"])
	if strings.Contains(relative, "..") || strings.Contains(relative, "\\") || filepath.IsAbs(relative) {
		return nil, Fail("UNSAFE_REGISTERED_PATH", relative)
	}
	payload, err := r.Read("src/vpms/contracts/" + relative)
	if err != nil {
		return nil, err
	}
	if err := r.ValidateResourceSet("src/vpms/contracts/schemas/product-contract.schema.json", payload, resources); err != nil {
		return nil, err
	}
	for _, field := range []string{"id", "status", "responsibility", "runtime_enabled"} {
		expected := entry[field]
		if field == "id" {
			expected = id
		}
		if payload[field] != expected {
			return nil, Fail("CONTRACT_INDEX_MISMATCH", field)
		}
	}
	if payload["contract_class"] != identity["contract_class"] {
		return nil, Fail("CONTRACT_CLASS_MISMATCH", id)
	}
	if requireActive && (catalog["capability"] != "ACTIVE" || payload["status"] != "ACTIVE" || payload["runtime_enabled"] != true) {
		return nil, Fail("CONTRACT_NOT_ACTIVE", id)
	}
	return payload, nil
}

// SupportField consumes an exact class-owned Root ref registered in the Go audit contract.
func (r *Repository) SupportField(name string) (any, error) {
	refs, err := r.Read("developer/policy/contracts/go-support-audit-refs.v1.json")
	if err != nil {
		return nil, err
	}
	reference := Map(Map(refs["fields"])[name])
	if reference == nil {
		return nil, Fail("SUPPORT_AUDIT_REF_UNREGISTERED", name)
	}
	id := Text(reference["policy_id"])
	path := Text(reference["policy_path"])
	index, err := r.Read("src/policy/index.yaml")
	if err != nil {
		return nil, err
	}
	selected := []Object{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		if entry["id"] == id {
			selected = append(selected, entry)
		}
	}
	if len(selected) != 1 || "src/policy/"+Text(selected[0]["path"]) != path {
		return nil, Fail("SUPPORT_ROOT_METADATA_MISMATCH", id)
	}
	policy, err := r.Read(path)
	if err != nil {
		return nil, err
	}
	if policy["policy_class"] != "PTSIP_SUPPORT_FEATURE" || Map(policy["policy"])["id"] != id || Map(policy["policy"])["status"] != selected[0]["status"] {
		return nil, Fail("SUPPORT_ROOT_METADATA_MISMATCH", id)
	}
	value, exists := Map(policy["authority_semantics"])[Text(reference["section"])]
	if !exists {
		return nil, Fail("SUPPORT_ROOT_SECTION_MISSING", name)
	}
	for _, field := range Strings(reference["value_path"]) {
		mapped := Map(value)
		var ok bool
		value, ok = mapped[field]
		if !ok {
			return nil, Fail("SUPPORT_ROOT_FIELD_MISSING", field)
		}
	}
	return value, nil
}
