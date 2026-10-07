package machine

import (
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func (r *Repository) ValidateResourceSet(schemaPath string, value any, resources []string) error {
	compiler := jsonschema.NewCompiler()
	compiler.UseRegexpEngine(compileSchemaRegexp)
	compiler.UseLoader(closedLoader{})
	for _, path := range UniqueStrings(append(resources, schemaPath)) {
		document, err := r.Read(path)
		if err != nil {
			return err
		}
		id := Text(document["$id"])
		if id == "" {
			return Fail("SCHEMA_ID_REQUIRED", path)
		}
		if err := compiler.AddResource(id, document); err != nil {
			return err
		}
	}
	schema, err := r.Read(schemaPath)
	if err != nil {
		return err
	}
	compiled, err := compiler.Compile(Text(schema["$id"]))
	if err != nil {
		return err
	}
	return compiled.Validate(value)
}
