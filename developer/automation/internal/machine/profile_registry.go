package machine

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

func (r *Repository) PublicProfileCatalog() (Object, error) {
	storage, err := ppSnapshotStorage(PPGitSnapshot{Root: r.Root, Worktree: true})
	if err != nil {
		return nil, err
	}
	return r.Read(storage.Catalog)
}
func (r *Repository) ProfileContracts() (Object, error) {
	return r.Read("registry/project-profile-contracts.yaml")
}
func (r *Repository) CurrentProfileContract() (Object, error) {
	registry, err := r.ProfileContracts()
	if err != nil {
		return nil, err
	}
	selected := []Object{}
	for _, value := range List(registry["contracts"]) {
		item := Map(value)
		if item["version"] == registry["current"] {
			selected = append(selected, item)
		}
	}
	if len(selected) != 1 {
		return nil, Fail("CURRENT_PP_CONTRACT_AMBIGUOUS", "current contract must resolve exactly once")
	}
	return selected[0], nil
}
func (r *Repository) ProfileRegistryErrors() ([]string, error) {
	catalog, err := r.PublicProfileCatalog()
	if err != nil {
		return nil, err
	}
	registry, err := r.ProfileContracts()
	if err != nil {
		return nil, err
	}
	embedded, err := r.Read("src/ptsip/specdata/project-profile-contracts.yaml")
	if err != nil {
		return nil, err
	}
	errors := []string{}
	for path, value := range map[string]any{"developer/policy/schemas/public-profile-catalog.schema.json": catalog, "developer/policy/schemas/project-profile-contract-registry.schema.json": registry} {
		if err := r.Validate(path, value); err != nil {
			errors = append(errors, path+": "+err.Error())
		}
	}
	if !reflect.DeepEqual(registry, embedded) {
		errors = append(errors, "embedded Project Profile contract registry must equal canonical registry")
	}
	contracts := map[string]Object{}
	for _, raw := range List(registry["contracts"]) {
		item := Map(raw)
		id := Text(item["version"])
		if contracts[id] != nil {
			errors = append(errors, "project-profile contract registry versions must be unique")
		}
		contracts[id] = item
	}
	current := Text(registry["current"])
	selected := contracts[current]
	if selected == nil {
		errors = append(errors, "project-profile contract registry current identity must resolve exactly once")
	} else {
		if selected["lifecycle"] != "CURRENT" {
			errors = append(errors, "current project-profile contract must have lifecycle CURRENT")
		}
		baseline, err := r.Path(Text(selected["baseline"]))
		if err != nil {
			return nil, err
		}
		if info, err := os.Stat(baseline); err != nil || !info.IsDir() {
			errors = append(errors, "current project-profile baseline is missing")
		}
		schema, err := r.Read(Text(selected["schema"]))
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			if Map(Map(Map(Map(schema["properties"])["ptsip"])["properties"])["version"])["const"] != current {
				errors = append(errors, "current profile schema version const does not match registry")
			}
		}
	}
	transitions := map[string]bool{}
	for _, raw := range List(registry["transitions"]) {
		item := Map(raw)
		source := Text(item["from"])
		target := Text(item["to"])
		key := source + "|" + target
		if transitions[key] {
			errors = append(errors, "duplicate project-profile transition: "+key)
		}
		transitions[key] = true
		if contracts[source] == nil || contracts[target] == nil {
			errors = append(errors, "transition references an unregistered contract: "+key)
		}
	}
	profiles := List(catalog["profiles"])
	ids := map[string]bool{}
	resources := []string{}
	profileRoot := Text(catalog["root"])
	if profileRoot == "" {
		profileRoot = "profiles"
	}
	for _, raw := range profiles {
		entry := Map(raw)
		id := Text(entry["id"])
		resource := Text(entry["resource"])
		if ids[id] || Has(resources, resource) {
			errors = append(errors, "public profile catalog IDs/resources must be unique")
		}
		ids[id] = true
		resources = append(resources, resource)
		if contracts[Text(entry["contract"])] == nil {
			errors = append(errors, "public profile references unregistered contract: "+resource)
		}
		payload, err := r.Read(profileRoot + "/" + resource)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		if Map(payload["ptsip"])["version"] != entry["contract"] || Map(payload["responsibility_map"])["mode"] != entry["responsibility_mode"] {
			errors = append(errors, "public profile contract/responsibility mode does not match catalog: "+resource)
		}
		if entry["contract"] != current {
			errors = append(errors, "current catalog must bind every distributed profile to registry current")
		}
	}
	sort.Strings(resources)
	rootPath, err := r.Path(profileRoot)
	if err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(rootPath, "*.ptsip.yaml"))
	if err != nil {
		return nil, err
	}
	discovered := []string{}
	for _, path := range paths {
		discovered = append(discovered, filepath.Base(path))
	}
	sort.Strings(discovered)
	if !reflect.DeepEqual(discovered, resources) {
		errors = append(errors, "profile catalog must cover root *.ptsip.yaml exactly")
	}
	for id, contract := range contracts {
		if Text(contract["baseline"]) == "" {
			continue
		}
		baseline, err := r.Path(Text(contract["baseline"]))
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(baseline)
		if err != nil || !info.IsDir() {
			errors = append(errors, "profile baseline is missing: "+id)
			continue
		}
		baselinePaths, err := filepath.Glob(filepath.Join(baseline, "*.ptsip.yaml"))
		if err != nil {
			return nil, err
		}
		baselineNames := []string{}
		for _, path := range baselinePaths {
			baselineNames = append(baselineNames, filepath.Base(path))
			relative, _ := filepath.Rel(r.Root, path)
			payload, err := r.Read(filepath.ToSlash(relative))
			if err != nil {
				return nil, err
			}
			if Map(payload["ptsip"])["version"] != id {
				errors = append(errors, "baseline contract identity mismatch: "+filepath.ToSlash(relative))
			}
		}
		sort.Strings(baselineNames)
		if id == current && !reflect.DeepEqual(baselineNames, resources) {
			errors = append(errors, "current baseline must cover current public profiles exactly")
		}
	}
	sort.Strings(errors)
	return errors, nil
}
func init() {
	RegisterOperations("profile-registry", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "current":
			return r.CurrentProfileContract()
		case "catalog":
			return r.PublicProfileCatalog()
		case "contracts":
			return r.ProfileContracts()
		case "validate":
			errors, err := r.ProfileRegistryErrors()
			if err != nil {
				return nil, err
			}
			status := "PASS"
			if len(errors) > 0 {
				status = "FAIL"
			}
			return Object{"status": status, "errors": errors}, nil
		}
		return nil, fmt.Errorf("unregistered profile registry command: %s", command)
	})
}
