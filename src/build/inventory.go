package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

const (
	productRoot     = "src/ptsip/"
	engineRoot      = productRoot + "engines/"
	schemaRoot      = productRoot + "schemas/"
	engineIndex     = engineRoot + "index.json"
	engineSupport   = engineRoot + "discovery/profile-support.json"
	engineRelations = engineRoot + "validation/profile-transitions.json"
)

type buildReport struct {
	SourceRevision  string   `json:"source_revision"`
	ProductFiles    int      `json:"product_files"`
	EngineKinds     []string `json:"engine_kinds"`
	ProfileVersions []string `json:"profile_versions"`
	ActiveProfile   string   `json:"active_profile"`
	SourceArchive   string   `json:"product_source_archive,omitempty"`
}

type engineEntry struct {
	Kind      string `json:"kind"`
	Manifest  string `json:"manifest_ref"`
	Contract  string `json:"contract_ref"`
	Bindings  string `json:"execution_binding_ref"`
	CargoFile string `json:"cargo_manifest_ref"`
}

type profileRegistration struct {
	ID        string `json:"$id"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	SHA256    string `json:"schema_sha256"`
	Lifecycle string `json:"registered_lifecycle"`
}

func (source productSource) decode(name string, value any) error {
	file, ok := source.Files[name]
	if !ok {
		return fmt.Errorf("missing product source %s", name)
	}
	if _, err := jsondoc.Decode(file.Data); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := json.Unmarshal(file.Data, value); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func inspect(source productSource) (buildReport, error) {
	engines, err := inspectEngines(source)
	if err != nil {
		return buildReport{}, err
	}
	profiles, active, err := inspectProfiles(source)
	if err != nil {
		return buildReport{}, err
	}
	return buildReport{
		SourceRevision:  source.Revision,
		ProductFiles:    len(source.Files),
		EngineKinds:     engines,
		ProfileVersions: profiles,
		ActiveProfile:   active,
	}, nil
}

func inspectEngines(source productSource) ([]string, error) {
	var index struct {
		Entries   []engineEntry     `json:"official_engine_entries"`
		Schemas   map[string]string `json:"engine_schema_refs"`
		Transport string            `json:"transport_contract_ref"`
	}
	if err := source.decode(engineIndex, &index); err != nil {
		return nil, err
	}
	if len(index.Entries) == 0 {
		return nil, fmt.Errorf("no official engines registered in %s", engineIndex)
	}
	var kinds []string
	for _, entry := range index.Entries {
		if entry.Kind == "" || slices.Contains(kinds, entry.Kind) {
			return nil, fmt.Errorf("empty or repeated official engine kind %q", entry.Kind)
		}
		for _, name := range []string{entry.Manifest, entry.Contract, entry.Bindings} {
			if !strings.HasPrefix(name, engineRoot) || path.Clean(name) != name {
				return nil, fmt.Errorf("invalid engine %s registered source %q", entry.Kind, name)
			}
			file, ok := source.Files[name]
			if !ok || len(file.Data) == 0 {
				return nil, fmt.Errorf("missing engine %s registered source %s", entry.Kind, name)
			}
		}
		if !strings.HasPrefix(entry.CargoFile, engineRoot) {
			return nil, fmt.Errorf("engine %s Cargo manifest escapes product engines", entry.Kind)
		}
		if _, ok := source.Files[entry.CargoFile]; !ok {
			return nil, fmt.Errorf("missing engine %s Cargo manifest %s", entry.Kind, entry.CargoFile)
		}
		kinds = append(kinds, entry.Kind)
	}
	paths := []string{index.Transport, engineSupport, engineRelations}
	for _, name := range index.Schemas {
		paths = append(paths, name)
	}
	slices.Sort(paths)
	for _, name := range paths {
		if !strings.HasPrefix(name, engineRoot) {
			return nil, fmt.Errorf("registered engine asset escapes product engines: %q", name)
		}
		var asset map[string]json.RawMessage
		if err := source.decode(name, &asset); err != nil {
			return nil, err
		}
		if len(asset) == 0 {
			return nil, fmt.Errorf("registered engine asset is empty: %s", name)
		}
	}
	slices.Sort(kinds)
	return kinds, nil
}

func inspectProfiles(source productSource) ([]string, string, error) {
	var catalog struct {
		Storage struct {
			Profile map[string]profileRegistration `json:"profile"`
		} `json:"storage"`
	}
	if err := source.decode(schemaRoot+"config.json", &catalog); err != nil {
		return nil, "", err
	}
	var versions []string
	active := ""
	ids := map[string]bool{}
	for selector, row := range catalog.Storage.Profile {
		if row.ID == "" || row.Version == "" || slices.Contains(versions, row.Version) || ids[row.ID] {
			return nil, "", fmt.Errorf("invalid or repeated profile registration %s", selector)
		}
		if !strings.HasPrefix(row.Path, "storage/profile/") || path.Clean(row.Path) != row.Path {
			return nil, "", fmt.Errorf("profile registration %s escapes its schema root", selector)
		}
		name := schemaRoot + row.Path
		var schema struct {
			ID  string `json:"$id"`
			Ref string `json:"$ref"`
		}
		if err := source.decode(name, &schema); err != nil {
			return nil, "", err
		}
		digest := sha256.Sum256(bytes.ReplaceAll(source.Files[name].Data, []byte("\r\n"), []byte("\n")))
		if schema.ID != row.ID || hex.EncodeToString(digest[:]) != row.SHA256 {
			return nil, "", fmt.Errorf("profile schema %s differs from its registered identity or SHA-256", name)
		}
		if selector == "ptsip.project-profile" {
			if row.Lifecycle != "ACTIVE" || schema.Ref != "" {
				return nil, "", fmt.Errorf("official profile %s must be an active direct schema", name)
			}
			active = row.Version
		}
		ids[row.ID] = true
		versions = append(versions, row.Version)
	}
	if active == "" {
		return nil, "", fmt.Errorf("no official profile registration")
	}
	slices.Sort(versions)
	return versions, active, nil
}
