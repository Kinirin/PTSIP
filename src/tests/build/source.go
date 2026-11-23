package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

const (
	releaseRoot                = ".ptsip/contracts/release/"
	releaseIndex               = releaseRoot + "index.json"
	buildVerificationReference = releaseIndex + "#build_verification"
)

type sourceRules struct {
	ProductPrefix          string   `json:"product_prefix"`
	SchemaConfig           string   `json:"schema_config"`
	SchemaGroups           []string `json:"schema_groups"`
	EngineIndex            string   `json:"engine_index"`
	EnginePins             string   `json:"engine_snapshot"`
	EngineRuntimeResources []string `json:"engine_runtime_resources"`
	CommonRegistry         string   `json:"common_contract_registry"`
}

type sdistRules struct {
	SourceDescriptor string   `json:"source_descriptor"`
	ExcludedPrefixes []string `json:"excluded_prefixes"`
}

type wheelRules struct {
	EngineOriginSuffix   string   `json:"engine_origin_suffix"`
	CommonContractOrigin string   `json:"common_contract_origin"`
	GoEntrypoint         string   `json:"go_entrypoint"`
	ExcludedPrefixes     []string `json:"excluded_prefixes"`
}

type buildContract struct {
	Version               string      `json:"schema_version"`
	Root                  string      `json:"verifier_root"`
	SourceBinding         string      `json:"source_binding"`
	PublicationAuthorized *bool       `json:"publication_authorized"`
	Source                sourceRules `json:"source"`
	Prebuild              struct {
		Frontend      string   `json:"frontend"`
		RequiredFiles []string `json:"required_product_inputs"`
	} `json:"prebuild"`
	SDist        sdistRules `json:"sdist"`
	Wheel        wheelRules `json:"wheel"`
	Installation struct {
		Environment string `json:"environment"`
		Package     string `json:"smoke_package"`
		Activation  *bool  `json:"runtime_activation_authorized"`
	} `json:"installation"`
}

type gitSource struct {
	root, revision string
}

func gitOutput(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", root, "--no-replace-objects"}, args...)...)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func openSource(root string) (gitSource, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return gitSource{}, err
	}
	top, err := gitOutput(absolute, "rev-parse", "--show-toplevel")
	if err != nil {
		return gitSource{}, err
	}
	relative, err := filepath.Rel(absolute, strings.TrimSpace(string(top)))
	if err != nil || relative != "." {
		return gitSource{}, fmt.Errorf("verify-build must select the exact Git checkout root")
	}
	head, err := gitOutput(absolute, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return gitSource{}, err
	}
	return gitSource{absolute, strings.TrimSpace(string(head))}, nil
}

func safePath(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.HasPrefix(name, "../") &&
		!path.IsAbs(name) && !strings.ContainsAny(name, "\\:\x00\r\n") && path.Clean(name) == name
}

func (source gitSource) read(name string) ([]byte, error) {
	if !safePath(name) {
		return nil, fmt.Errorf("invalid release source reference %q", name)
	}
	meta, err := gitOutput(source.root, "ls-tree", "--full-tree", source.revision, "--", name)
	if err != nil {
		return nil, err
	}
	entry, member, ok := strings.Cut(strings.TrimSuffix(string(meta), "\n"), "\t")
	if !ok || member != name || !(strings.HasPrefix(entry, "100644 blob ") || strings.HasPrefix(entry, "100755 blob ")) {
		return nil, fmt.Errorf("release source is missing or not a regular Git file: %s", name)
	}
	return gitOutput(source.root, "show", source.revision+":"+name)
}

