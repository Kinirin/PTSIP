package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

const (
	engineGraph = engineRoot + "contracts/dependency-graph.json"
	engineState = "urn:ptsip:official-engine-discovery-snapshot:v1"
)

type engineSourceRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

type engineDeclaration struct {
	Provider      string          `json:"provider_engine_id"`
	Capability    string          `json:"capability_id"`
	Version       string          `json:"contract_version"`
	Compatibility engineSourceRef `json:"compatibility_ref"`
}

type registeredEngine struct {
	Manifest     engineSourceRef
	Contract     engineSourceRef
	Dependencies []engineDeclaration
}

type engineSnapshotDocument struct {
	StateID          string            `json:"state_id"`
	SourceRefs       []engineSourceRef `json:"source_refs"`
	SnapshotRevision string            `json:"snapshot_revision"`
	Authority        bool              `json:"authority"`
	WritePermission  bool              `json:"write_permission"`
	ReuseEligible    bool              `json:"reuse_eligible"`
}

type engineGraphEdge struct {
	Provider         string          `json:"provider"`
	Consumer         string          `json:"consumer"`
	Relation         string          `json:"relation"`
	ProviderEvidence engineSourceRef `json:"provider_evidence"`
	ConsumerEvidence engineSourceRef `json:"consumer_evidence"`
}

type engineGraphDocument struct {
	Nodes []engineSourceRef `json:"nodes"`
	Edges []engineGraphEdge `json:"edges"`
}

type engineOwnerIndex struct {
	Format     string            `json:"schema_version"`
	Class      string            `json:"policy_class"`
	Entries    []engineEntry     `json:"official_engine_entries"`
	Schemas    map[string]string `json:"engine_schema_refs"`
	Transport  string            `json:"transport_contract_ref"`
	Federation struct {
		Owner     string `json:"owner_id"`
		Static    string `json:"static_registry_ref"`
		Execution string `json:"execution_registry_ref"`
	} `json:"federation"`
}

func engineSourcePath(name string) bool {
	return strings.HasPrefix(name, engineRoot) && path.Clean(name) == name &&
		!strings.ContainsAny(name, "\\:\x00\r\n")
}

func decodeEngineSource(source productSource, name string, dst any) error {
	if !engineSourcePath(name) {
		return fmt.Errorf("invalid registered engine source path %q", name)
	}
	return source.decode(name, dst)
}

func engineSourceDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func engineReference(source productSource, name, id, version string) (engineSourceRef, error) {
	if !engineSourcePath(name) || id == "" || version == "" {
		return engineSourceRef{}, fmt.Errorf("missing registered engine source identity for %q", name)
	}
	item, ok := source.Files[name]
	if !ok {
		return engineSourceRef{}, fmt.Errorf("missing registered engine source %q", name)
	}
	return engineSourceRef{id, version, name, engineSourceDigest(item.Data)}, nil
}

func verifyEngineReference(source productSource, ref engineSourceRef) (map[string]any, error) {
	var object map[string]any
	if err := decodeEngineSource(source, ref.Path, &object); err != nil {
		return nil, err
	}
	id := firstEngineIdentity(object, "contract_id", "id", "engine_id")
	version := firstEngineIdentity(object, "version", "implementation_version")
	current, err := engineReference(source, ref.Path, id, version)
	if err != nil {
		return nil, err
	}
	if current != ref {
		return nil, fmt.Errorf("registered engine identity or SHA-256 mismatch for %s", ref.Path)
	}
	return object, nil
}

func firstEngineIdentity(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := object[key]; ok {
			text, _ := value.(string)
			return text
		}
	}
	return ""
}

type engineStaticRoute struct {
	Engine   string `json:"engine_id"`
	Manifest string `json:"manifest_ref"`
}

type engineRuntimeRoute struct {
	Engine  string `json:"engine_id"`
	Binding string `json:"binding_ref"`
}

