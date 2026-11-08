// Package validator performs Draft 2020-12 validation without external I/O.
package validator

import (
	"fmt"
	"sort"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const Draft = "https://json-schema.org/draft/2020-12/schema"

type closedLoader struct{}

func (closedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("UNREGISTERED_SCHEMA_REFERENCE: %s", url)
}

func Compile(g resolver.Graph) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.UseLoader(closedLoader{})
	ids := make([]string, 0, len(g.Docs))
	for id := range g.Docs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		doc := g.Docs[id]
		if draft, ok := doc["$schema"]; ok && draft != Draft {
			return nil, fmt.Errorf("UNSUPPORTED_SCHEMA_DRAFT: %s", id)
		}
		if err := c.AddResource(id, doc); err != nil {
			return nil, err
		}
	}
	return c.Compile(g.Root)
}
