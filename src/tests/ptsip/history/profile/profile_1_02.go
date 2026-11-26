package profile

import "github.com/santhosh-tekuri/jsonschema/v6"

func Validate102(contract *jsonschema.Schema, instance map[string]any) error {
	return validate("pp.1.02", contract, instance)
}