func decode(raw []byte, target any) error {
	if _, err := jsondoc.Decode(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func (source gitSource) decode(name string, target any) ([]byte, error) {
	raw, err := source.read(name)
	if err != nil {
		return nil, err
	}
	if err := decode(raw, target); err != nil {
		return nil, fmt.Errorf("invalid release JSON %s: %w", name, err)
	}
	return raw, nil
}

func sha(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

func digestReader(reader io.Reader) (string, error) {
	digest := sha256.New()
	if _, err := io.Copy(digest, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func loadContract(source gitSource) (buildContract, string, string, error) {
	var index struct {
		Version string          `json:"schema_version"`
		Build   json.RawMessage `json:"build_verification"`
	}
	var contract buildContract
	if _, err := source.decode(releaseIndex, &index); err != nil {
		return contract, "", "", err
	}
	if index.Version != "ptsip-release-contract-index/v2" || len(index.Build) == 0 {
		return contract, "", "", fmt.Errorf("registered build verification contract is missing from %s", releaseIndex)
	}
	if err := decode(index.Build, &contract); err != nil {
		return contract, "", "", fmt.Errorf("invalid owner contract %s: %w", buildVerificationReference, err)
	}
	if contract.Version != "ptsip-build-verification/v1" || contract.Root != "src/tests/build" || contract.SourceBinding != "EXACT_GIT_HEAD" ||
		contract.PublicationAuthorized == nil || *contract.PublicationAuthorized || contract.Installation.Activation == nil || *contract.Installation.Activation ||
		contract.Installation.Environment != "TEMPORARY_VENV" || contract.Installation.Package != "ptsip" || !safePath(contract.Source.ProductPrefix+"schema") ||
		!strings.HasSuffix(contract.Source.ProductPrefix, "/") || len(contract.Source.SchemaGroups) == 0 || !safePath(contract.Prebuild.Frontend) ||
		!safePath(contract.SDist.SourceDescriptor) || !safePath(contract.Wheel.CommonContractOrigin) || path.Base(contract.Wheel.GoEntrypoint) != contract.Wheel.GoEntrypoint ||
		!safePath(contract.Wheel.GoEntrypoint) || !strings.HasPrefix(contract.Wheel.EngineOriginSuffix, ".dist-info/") {
		return contract, "", "", fmt.Errorf("invalid or unsafe build-verification owner contract %s", buildVerificationReference)
	}
	return contract, buildVerificationReference, sha(index.Build), nil
}

type engineEntry struct {
	Kind    string `json:"kind"`
	ID      string `json:"engine_id"`
	Version string `json:"version"`
}

type registeredAssets struct {
	productPrefix        string
	digests              map[string]string
	exactDigestPaths     map[string]bool
	engines              map[string]engineEntry
	activeProfile        string
	activeProfilePath    string
	commonRegistryDigest string
}

func loadAssets(source gitSource, rules sourceRules) (registeredAssets, error) {
	assets := registeredAssets{productPrefix: rules.ProductPrefix, digests: map[string]string{}, engines: map[string]engineEntry{}}
	read := func(name string, target any) ([]byte, error) {
		if !strings.HasPrefix(name, rules.ProductPrefix) {
			return nil, fmt.Errorf("registered asset outside product root: %s", name)
		}
		raw, err := source.decode(name, target)
		if err == nil {
			assets.digests[name] = canonicalJSONDigest(raw)
		}
		return raw, err
	}
	var config struct {
		Storage map[string]map[string]struct {
			ID        string `json:"$id"`
			Path      string `json:"path"`
			Digest    string `json:"schema_sha256"`
			Version   string `json:"version"`
			Lifecycle string `json:"registered_lifecycle"`
		} `json:"storage"`
	}
	if _, err := read(rules.SchemaConfig, &config); err != nil {
		return assets, err
	}
	seenIDs := map[string]bool{}
	for _, group := range rules.SchemaGroups {
		if len(config.Storage[group]) == 0 {
			return assets, fmt.Errorf("required storage schema group missing: %s", group)
		}
		for selector, entry := range config.Storage[group] {
			if !safePath(entry.Path) || !schemaRegistrationPath(group, selector, entry.Path, entry.Version, entry.Lifecycle) ||
				entry.ID == "" || entry.Version == "" || seenIDs[entry.ID] {
				return assets, fmt.Errorf("invalid schema registration %s/%s", group, selector)
			}
			seenIDs[entry.ID] = true
			name := path.Join(path.Dir(rules.SchemaConfig), entry.Path)
			var identity struct {
				ID string `json:"$id"`
			}
			raw, err := read(name, &identity)
			if err != nil {
				return assets, err
			}
			if identity.ID != entry.ID || (entry.Digest != "" && sha(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))) != entry.Digest) {
				return assets, fmt.Errorf("schema identity or registered digest mismatch: %s", name)
			}
			if group == "profile" && entry.Lifecycle == "ACTIVE" {
				if assets.activeProfile != "" {
					return assets, fmt.Errorf("more than one active Profile registration")
				}
				assets.activeProfile = entry.Version
				assets.activeProfilePath = strings.TrimPrefix(name, rules.ProductPrefix)
			}
		}
	}
	if assets.activeProfile == "" {
		return assets, fmt.Errorf("active Profile registration is missing")
	}
	var index struct {
		Entries []engineEntry     `json:"official_engine_entries"`
		Schemas map[string]string `json:"engine_schema_refs"`
	}
	if _, err := read(rules.EngineIndex, &index); err != nil {
		return assets, err
	}
	for _, engine := range index.Entries {
		kind := strings.ToLower(engine.Kind)
		if !safePath(kind) || path.Base(kind) != kind || engine.ID == "" || engine.Version == "" || assets.engines[kind].ID != "" {
			return assets, fmt.Errorf("invalid or duplicate registered engine %s", engine.Kind)
		}
		assets.engines[kind] = engine
	}
	if len(assets.engines) == 0 {
		return assets, fmt.Errorf("official engine inventory is empty")
	}
	for _, reference := range index.Schemas {
		var document map[string]any
		if _, err := read(reference, &document); err != nil {
			return assets, err
		}
	}
	for _, reference := range rules.EngineRuntimeResources {
		var document map[string]any
		if _, err := read(reference, &document); err != nil {
			return assets, err
		}
	}
	var snapshot struct {
		Refs []struct {
			Path   string `json:"path"`
			Digest string `json:"sha256"`
		} `json:"source_refs"`
	}
	if _, err := read(rules.EnginePins, &snapshot); err != nil {
		return assets, err
	}
	for _, pin := range snapshot.Refs {
		var document map[string]any
		raw, err := read(pin.Path, &document)
		if err != nil {
			return assets, err
		}
		if sha(raw) != pin.Digest {
			return assets, fmt.Errorf("registered engine asset digest mismatch: %s", pin.Path)
		}
	}
	err := assets.loadCommonAssets(source, rules.CommonRegistry)
	return assets, err
}

func schemaRegistrationPath(group, selector, name, version, lifecycle string) bool {
	if group != "profile" {
		return strings.HasPrefix(name, "storage/")
	}
	if selector == "ptsip.project-profile" {
		return lifecycle == "ACTIVE" && name == "storage/profile/project-profile.schema.json"
	}
	return lifecycle == "DEPRECATED" && path.Dir(name) == "history/profile" &&
		path.Base(name) == "project-profile-"+version+".schema.json"
}

func (assets *registeredAssets) loadCommonAssets(source gitSource, registry string) error {
	if !strings.HasPrefix(registry, assets.productPrefix) {
		return fmt.Errorf("common-contract registry outside product root: %s", registry)
	}
	type resource struct {
		Path   string `json:"resource"`
		Digest string `json:"sha256"`
	}
	var common struct {
		Format      string     `json:"format"`
		Revision    string     `json:"revision"`
		Contracts   []resource `json:"contracts"`
		Diagnostics []resource `json:"diagnostics"`
	}
	registryBytes, err := source.decode(registry, &common)
	if err != nil {
		return err
	}
	if common.Format != "ptsip-common-contract-registry/v1" || common.Revision != "v2" || len(common.Contracts) == 0 || len(common.Diagnostics) == 0 {
		return fmt.Errorf("common-contract registry must declare v2 contracts and diagnostics: %s", registry)
	}
	assets.commonRegistryDigest = sha(registryBytes)
	assets.digests[registry] = assets.commonRegistryDigest
	assets.exactDigestPaths = map[string]bool{registry: true}
	for _, row := range append(common.Contracts, common.Diagnostics...) {
		if !safePath(row.Path) || !(strings.HasPrefix(row.Path, "schemas/policy/") || strings.HasPrefix(row.Path, "schemas/plan/")) {
			return fmt.Errorf("invalid common-contract resource: %s", row.Path)
		}
		name := assets.productPrefix + row.Path
		if assets.exactDigestPaths[name] {
			if assets.digests[name] != row.Digest {
				return fmt.Errorf("conflicting common-contract resource digests: %s", row.Path)
			}
			continue
		}
		var document map[string]any
		raw, err := source.decode(name, &document)
		if err != nil {
			return err
		}
		if document == nil || sha(raw) != row.Digest {
			return fmt.Errorf("common-contract resource digest mismatch: %s", row.Path)
		}
		assets.digests[name], assets.exactDigestPaths[name] = row.Digest, true
	}
	return nil
}

func prebuildProblems(source gitSource, contract buildContract) []string {
	var problems []string
	metadata, err := source.read(contract.Prebuild.Frontend)
	if err != nil {
		problems = append(problems, err.Error())
	} else if _, section, ok := strings.Cut(string(metadata), "[build-system]"); !ok {
		problems = append(problems, "wheel frontend lacks a build-system declaration")
	} else {
		section, _, _ = strings.Cut(section, "\n[")
		if strings.Contains(section, ".ptsip/") {
			problems = append(problems, "wheel frontend still loads a developer-private build backend")
		}
	}
	for _, name := range contract.Prebuild.RequiredFiles {
		if !strings.HasPrefix(name, contract.Source.ProductPrefix) && !strings.HasPrefix(name, "src/build/") && !strings.HasPrefix(name, "src/policy/") {
			problems = append(problems, "wheel input outside the product and canonical projection roots: "+name)
		} else if _, err := source.read(name); err != nil {
			problems = append(problems, err.Error())
		}
	}
	return problems
}