func registeredEngineSources(source productSource) (engineOwnerIndex, []registeredEngine, []engineSourceRef, error) {
	var index engineOwnerIndex
	if err := source.decode(engineIndex, &index); err != nil {
		return index, nil, nil, err
	}
	if index.Format != "ptsip-official-engine-index/v1" || index.Class != "PTSIP_SUPPORT_FEATURE" ||
		index.Federation.Owner == "" || len(index.Entries) != 6 {
		return index, nil, nil, fmt.Errorf("unsupported official engine owner index")
	}
	var statics struct {
		Owner      string              `json:"owner_id"`
		Origin     string              `json:"origin"`
		Permission *bool               `json:"execution_permission"`
		Entries    []engineStaticRoute `json:"entries"`
	}
	if err := decodeEngineSource(source, index.Federation.Static, &statics); err != nil {
		return index, nil, nil, err
	}
	var runtime struct {
		Owner      string               `json:"owner_id"`
		Permission *bool                `json:"execution_permission"`
		Bindings   []engineRuntimeRoute `json:"bindings"`
	}
	if err := decodeEngineSource(source, index.Federation.Execution, &runtime); err != nil {
		return index, nil, nil, err
	}
	if statics.Owner != index.Federation.Owner || runtime.Owner != index.Federation.Owner || statics.Origin != "OFFICIAL" ||
		!declaresNoPermission(statics.Permission) || !declaresNoPermission(runtime.Permission) ||
		len(statics.Entries) != len(index.Entries) || len(runtime.Bindings) != len(index.Entries) {
		return index, nil, nil, fmt.Errorf("federated engine registrations differ from their owner index")
	}
	seenKinds := map[string]bool{}
	seenIDs := map[string]bool{}
	seenPaths := map[string]bool{}
	kinds := []string{"CONTEXT", "DISCOVERY", "EXECUTION", "PROJECTION", "STORAGE", "VALIDATION"}
	var engines []registeredEngine
	var refs []engineSourceRef
	for _, entry := range index.Entries {
		if !slices.Contains(kinds, entry.Kind) || seenKinds[entry.Kind] ||
			!slices.Contains(statics.Entries, engineStaticRoute{entry.ID, entry.Manifest}) ||
			!slices.Contains(runtime.Bindings, engineRuntimeRoute{entry.ID, entry.Bindings}) {
			return index, nil, nil, fmt.Errorf("missing or ambiguous registration for official engine %q", entry.Kind)
		}
		seenKinds[entry.Kind] = true
		engine, binding, err := readRegisteredEngine(source, entry, index.Transport)
		if err != nil {
			return index, nil, nil, err
		}
		for _, ref := range []engineSourceRef{engine.Manifest, engine.Contract, binding} {
			if seenPaths[ref.Path] || seenIDs[ref.ID] {
				return index, nil, nil, fmt.Errorf("ambiguous registered engine source %q", ref.Path)
			}
			seenPaths[ref.Path], seenIDs[ref.ID] = true, true
			refs = append(refs, ref)
		}
		engines = append(engines, engine)
	}
	if err := verifyEngineDeclarations(source, engines); err != nil {
		return index, nil, nil, err
	}
	return index, engines, refs, nil
}

func declaresNoPermission(permission *bool) bool {
	return permission != nil && !*permission
}

