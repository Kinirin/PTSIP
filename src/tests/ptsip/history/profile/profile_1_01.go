package profile

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed index.json
var fixturesJSON []byte

type Fixture struct {
	SourcePath string          `json:"source_path"`
	SourceBlob string          `json:"source_blob"`
	SourceSize int             `json:"source_size"`
	Version    string          `json:"version"`
	Instance   json.RawMessage `json:"instance"`
}

func Fixtures() (string, []Fixture, error) {
	var index struct {
		Revision string    `json:"source_revision"`
		Fixtures []Fixture `json:"fixtures"`
	}
	err := json.Unmarshal(fixturesJSON, &index)
	return index.Revision, index.Fixtures, err
}

func Validate101(contract *jsonschema.Schema, instance map[string]any) error {
	return validate("pp.1.01", contract, instance)
}

func validate(version string, contract *jsonschema.Schema, instance map[string]any) error {
	identity, _ := instance["ptsip"].(map[string]any)
	if identity["version"] != version {
		return fmt.Errorf("expected Profile version %s, got %v", version, identity["version"])
	}
	return contract.Validate(instance)
}
