// Package releasenote generates English documentation from declared version
// changes and explicit formatting contracts. It has no release admission role.
package releasenote

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractcheck"
)

const SchemaRoot = ".ptsip/contracts/schemas/release/releasenote/"
const WorkflowPath = ".ptsip/contracts/release/releasenote.json"
const BasicID = "urn:ptsip:docs:format:basic-readme:v1"

type Catalog struct {
	Root      string
	Resources []any
	Workflow  Workflow
}

type Workflow struct {
	Schema          string                                  `json:"schema_version"`
	Readme          struct{ Path, Language, Schema string } `json:"canonical_readme"`
	VersionContract string                                  `json:"version_change_contract"`
	VersionSchema   string                                  `json:"version_change_schema"`
	Translation     struct {
		Registry       string `json:"registry"`
		RegistrySchema string `json:"registry_schema"`
		PathSchema     string `json:"path_schema"`
		Source         string `json:"source"`
		Archive        bool   `json:"archive"`
		TargetSet      string `json:"target_set"`
	} `json:"translation"`
	Archive struct {
		Root, Language string
		WritePolicy    string `json:"write_policy"`
		Translated     bool   `json:"translated_snapshots"`
		EngineIndex    bool   `json:"engine_directory_index"`
		Types          map[string]struct{ Schema, Path, Documentation string }
	} `json:"archive"`
	NoteSections []string `json:"note_sections"`
}

func ReadJSON(root, relative string, value any) error {
	filename, err := authoring.LocalFile(root, relative)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, value); err != nil {
		return fmt.Errorf("INVALID_DOCUMENTATION_JSON: %s: %w", relative, err)
	}
	return nil
}

func Load(root string) (*Catalog, error) {
	c := &Catalog{Root: root}
	if err := ReadJSON(root, WorkflowPath, &c.Workflow); err != nil {
		return nil, err
	}
	w := c.Workflow
	if w.Schema != "ptsip-release-documentation/v1" || w.Readme.Path != "README.md" || w.Readme.Language != "en" ||
		w.Readme.Schema != SchemaRoot+"Basic_Formatting_readme.schema.json" ||
		w.VersionContract != ".ptsip/contracts/release/versions_different.json" || w.VersionSchema != ".ptsip/contracts/schemas/release/versions_different.schema.json" ||
		w.Translation.Registry != "docs/translated/index.yaml" || w.Translation.Source != "README.md" || w.Translation.Archive || w.Translation.TargetSet != "REGISTERED_PATHS_ONLY" ||
		w.Translation.RegistrySchema != SchemaRoot+"readme-translation-registry.schema.json" || w.Translation.PathSchema != SchemaRoot+"readme-translation-path.schema.json" ||
		w.Archive.Root != "docs/releasenote" || w.Archive.Language != "en" || w.Archive.WritePolicy != "CREATE_ONLY_OR_IDENTICAL" || w.Archive.Translated || w.Archive.EngineIndex ||
		strings.Join(w.NoteSections, ",") != "Summary,Version changes,Technical details,Documentation" || len(w.Archive.Types) != 4 {
		return nil, fmt.Errorf("DOCUMENTATION_ROUTING_CONTRACT_INVALID")
	}
	for kind, expected := range map[string]string{
		"engine":  "docs/releasenote/engine/{engine}/{engine}-engine-{version}.md",
		"profile": "docs/releasenote/project-profile/{version}.md",
		"tool":    "docs/releasenote/tool/{filename_version}.md",
		"package": "docs/releasenote/PTSIP_Package/{version}.md",
	} {
		route, present := w.Archive.Types[kind]
		expectedSchema := SchemaRoot + kind + "_Formatting_readme.schema.json"
		if kind == "package" {
			expectedSchema = SchemaRoot + "Basic_Formatting_readme.schema.json#/$defs/ReleaseNote"
		}
		if !present || route.Path != expected || route.Schema != expectedSchema || route.Documentation == "" {
			return nil, fmt.Errorf("ARCHIVE_ROUTE_INVALID: %s", kind)
		}
	}
	sources := []contractcheck.Source{
		{Path: SchemaRoot + "Basic_Formatting_readme.schema.json", ID: BasicID},
		{Path: SchemaRoot + "engine_Formatting_readme.schema.json", ID: "urn:ptsip:docs:format:engine-readme:v1"},
		{Path: SchemaRoot + "profile_Formatting_readme.schema.json", ID: "urn:ptsip:docs:format:profile-readme:v1"},
		{Path: SchemaRoot + "tool_Formatting_readme.schema.json", ID: "urn:ptsip:docs:format:tool-readme:v1"},
		{Path: w.VersionSchema, ID: "urn:ptsip:release:versions-different-contract:v1"},
		{Path: w.Translation.RegistrySchema, ID: "urn:ptsip:docs:readme-translation-registry:v1"},
		{Path: w.Translation.PathSchema, ID: "urn:ptsip:docs:readme-translation-path:v1"},
	}
	var err error
	c.Resources, err = contractcheck.Load(root, sources...)
	return c, err
}

func (c *Catalog) Validate(v authoring.Validator, id string, instance any) error {
	raw, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return err
	}
	schema := map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "$ref": id}
	return contractcheck.Validate(v, schema, document, c.Resources)
}

// ValidateRegistry is shared by the native developer entrypoint and the
// translation command's existing closed-schema adapter.
func ValidateRegistry(root string, value any, v authoring.Validator) error {
	resources, err := contractcheck.Load(root,
		contractcheck.Source{Path: SchemaRoot + "readme-translation-registry.schema.json", ID: "urn:ptsip:docs:readme-translation-registry:v1"},
		contractcheck.Source{Path: SchemaRoot + "readme-translation-path.schema.json", ID: "urn:ptsip:docs:readme-translation-path:v1"},
	)
	if err != nil {
		return err
	}
	c := Catalog{Root: root, Resources: resources}
	if err := c.Validate(v, "urn:ptsip:docs:readme-translation-registry:v1", value); err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var registry struct {
		Canonical    struct{ Language, Path string }
		Translations []struct{ Language, Path string }
	}
	if err := json.Unmarshal(raw, &registry); err != nil {
		return err
	}
	languages, paths := map[string]bool{"en": true}, map[string]bool{"README.md": true}
	for _, target := range registry.Translations {
		if languages[target.Language] || paths[target.Path] || target.Path != "docs/translated/README."+target.Language+".md" {
			return fmt.Errorf("TRANSLATION_LANGUAGE_PATH_MISMATCH_OR_DUPLICATE")
		}
		languages[target.Language], paths[target.Path] = true, true
	}
	return nil
}
