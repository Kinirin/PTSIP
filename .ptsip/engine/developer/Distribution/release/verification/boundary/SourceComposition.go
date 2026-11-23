package boundary

import (
	"fmt"
	"os"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
)

const (
	releaseIndexPath          = ".ptsip/contracts/release/index.json"
	CompositionPath           = platformRoot + "Common/source-composition.json"
	CompositionSchema         = ".ptsip/contracts/schemas/release/source-composition.schema.json"
	CompositionVersion        = "ptsip-release-source-composition/v1"
	compositionSchemaID       = "urn:ptsip:release:source-composition:v1"
	historicalCompositionPath = platformRoot + "Common/composition-contract.json"
)

func repositoryJSON(root, ref string) (map[string]any, []byte, error) {
	file, err := authoring.LocalFile(root, ref)
	if err != nil {
		return nil, nil, err
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", ref, err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("%s must contain a JSON object", ref)
	}
	return object, raw, nil
}

// ReadSourceComposition validates the single registered Common release route
// and its source composition. It returns verified structure only.
func ReadSourceComposition(root string) ([]byte, error) {
	catalog, _, err := repositoryJSON(root, releaseIndexPath)
	if err != nil {
		return nil, err
	}
	routes, _ := catalog["platforms"].([]any)
	count := 0
	for _, value := range routes {
		route, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid release platform route")
		}
		if route["platform"] != "Common" {
			continue
		}
		count++
		contracts, _ := route["contracts"].(map[string]any)
		if route["root"] != platformRoot+"Common/" || contracts["composition"] != CompositionPath {
			return nil, fmt.Errorf("unregistered release source composition route")
		}
	}
	if catalog["platform_dimension"] != "DISTRIBUTION_SERVICE" || count != 1 {
		return nil, fmt.Errorf("missing or ambiguous Common release source composition route")
	}
	schema, _, err := repositoryJSON(root, CompositionSchema)
	if err != nil {
		return nil, err
	}
	if schema["$id"] != compositionSchemaID || schema["$schema"] != validator.Draft {
		return nil, fmt.Errorf("invalid release source composition schema identity")
	}
	compiled, err := validator.Compile(resolver.Graph{
		Root: compositionSchemaID,
		Docs: map[string]map[string]any{compositionSchemaID: schema},
	})
	if err != nil {
		return nil, fmt.Errorf("compile release source composition schema: %w", err)
	}
	composition, raw, err := repositoryJSON(root, CompositionPath)
	if err != nil {
		return nil, err
	}
	if err := compiled.Validate(composition); err != nil {
		return nil, fmt.Errorf("release source composition: %w", err)
	}
	if composition["schema_version"] != CompositionVersion || composition["scope"] != "STRUCTURAL_RELEASE_VERIFICATION" {
		return nil, fmt.Errorf("invalid release source composition identity or scope")
	}
	return raw, nil
}

// ValidateNativeWheelReceipt validates the receipt against the definition in
// the registered source composition schema.
func ValidateNativeWheelReceipt(root string, raw []byte) error {
	schema, _, err := repositoryJSON(root, CompositionSchema)
	if err != nil {
		return err
	}
	if schema["$id"] != compositionSchemaID || schema["$schema"] != validator.Draft {
		return fmt.Errorf("invalid release source composition schema identity")
	}
	compiled, err := validator.Compile(resolver.Graph{
		Root: compositionSchemaID + "#/$defs/artifactReceipt",
		Docs: map[string]map[string]any{compositionSchemaID: schema},
	})
	if err != nil {
		return fmt.Errorf("compile native wheel receipt definition: %w", err)
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return fmt.Errorf("decode native wheel receipt: %w", err)
	}
	if err := compiled.Validate(value); err != nil {
		return fmt.Errorf("invalid native wheel receipt: %w", err)
	}
	return nil
}
