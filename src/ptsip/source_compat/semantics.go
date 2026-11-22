package sourcecompat

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

func project(family Family, generation ObservedBinding, doc source.Document) Profile {
	profile := Profile{
		Kind: "PTSIP_COMPATIBILITY_SOURCE", Authority: "SOURCE_DECLARATION_ONLY", Family: family,
		Generation: Binding{generation.Path, generation.DeclaredVersion, generation.SpecificationRevision,
			generation.SpecificationSource, generation.ContentSHA256, generation.Temporary},
		Components: []Component{}, Artifacts: []Artifact{}, Relations: []Relationship{}, Policies: []Policy{}, RawPayload: doc,
	}
	location := func(pointer string) Location { return Location{ProfilePath: generation.Path, Pointer: pointer} }
	policies := source.Object(doc["policies"])
	for _, name := range slices.Sorted(maps.Keys(policies)) {
		profile.Policies = append(profile.Policies, Policy{name, policies[name], location("/policies/" + pointerToken(name))})
	}
	if policy, found := doc["component_dependency_policy"]; found {
		profile.Policies = append(profile.Policies, Policy{"component_dependency_policy", policy, location("/component_dependency_policy")})
	}

	declarations, prefix, scope := doc, "", TopLevel
	if family == Tool035Profile {
		semantics := V034Semantics{Kind: "V034", DeclarationForm: "COMPONENTS", Boundaries: []Boundary{}}
		if boundaries := source.Object(doc["boundaries"]); boundaries != nil {
			semantics.DeclarationForm = "BOUNDARIES"
			for _, entry := range []struct{ name, classification string }{
				{"product", "PRODUCT"}, {"toolchain", "TOOLCHAIN"}, {"neutral_contract", "NEUTRAL_CONTRACT"},
			} {
				if boundary := source.Object(boundaries[entry.name]); boundary != nil {
					semantics.Boundaries = append(semantics.Boundaries, Boundary{entry.classification, stringsOf(boundary["roots"]), location("/boundaries/" + entry.name)})
				}
			}
		}
		profile.Semantics = semantics
	} else {
		responsibility := source.Object(doc["responsibility_map"])
		mode := source.String(responsibility["mode"])
		template := source.Object(responsibility["template"])
		var overrides source.Document
		if mode == "hybrid" {
			overrides = source.Object(responsibility["overrides"])
			declarations, prefix, scope = overrides, "/responsibility_map/overrides", Override
		}
		profile.Semantics = V036Semantics{
			Kind: "V036", Mode: mode, TemplateID: stringPointer(template["id"]), TemplateRevision: stringPointer(template["revision"]),
			RemoveComponents: stringsOf(overrides["remove_component_ids"]), RemoveArtifacts: stringsOf(overrides["remove_associated_artifact_ids"]),
			RemoveRelations: stringsOf(overrides["remove_relationship_ids"]),
		}
	}
	for i, item := range source.Array(declarations["components"]) {
		value := source.Object(item)
		attributes := []Attribute{}
		for _, key := range slices.Sorted(maps.Keys(value)) {
			switch key {
			case "id", "classification", "include", "exclude", "purpose":
				continue
			}
			attributes = append(attributes, Attribute{key, value[key]})
		}
		profile.Components = append(profile.Components, Component{
			ID: source.String(value["id"]), Classification: source.String(value["classification"]), Include: stringsOf(value["include"]),
			Exclude: stringsOf(value["exclude"]), Purpose: source.String(value["purpose"]), Scope: scope,
			Location: location(fmt.Sprintf("%s/components/%d", prefix, i)), Attributes: attributes,
		})
	}
	for i, item := range source.Array(declarations["associated_artifacts"]) {
		value := source.Object(item)
		profile.Artifacts = append(profile.Artifacts, Artifact{
			ID: source.String(value["id"]), Anchor: source.String(value["anchor"]), Include: stringsOf(value["include"]),
			Exclude: stringsOf(value["exclude"]), Purpose: source.String(value["purpose"]), Scope: scope,
			Location: location(fmt.Sprintf("%s/associated_artifacts/%d", prefix, i)),
		})
	}
	for i, item := range source.Array(declarations["relationships"]) {
		value := source.Object(item)
		profile.Relations = append(profile.Relations, Relationship{
			ID: source.String(value["id"]), From: source.String(value["from"]), To: source.String(value["to"]), Type: source.String(value["type"]),
			Scope: scope, Location: location(fmt.Sprintf("%s/relationships/%d", prefix, i)),
		})
	}
	return profile
}

func stringsOf(value any) []string {
	list := source.Array(value)
	items := make([]string, 0, len(list))
	for _, item := range list {
		items = append(items, source.String(item))
	}
	return items
}
