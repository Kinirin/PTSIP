package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

func retiredWheelMember(name string) bool {
	if path.Dir(name) == "ptsip" && strings.HasSuffix(name, ".py") {
		return true
	}
	switch name {
	case "ptsip/ptsip_specification/spec_identity.py", "ptsip/ptsip_specification/specification_binding.py",
		"ptsip/specdata/project-profile-contracts.yaml":
		return true
	}
	for _, root := range []string{"migration", "profiles", "repository", "policy_contracts", "compile_sealed_rbac", "specdata/compile-sealed-rbac"} {
		if strings.HasPrefix(name, "ptsip/"+root+"/") {
			return true
		}
	}
	return false
}

// stageWheelAssets adds the inputs owned by the product source catalogs. The
// setuptools wheel provides distribution metadata and remaining Python packages.
func stageWheelAssets(root string, files wheelContents) error {
	version, err := readAsset(root, "src/VERSION")
	if err != nil {
		return err
	}
	if err := files.setCanonical("ptsip/VERSION", version); err != nil {
		return err
	}
	if err := stageStorageSchemas(root, files); err != nil {
		return err
	}
	const candidate = "ptsip/specdata/ptsip-profile-pp-2.00.candidate.schema.json"
	candidateBytes, err := readAsset(root, "src/"+candidate)
	if err != nil {
		return err
	}
	doc, err := jsondoc.Decode(candidateBytes)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(source.String(source.Object(doc)["$comment"]), "DRAFT_NON_NORMATIVE:") {
		return fmt.Errorf("packaged pp.2.00 profile schema candidate must remain non-normative")
	}
	if err := files.setCanonical(candidate, candidateBytes); err != nil {
		return err
	}
	return stageSupportPolicy(root, files)
}

func readAsset(root, relative string) ([]byte, error) {
	if !source.Relative(relative) {
		return nil, fmt.Errorf("invalid product asset reference %q", relative)
	}
	file := filepath.Join(root, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(file)
	if err != nil {
		return nil, err
	}
	if resolved != file {
		return nil, fmt.Errorf("product asset is an alias: %s", relative)
	}
	info, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("product asset must be a regular file: %s", relative)
	}
	return os.ReadFile(file)
}

func (files wheelContents) setCanonical(name string, data []byte) error {
	if staged, exists := files[name]; exists {
		if !bytes.Equal(staged.data, data) {
			return fmt.Errorf("base wheel differs from the canonical product asset %s", name)
		}
		return nil
	}
	return files.add(name, data, false)
}

func stageStorageSchemas(root string, files wheelContents) error {
	const config = schemaRoot + "config.json"
	raw, err := readAsset(root, config)
	if err != nil {
		return err
	}
	registry := productSource{Files: map[string]sourceFile{config: {Data: raw}}}
	var catalog struct {
		Storage struct {
			Profile map[string]profileRegistration `json:"profile"`
			RBAC    map[string]profileRegistration `json:"Responsibility-Authority"`
		} `json:"storage"`
	}
	if err := registry.decode(config, &catalog); err != nil {
		return err
	}
	for selector, binding := range catalog.Storage.Profile {
		if !strings.HasPrefix(binding.Path, "storage/profile/") || !source.Relative(binding.Path) {
			return fmt.Errorf("profile registration escapes its schema root: %s", binding.Path)
		}
		if selector != "ptsip.project-profile" && binding.Lifecycle != "DEPRECATED" {
			return fmt.Errorf("historical Profile %s must be registered as DEPRECATED", selector)
		}
		name := schemaRoot + binding.Path
		data, err := readAsset(root, name)
		if err != nil {
			return err
		}
		registry.Files[name] = sourceFile{Data: data}
		var schema map[string]any
		if err := registry.decode(name, &schema); err != nil {
			return err
		}
		version, _ := resolver.Pointer(schema, "/properties/ptsip/properties/version/const")
		if version != binding.Version {
			return fmt.Errorf("profile schema %s differs from its registered version", name)
		}
	}
	if _, _, err := inspectProfiles(registry); err != nil {
		return err
	}
	if catalog.Storage.Profile["ptsip.project-profile"].ID != "urn:ptsip:storage:contract:profile:official" {
		return fmt.Errorf("current Profile must use the approved official schema identity")
	}
	if len(catalog.Storage.RBAC) != 3 {
		return fmt.Errorf("central RBAC catalog must register RA_list, RAC_list and AEC_list")
	}
	ids := map[string]bool{}
	for _, selector := range []string{"RA_list", "RAC_list", "AEC_list"} {
		binding := catalog.Storage.RBAC[selector]
		name := schemaRoot + binding.Path
		if !source.Relative(binding.Path) || path.Dir(binding.Path) != "storage/rbac" ||
			!strings.HasSuffix(binding.Path, ".schema.json") || registry.Files[name].Data != nil ||
			binding.ID == "" || ids[binding.ID] || binding.Version == "" {
			return fmt.Errorf("invalid or repeated registered RBAC schema path or identity: %s", selector)
		}
		data, err := readAsset(root, name)
		if err != nil {
			return err
		}
		registry.Files[name] = sourceFile{Data: data}
		var schema struct {
			ID string `json:"$id"`
		}
		if err := registry.decode(name, &schema); err != nil {
			return err
		}
		if schema.ID != binding.ID {
			return fmt.Errorf("central RBAC schema identity mismatch: %s", selector)
		}
		ids[binding.ID] = true
	}
	for name, file := range registry.Files {
		if err := files.setCanonical(strings.TrimPrefix(name, "src/"), file.Data); err != nil {
			return err
		}
	}
	return nil
}

func stageSupportPolicy(root string, files wheelContents) error {
	const policy = "src/policy/"
	index, err := readAsset(root, policy+"index.yaml")
	if err != nil {
		return err
	}
	if err := files.setCanonical("ptsip/support/policy/index.yaml", index); err != nil {
		return err
	}
	records := 0
	for _, directory := range []string{"schemas", "registries", "."} {
		start := filepath.Join(root, filepath.FromSlash(policy), directory)
		err := filepath.WalkDir(start, func(file string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			relative, err := filepath.Rel(filepath.Join(root, filepath.FromSlash(policy)), file)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if directory == "." && !(strings.HasPrefix(entry.Name(), "SFP-") && strings.HasSuffix(entry.Name(), ".yaml")) {
				return nil
			}
			raw, err := readAsset(root, policy+relative)
			if err != nil {
				return err
			}
			if directory != "." {
				return files.setCanonical("ptsip/support/"+relative, raw)
			}
			records++
			doc, err := source.Decode(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}))
			if err == nil {
				var projected source.Document
				projected, err = source.ProjectSupport(doc, relative)
				if err == nil && projected != nil {
					raw, err = json.MarshalIndent(projected, "", "  ")
					raw = append(raw, '\n')
				}
			}
			if err != nil {
				return fmt.Errorf("Support Policy source %s: %w", relative, err)
			}
			return files.setCanonical("ptsip/support/policy/"+relative, raw)
		})
		if err != nil {
			return err
		}
	}
	if records == 0 {
		return fmt.Errorf("canonical Support Policy source contains no SFP records")
	}
	return nil
}
