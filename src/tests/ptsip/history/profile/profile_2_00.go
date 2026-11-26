package profile

import "github.com/santhosh-tekuri/jsonschema/v6"

func Validate200(contract *jsonschema.Schema, instance map[string]any) error {
	return validate("pp.2.00", contract, instance)
}