func readRegisteredEngine(source productSource, entry engineEntry, transportPath string) (registeredEngine, engineSourceRef, error) {
	bad := func(err error) (registeredEngine, engineSourceRef, error) {
		return registeredEngine{}, engineSourceRef{}, fmt.Errorf("engine %s: %w", entry.Kind, err)
	}
	var manifest struct {
		ID           string              `json:"engine_id"`
		Origin       string              `json:"origin"`
		Version      string              `json:"implementation_version"`
		Capabilities []engineSourceRef   `json:"capabilities"`
		Extensions   []engineSourceRef   `json:"extensions"`
		Dependencies []engineDeclaration `json:"dependencies"`
	}
	if err := decodeEngineSource(source, entry.Manifest, &manifest); err != nil {
		return bad(err)
	}
	if manifest.ID != entry.ID || manifest.Version != entry.Version || manifest.Origin != "OFFICIAL" ||
		manifest.Extensions == nil || manifest.Dependencies == nil || len(manifest.Capabilities) != 1 {
		return bad(fmt.Errorf("manifest differs from its registered implementation or capability"))
	}
	var contract struct {
		EngineID   string `json:"engine_id"`
		ID         string `json:"contract_id"`
		Version    string `json:"version"`
		Owner      string `json:"semantic_owner"`
		Permission *bool  `json:"execution_permission_from_declaration"`
		Operations []struct {
			ID string `json:"operation_id"`
		} `json:"operations"`
	}
	if err := decodeEngineSource(source, entry.Contract, &contract); err != nil {
		return bad(err)
	}
	if contract.EngineID != entry.ID || contract.ID != entry.Capability || contract.Owner != entry.Kind ||
		!declaresNoPermission(contract.Permission) || len(contract.Operations) != len(entry.Operations) {
		return bad(fmt.Errorf("capability contract differs from its registered owner or operations"))
	}
	operations := map[string]bool{}
	for _, operation := range contract.Operations {
		if operation.ID == "" || operations[operation.ID] || !slices.Contains(entry.Operations, operation.ID) {
			return bad(fmt.Errorf("missing or ambiguous registered operation %q", operation.ID))
		}
		operations[operation.ID] = true
	}
	contractRef, err := engineReference(source, entry.Contract, contract.ID, contract.Version)
	if err != nil {
		return bad(err)
	}
	if manifest.Capabilities[0] != contractRef {
		return bad(fmt.Errorf("current capability identity or SHA-256 differs from the manifest"))
	}
	for _, extension := range manifest.Extensions {
		item, err := verifyEngineReference(source, extension)
		if err != nil {
			return bad(err)
		}
		if item["engine_id"] != entry.ID {
			return bad(fmt.Errorf("extension belongs to another engine: %s", extension.Path))
		}
	}
	var bindings versionsource.EngineBindings
	if err := decodeEngineSource(source, entry.Bindings, &bindings); err != nil {
		return bad(err)
	}
	binding, err := bindings.Select(entry.ID, entry.Version, contract.ID, contract.Version)
	if err != nil {
		return bad(err)
	}
	if binding.Transport.Path != transportPath {
		return bad(fmt.Errorf("execution binding differs from registered implementation, contract or transport"))
	}
	if _, err := verifyEngineReference(source, engineSourceRef(binding.Transport)); err != nil {
		return bad(err)
	}
	manifestRef, err := engineReference(source, entry.Manifest, manifest.ID, manifest.Version)
	if err != nil {
		return bad(err)
	}
	bindingRef, err := engineReference(source, entry.Bindings, binding.ID, binding.EngineVersion)
	if err != nil {
		return bad(err)
	}
	return registeredEngine{manifestRef, contractRef, manifest.Dependencies}, bindingRef, nil
}

