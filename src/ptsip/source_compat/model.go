// Package sourcecompat reads frozen historical Profile declarations.
package sourcecompat

import "github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"

type Family string

const (
	Tool035Profile Family = "TOOL_0.3.5_PROFILE_0.3.4_DRAFT"
	Tool036Profile Family = "TOOL_0.3.6_PROFILE_0.3.6_DRAFT"
)

type Scope string

const (
	TopLevel Scope = "TOP_LEVEL"
	Override Scope = "RESPONSIBILITY_MAP_OVERRIDE"
)

// ObservedBinding is the binding returned by discovery.profile.candidates.
type ObservedBinding struct {
	Path                  string  `json:"path"`
	DeclaredVersion       string  `json:"declared_version"`
	SpecificationRevision string  `json:"specification_revision"`
	SpecificationSource   *string `json:"specification_source"`
	ContentSHA256         string  `json:"content_sha256"`
	Temporary             bool    `json:"temporary"`
}

type Binding struct {
	ProfilePath           string  `json:"profile_path"`
	DeclaredVersion       string  `json:"declared_version"`
	SpecificationRevision string  `json:"specification_revision"`
	SpecificationSource   *string `json:"specification_source"`
	ContentSHA256         string  `json:"content_sha256"`
	Temporary             bool    `json:"temporary"`
}

type Location struct {
	ProfilePath string `json:"profile_path"`
	Pointer     string `json:"pointer"`
}

type Attribute struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type Component struct {
	ID             string      `json:"id"`
	Classification string      `json:"source_classification"`
	Include        []string    `json:"include"`
	Exclude        []string    `json:"exclude"`
	Purpose        string      `json:"purpose"`
	Scope          Scope       `json:"scope"`
	Location       Location    `json:"location"`
	Attributes     []Attribute `json:"attributes"`
}

type Artifact struct {
	ID       string   `json:"id"`
	Anchor   string   `json:"anchor"`
	Include  []string `json:"include"`
	Exclude  []string `json:"exclude"`
	Purpose  string   `json:"purpose"`
	Scope    Scope    `json:"scope"`
	Location Location `json:"location"`
}

type Relationship struct {
	ID       string   `json:"id"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Type     string   `json:"type"`
	Scope    Scope    `json:"scope"`
	Location Location `json:"location"`
}

type Policy struct {
	Name     string   `json:"name"`
	Value    any      `json:"value"`
	Location Location `json:"location"`
}

type Boundary struct {
	Classification string   `json:"source_classification"`
	Roots          []string `json:"roots"`
	Location       Location `json:"location"`
}

type Semantics interface{ profileSemantics() }

type V034Semantics struct {
	Kind            string     `json:"kind"`
	DeclarationForm string     `json:"declaration_form"`
	Boundaries      []Boundary `json:"boundaries"`
}

func (V034Semantics) profileSemantics() {}

type V036Semantics struct {
	Kind             string   `json:"kind"`
	Mode             string   `json:"responsibility_map_mode"`
	TemplateID       *string  `json:"template_id"`
	TemplateRevision *string  `json:"template_revision"`
	RemoveComponents []string `json:"remove_component_ids"`
	RemoveArtifacts  []string `json:"remove_associated_artifact_ids"`
	RemoveRelations  []string `json:"remove_relationship_ids"`
}

func (V036Semantics) profileSemantics() {}

type Profile struct {
	Kind       string          `json:"kind"`
	Authority  string          `json:"authority"`
	Family     Family          `json:"family"`
	Generation Binding         `json:"generation"`
	Components []Component     `json:"components"`
	Artifacts  []Artifact      `json:"associated_artifacts"`
	Relations  []Relationship  `json:"relationships"`
	Policies   []Policy        `json:"policies"`
	Semantics  Semantics       `json:"family_semantics"`
	RawPayload source.Document `json:"raw_payload"`
}

type Issue struct {
	Code        string  `json:"code"`
	Message     string  `json:"message"`
	ProfilePath string  `json:"profile_path"`
	Pointer     *string `json:"pointer"`
}

type ReadResult struct {
	Complete bool     `json:"complete"`
	Profile  *Profile `json:"profile"`
	Issues   []Issue  `json:"issues"`
}

// ProjectionInput prepares input for the registered projection.migration.source
// operation, preserving the declaration pointers and the historical semantics.
func (p Profile) ProjectionInput() map[string]any {
	components := make([]map[string]any, 0, len(p.Components))
	for _, item := range p.Components {
		components = append(components, map[string]any{"id": item.ID, "include": item.Include, "exclude": item.Exclude,
			"purpose": item.Purpose, "source_pointer": item.Location.Pointer, "source_classification": item.Classification})
	}
	artifacts := make([]map[string]any, 0, len(p.Artifacts))
	for _, item := range p.Artifacts {
		artifacts = append(artifacts, map[string]any{"id": item.ID, "include": item.Include, "exclude": item.Exclude,
			"purpose": item.Purpose, "source_pointer": item.Location.Pointer})
	}
	relations := make([]map[string]string, 0, len(p.Relations))
	for _, item := range p.Relations {
		relations = append(relations, map[string]string{"id": item.ID, "from": item.From, "to": item.To, "type": item.Type})
	}
	var semantics any = p.Semantics
	if v, ok := p.Semantics.(V034Semantics); ok {
		boundaries := make([]map[string]any, 0, len(v.Boundaries))
		for _, item := range v.Boundaries {
			boundaries = append(boundaries, map[string]any{"source_classification": item.Classification, "roots": item.Roots, "source_pointer": item.Location.Pointer})
		}
		semantics = map[string]any{"kind": v.Kind, "declaration_form": v.DeclarationForm, "boundaries": boundaries}
	}
	return map[string]any{"family": p.Family, "family_semantics": semantics, "components": components,
		"associated_artifacts": artifacts, "relationships": relations}
}
