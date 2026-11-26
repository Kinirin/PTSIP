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

func retiredProductMember(name string) bool {
	if strings.HasPrefix(name, "ptsip/") && strings.HasSuffix(name, ".py") {
		return true
	}
	switch name {
	case "ptsip/specdata/project-profile-contracts.yaml",
		"ptsip/specdata/ptsip-profile-pp-1.01.schema.json",
		"ptsip/specdata/ptsip-profile-pp-1.02.schema.json",
		"ptsip/specdata/ptsip-profile-pp-2.00.candidate.schema.json",
		"ptsip/schemas/history/profile/ptsip-profile-pp-2.00.candidate.schema.json",
		"ptsip/specdata/ptsip-profile-pp-2.00.schema.json":
		return true
	}
	for _, root := range []string{"migration", "profiles", "repository", "policy_contracts", "compile_sealed_rbac", "specdata/compile-sealed-rbac", "schemas/storage/profile/history"} {
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
	const candidate = "ptsip/schemas/history/profile/ptsip-profile-pp-2.00.schema.json"
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

// stageRegisteredProductAssets uses the same committed source and registrations
// as the source archive. Wheel assembly supplies these assets even when the
// metadata frontend does not carry a copy.
func stageRegisteredProductAssets(source productSource, files wheelContents) error {
	stage := func(ref string) error {
		file, ok := source.Files[ref]
		if !ok || !strings.HasPrefix(ref, productRoot) || !productPath(ref) {
			return fmt.Errorf("missing or invalid registered Wheel asset %s", ref)
		}
		member := strings.TrimPrefix(ref, "src/")
		if base, exists := files[member]; exists && !bytes.Equal(base.data, file.Data) {
			if !bytes.Equal(bytes.ReplaceAll(base.data, []byte("\r\n"), []byte("\n")), file.Data) {
				return fmt.Errorf("base wheel differs from the canonical product asset %s", member)
			}
			// Use the tracked bytes so that pinned digests are stable on Windows.
			delete(files, member)
		}
		return files.setCanonical(member, file.Data)
	}
	if err := stageCommonWheelAssets(source, stage); err != nil {
		return err
	}
	return stageEngineWheelAssets(source, stage)
}

const commonManifestPath = productRoot + "common_contract/registry/manifest.json"
const commonProtocolPath = productRoot + "common_contract/cmd/ptsip-common-contract/protocol.json"
const commonTemplateRoot = productRoot + "common_contract/user_contracts/templates/"

type commonWheelRegistry struct {
	Format      string                `json:"format"`
	Revision    string                `json:"revision"`
	Contracts   []commonWheelResource `json:"contracts"`
	Diagnostics []commonWheelResource `json:"diagnostics"`
}

type commonWheelResource struct {
	Path   string `json:"resource"`
	Digest string `json:"sha256"`
}

func stageCommonWheelAssets(source productSource, stage func(string) error) error {
	var registry commonWheelRegistry
	if err := source.decode(commonManifestPath, &registry); err != nil {
		return err
	}
	if registry.Format != "ptsip-common-contract-registry/v1" || registry.Revision == "" ||
		len(registry.Contracts) == 0 || len(registry.Diagnostics) == 0 {
		return fmt.Errorf("incomplete Common Contract registry")
	}
	for _, row := range append(registry.Contracts, registry.Diagnostics...) {
		name := productRoot + row.Path
		if !sourcePath(row.Path) || !strings.HasSuffix(row.Path, ".schema.json") ||
			!(strings.HasPrefix(row.Path, "schemas/policy/") || strings.HasPrefix(row.Path, "schemas/plan/")) ||
			wheelDigest(source.Files[name].Data) != row.Digest {
			return fmt.Errorf("invalid Common Contract path or digest: %s", row.Path)
		}
		if err := stage(name); err != nil {
			return err
		}
	}
	for _, name := range []string{commonManifestPath, commonProtocolPath} {
		if err := stage(name); err != nil {
			return err
		}
	}
	templates := 0
	for name := range source.Files {
		if path.Dir(name) != strings.TrimSuffix(commonTemplateRoot, "/") || !strings.HasSuffix(name, ".json") {
			continue
		}
		if err := stage(name); err != nil {
			return err
		}
		templates++
	}
	if templates == 0 {
		return fmt.Errorf("Common Contract templates are missing")
	}
	return nil
}

func sourcePath(ref string) bool {
	return source.Relative(ref) && path.Clean(ref) == ref
}

func stageEngineWheelAssets(source productSource, stage func(string) error) error {
	index, _, pins, err := registeredEngineSources(source)
	if err != nil {
		return err
	}
	for _, ref := range []string{engineIndex, engineSnapshot, engineGraph, engineSupport, engineRelations, index.Transport, index.Federation.Static, index.Federation.Execution} {
		if err := stage(ref); err != nil {
			return err
		}
	}
	for _, ref := range pins {
		if err := stage(ref.Path); err != nil {
			return err
		}
	}
	for _, ref := range index.Schemas {
		if err := stage(ref); err != nil {
			return err
		}
	}
	for _, entry := range index.Entries {
		if err := stageEngineReferenceClosure(source, entry, stage); err != nil {
			return err
		}
	}
	var graph engineGraphDocument
	if err := source.decode(engineGraph, &graph); err != nil {
		return err
	}
	for _, edge := range graph.Edges {
		for _, ref := range []engineSourceRef{edge.ProviderEvidence, edge.ConsumerEvidence} {
			if err := stage(ref.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

func stageEngineReferenceClosure(source productSource, entry engineEntry, stage func(string) error) error {
	var manifest struct {
		Extensions   []engineSourceRef   `json:"extensions"`
		Dependencies []engineDeclaration `json:"dependencies"`
	}
	if err := source.decode(entry.Manifest, &manifest); err != nil {
		return err
	}
	for _, extension := range manifest.Extensions {
		if err := stage(extension.Path); err != nil {
			return err
		}
	}
	for _, dependency := range manifest.Dependencies {
		var evidence struct {
			Refs []engineSourceRef `json:"source_refs"`
		}
		if err := source.decode(dependency.Compatibility.Path, &evidence); err != nil {
			return err
		}
		for _, ref := range append(evidence.Refs, dependency.Compatibility) {
			if err := stage(ref.Path); err != nil {
				return err
			}
		}
	}
	return stageNativeOperationAssets(source, entry, stage)
}

func stageNativeOperationAssets(source productSource, entry engineEntry, stage func(string) error) error {
	var capability struct {
		Operations []struct {
			Verification struct {
				Input  engineSourceRef `json:"input_schema_ref"`
				Output string          `json:"output_schema_ref"`
			} `json:"verification_evidence"`
		} `json:"operations"`
	}
	if err := source.decode(entry.Contract, &capability); err != nil {
		return err
	}
	for _, operation := range capability.Operations {
		input := operation.Verification.Input
		output := operation.Verification.Output
		if input.Path == "" && output == "" {
			continue
		}
		inPath, _, _ := strings.Cut(input.Path, "#")
		outPath, _, _ := strings.Cut(output, "#")
		if !engineSourcePath(inPath) || !engineSourcePath(outPath) ||
			!strings.HasPrefix(inPath, path.Dir(entry.Manifest)+"/src/schemas/") ||
			!strings.HasPrefix(outPath, path.Dir(entry.Manifest)+"/src/schemas/") {
			return fmt.Errorf("native schema source escapes engine owner %s", entry.Kind)
		}
		var schema map[string]any
		if err := source.decode(inPath, &schema); err != nil {
			return err
		}
		if schema["$id"] != input.ID || wheelDigest(source.Files[inPath].Data) != input.SHA256 {
			return fmt.Errorf("native schema identity or digest differs: %s", inPath)
		}
		for _, ref := range []string{input.Path, output} {
			name, fragment, present := strings.Cut(ref, "#")
			var schema map[string]any
			if err := source.decode(name, &schema); err != nil {
				return err
			}
			if _, ok := resolver.Pointer(schema, fragment); !present || !ok {
				return fmt.Errorf("unresolved native schema anchor: %s", ref)
			}
			if err := stage(name); err != nil {
				return err
			}
		}
	}
	return nil
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
		if !registeredProfilePath(selector, binding) {
			return fmt.Errorf("profile registration escapes its schema root: %s", binding.Path)
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
			ID      string `json:"$id"`
			Version string `json:"version"`
		}
		if err := registry.decode(name, &schema); err != nil {
			return err
		}
		if schema.ID != binding.ID {
			return fmt.Errorf("central RBAC schema identity mismatch: %s", selector)
		}
		if schema.Version != binding.Version {
			return fmt.Errorf("central RBAC schema version mismatch: %s", selector)
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