func verifyEngineDeclarations(source productSource, engines []registeredEngine) error {
	capabilities := make(map[string]registeredEngine, len(engines))
	for _, engine := range engines {
		capabilities[engine.Contract.ID] = engine
	}
	for _, engine := range engines {
		for _, dependency := range engine.Dependencies {
			provider, ok := capabilities[dependency.Capability]
			if !ok || provider.Manifest.ID != dependency.Provider || provider.Contract.Version != dependency.Version {
				return fmt.Errorf("unknown or incompatible capability dependency for %s", engine.Manifest.ID)
			}
			evidence, err := verifyEngineReference(source, dependency.Compatibility)
			if err != nil {
				return err
			}
			if evidence["provider_engine_id"] != dependency.Provider || evidence["consumer_engine_id"] != engine.Manifest.ID ||
				evidence["outcome"] != "PASS" {
				return fmt.Errorf("current compatibility evidence unavailable for %s", engine.Manifest.ID)
			}
			raw, err := json.Marshal(evidence["source_refs"])
			if err != nil {
				return err
			}
			var references []engineSourceRef
			if err := json.Unmarshal(raw, &references); err != nil || references == nil {
				return fmt.Errorf("missing compatibility source bindings for %s", engine.Manifest.ID)
			}
			for _, ref := range references {
				if _, err := verifyEngineReference(source, ref); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func deriveEngineSnapshot(refs []engineSourceRef) (engineSnapshotDocument, error) {
	// The registered revision is SHA-256 of the JSON array with sorted object keys.
	objects := make([]map[string]string, len(refs))
	for i, ref := range refs {
		objects[i] = map[string]string{"id": ref.ID, "version": ref.Version, "path": ref.Path, "sha256": ref.SHA256}
	}
	raw, err := json.Marshal(objects)
	if err != nil {
		return engineSnapshotDocument{}, err
	}
	return engineSnapshotDocument{StateID: engineState, SourceRefs: refs, SnapshotRevision: engineSourceDigest(raw)}, nil
}

func deriveEngineGraph(source productSource, engines []registeredEngine) (engineGraphDocument, error) {
	result := engineGraphDocument{Nodes: make([]engineSourceRef, 0, len(engines)), Edges: []engineGraphEdge{}}
	nodes := make(map[string]engineSourceRef, len(engines))
	expected := map[[2]string]bool{}
	for _, engine := range engines {
		result.Nodes = append(result.Nodes, engine.Manifest)
		nodes[engine.Manifest.ID] = engine.Manifest
		for _, dependency := range engine.Dependencies {
			expected[[2]string{dependency.Provider, engine.Manifest.ID}] = true
		}
	}
	if len(expected) == 0 {
		return result, nil
	}
	var previous engineGraphDocument
	if err := source.decode(engineGraph, &previous); err != nil {
		return engineGraphDocument{}, err
	}
	verified := map[[3]string]bool{}
	for _, edge := range previous.Edges {
		pair := [2]string{edge.Provider, edge.Consumer}
		relation := [3]string{edge.Provider, edge.Consumer, edge.Relation}
		if !expected[pair] || edge.Relation != "DEPENDENCY" || verified[relation] {
			return engineGraphDocument{}, fmt.Errorf("graph edge lacks a unique current capability dependency between %s and %s", edge.Provider, edge.Consumer)
		}
		// Prior graph entries only identify the two separately owned records.
		// Their identity and provider/consumer source bytes are checked again.
		provider, err := verifyGraphEvidence(source, edge.ProviderEvidence, edge.Provider, edge, nodes)
		if err != nil {
			return engineGraphDocument{}, err
		}
		consumer, err := verifyGraphEvidence(source, edge.ConsumerEvidence, edge.Consumer, edge, nodes)
		if err != nil {
			return engineGraphDocument{}, err
		}
		if provider != consumer || (provider != "AFFECTED" && provider != "COMPATIBLE") {
			return engineGraphDocument{}, fmt.Errorf("graph provider and consumer evidence does not agree for %s and %s", edge.Provider, edge.Consumer)
		}
		delete(expected, pair)
		verified[relation] = true
		result.Edges = append(result.Edges, edge)
	}
	if len(expected) != 0 {
		return engineGraphDocument{}, fmt.Errorf("declared capability dependencies lack current provider and consumer graph evidence")
	}
	slices.SortFunc(result.Edges, func(a, b engineGraphEdge) int {
		return strings.Compare(a.Provider+"\x00"+a.Consumer, b.Provider+"\x00"+b.Consumer)
	})
	return result, nil
}

func verifyGraphEvidence(source productSource, ref engineSourceRef, owner string, edge engineGraphEdge, nodes map[string]engineSourceRef) (string, error) {
	record, err := verifyEngineReference(source, ref)
	if err != nil {
		return "", err
	}
	if record["producer_id"] != owner || record["provider"] != edge.Provider || record["consumer"] != edge.Consumer ||
		record["relation"] != edge.Relation || record["provider_sha256"] != nodes[edge.Provider].SHA256 ||
		record["consumer_sha256"] != nodes[edge.Consumer].SHA256 || record["outcome"] != "PASS" {
		return "", fmt.Errorf("graph evidence has stale owner, scope or source bindings: %s", ref.Path)
	}
	status, _ := record["compatibility"].(string)
	return status, nil
}

func verifyDerivedDocument(source productSource, name string, expected any) error {
	var observed map[string]any
	if err := source.decode(name, &observed); err != nil {
		return err
	}
	raw, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	var generated map[string]any
	if err := json.Unmarshal(raw, &generated); err != nil {
		return err
	}
	if !reflect.DeepEqual(observed, generated) {
		return fmt.Errorf("stale derived engine document %s", name)
	}
	return nil
}

type engineHandoff struct {
	SourceSHA      string `json:"source_sha"`
	SyncSHA        string `json:"sync_sha"`
	SnapshotSHA256 string `json:"snapshot_sha256"`
	GraphSHA256    string `json:"dependency_graph_sha256"`
}

// verifyEngineHandoff re-verifies the committed bytes selected by the CI
// handoff. Source synchronization and derived generation remain separate.
func verifyEngineHandoff(root, syncSHA string) (engineHandoff, error) {
	if !revisionPattern.MatchString(syncSHA) {
		return engineHandoff{}, fmt.Errorf("exact engine sync commit SHA is required")
	}
	revision, err := buildRevision(root)
	if err != nil {
		return engineHandoff{}, err
	}
	if _, err := runGit(root, "merge-base", "--is-ancestor", syncSHA, revision); err != nil {
		return engineHandoff{}, fmt.Errorf("verified source must descend from the engine sync commit: %w", err)
	}
	if _, err := runGit(root, "diff", "--quiet", syncSHA, revision, "--", "src/VERSION"); err != nil {
		return engineHandoff{}, fmt.Errorf("verified source version changed after the engine sync commit: %w", err)
	}
	source, err := loadGitSource(root)
	if err != nil {
		return engineHandoff{}, err
	}
	if source.Revision != revision {
		return engineHandoff{}, fmt.Errorf("source HEAD changed during engine verification")
	}
	if _, err := inspect(source); err != nil {
		return engineHandoff{}, err
	}
	current, err := buildRevision(root)
	if err != nil {
		return engineHandoff{}, err
	}
	if current != revision {
		return engineHandoff{}, fmt.Errorf("source HEAD changed during engine verification")
	}
	return engineHandoff{
		SourceSHA:      revision,
		SyncSHA:        syncSHA,
		SnapshotSHA256: engineSourceDigest(source.Files[engineSnapshot].Data),
		GraphSHA256:    engineSourceDigest(source.Files[engineGraph].Data),
	}, nil
}

func verifyEngineWheelSource(root, revision string) error {
	expected := os.Getenv("PTSIP_VERIFIED_ENGINE_SOURCE_SHA")
	if !revisionPattern.MatchString(expected) || expected != revision {
		return fmt.Errorf("wheel requires the exact verified engine source SHA from the CI handoff")
	}
	receipt, err := verifyEngineHandoff(root, os.Getenv("PTSIP_ENGINE_VERSION_SYNC_SHA"))
	if err != nil {
		return err
	}
	if receipt.SourceSHA != expected {
		return fmt.Errorf("engine handoff source differs from the wheel source")
	}
	return nil
}

func verifyEngineVersions(source productSource, entries []engineEntry) error {
	versions, err := versionsource.Parse(source.Files["src/VERSION"].Data)
	if err != nil {
		return err
	}
	if err := versions.VerifyOwners(source.Files[schemaRoot+"config.json"].Data,
		source.Files[productRoot+"common_contract/registry/manifest.json"].Data); err != nil {
		return err
	}
	for _, entry := range entries {
		if versions.Engines[strings.ToLower(entry.Kind)] != entry.Version {
			return fmt.Errorf("engine %s version differs from src/VERSION", entry.Kind)
		}
	}
	return nil
}

func runEngineDerived(command, root, syncSHA string, stdout io.Writer) error {
	output := json.NewEncoder(stdout)
	output.SetIndent("", "  ")
	if command == "wheel-ready" {
		revision, err := buildRevision(root)
		if err != nil {
			return err
		}
		return verifyEngineWheelSource(root, revision)
	}
	if command == "verify-engine-derived" {
		receipt, err := verifyEngineHandoff(root, syncSHA)
		if err != nil {
			return err
		}
		return output.Encode(receipt)
	}
	revision, err := buildRevision(root)
	if err != nil {
		return err
	}
	source, err := loadGitSource(root)
	if err != nil {
		return err
	}
	if source.Revision != revision {
		return fmt.Errorf("source HEAD changed during engine derivation")
	}
	index, engines, refs, err := registeredEngineSources(source)
	if err != nil {
		return err
	}
	if err := verifyEngineVersions(source, index.Entries); err != nil {
		return err
	}
	if command == "engine-snapshot" {
		snapshot, err := deriveEngineSnapshot(refs)
		if err != nil {
			return err
		}
		return output.Encode(snapshot)
	}
	graph, err := deriveEngineGraph(source, engines)
	if err != nil {
		return err
	}
	return output.Encode(graph)
}
