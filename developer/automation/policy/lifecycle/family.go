package lifecycle

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var familyPolicyID = regexp.MustCompile(`^(MPD|SFP)-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD)-([0-9]{4})$`)

func FamilyEntry(r Repository, policyClass, family string) (Object, error) {
	registry, err := r.Read("developer/policy/registries/root-family-entry-registry.json")
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/root-family-entry-registry.schema.json", registry); err != nil {
		return nil, err
	}
	route := Map(Map(registry["policy_classes"])[policyClass])
	if route == nil {
		return nil, fmt.Errorf("unregistered policy class: %s", policyClass)
	}
	recognized := false
	for _, item := range List(route["recognized_root_families"]) {
		if item == family {
			recognized = true
		}
	}
	if !recognized {
		return nil, fmt.Errorf("unregistered Root Family %q for %s", family, policyClass)
	}
	plane := Text(route["canonical_root"])
	index, err := r.Read(plane + "/index.yaml")
	if err != nil {
		return nil, err
	}
	if policyClass == "PTSIP_SUPPORT_FEATURE" {
		if index["policy_class"] != policyClass {
			return nil, fmt.Errorf("Support index class mismatch")
		}
		if err := r.Validate("src/policy/schemas/ptsip-support-feature-policy-index.schema.json", index); err != nil {
			return nil, err
		}
	} else if err := r.Validate("developer/policy/schemas/developer-policy-catalog.schema.json", index); err != nil {
		return nil, err
	}
	selected := []any{}
	maximum := 0
	states := map[string]bool{}
	seen := map[string]bool{}
	for _, value := range List(index["policies"]) {
		entry := Map(value)
		id := Text(entry["id"])
		if seen[id] {
			return nil, fmt.Errorf("duplicate indexed identity %s", id)
		}
		seen[id] = true
		match := familyPolicyID.FindStringSubmatch(id)
		if match == nil || match[1] != route["id_prefix"] {
			continue
		}
		if policyClass == DeveloperClass && entry["policy_class"] != policyClass {
			return nil, fmt.Errorf("%s: class route mismatch", id)
		}
		expected := plane + "/" + match[2] + "/" + id + ".yaml"
		indexedPath := Text(entry["path"])
		if policyClass == "PTSIP_SUPPORT_FEATURE" {
			indexedPath = plane + "/" + indexedPath
		}
		if indexedPath != expected {
			return nil, fmt.Errorf("%s: Root path mismatch", id)
		}
		record, err := r.Read(expected)
		if err != nil {
			return nil, err
		}
		identity := Map(record["policy"])
		if record["policy_class"] != policyClass || record["responsibility_family"] != match[2] || identity["id"] != id || identity["status"] != entry["status"] {
			return nil, fmt.Errorf("%s: Root metadata mismatch", id)
		}
		if err := r.Validate(Text(route["schema_ref"]), record); err != nil {
			return nil, err
		}
		if match[2] != family {
			continue
		}
		number, _ := strconv.Atoi(match[3])
		if number > maximum {
			maximum = number
		}
		states[Text(identity["status"])] = true
		selected = append(selected, Object{"policy_id": id, "canonical_path": expected, "status": identity["status"]})
	}
	// A new allocation cannot ignore a discovered but unregistered Root identity.
	for _, rawFamily := range List(route["recognized_root_families"]) {
		directory, err := r.Path(plane + "/" + Text(rawFamily))
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		if err := filepath.WalkDir(directory, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				return nil
			}
			id := strings.TrimSuffix(entry.Name(), ".yaml")
			match := familyPolicyID.FindStringSubmatch(id)
			if match == nil || match[1] != route["id_prefix"] {
				return nil
			}
			if !seen[id] {
				return fmt.Errorf("unregistered Root policy identity: %s", id)
			}
			expected, err := r.Path(plane + "/" + match[2] + "/" + id + ".yaml")
			if err != nil {
				return err
			}
			if expected != name {
				return fmt.Errorf("duplicate or misplaced Root policy identity: %s", id)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	if maximum == 9999 {
		return nil, fmt.Errorf("policy identity space exhausted")
	}
	sort.Slice(selected, func(i, j int) bool { return Text(Map(selected[i])["policy_id"]) < Text(Map(selected[j])["policy_id"]) })
	state := Text(route["unmaterialized_slot_default"])
	for _, candidate := range []string{"ACTIVE", "APPROVED", "DRAFT", "DEPRECATED", "SUPERSEDED", "RETIRED"} {
		if states[candidate] {
			state = candidate
			break
		}
	}
	allocated := fmt.Sprintf("%s-%s-%04d", Text(route["id_prefix"]), family, maximum+1)
	return Object{"status": "READY", "authority_identity": Object{"policy_class": policyClass, "responsibility_family": family},
		"semantic_inheritance": "FORBIDDEN", "family_state_before_materialization": state,
		"registered_policies": selected, "allocated_policy_id": allocated,
		"canonical_path": plane + "/" + family + "/" + allocated + ".yaml", "schema_ref": route["schema_ref"]}, nil
}
