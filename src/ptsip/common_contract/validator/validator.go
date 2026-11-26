// Package validator performs Draft 2020-12 validation without external I/O.
package validator

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/registry"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const Draft = "https://json-schema.org/draft/2020-12/schema"

type closedLoader struct{}

func (closedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("UNREGISTERED_SCHEMA_REFERENCE: %s", url)
}

type ecmaRegexp struct{ *regexp2.Regexp }

func (pattern ecmaRegexp) MatchString(value string) bool {
	matched, err := pattern.Regexp.MatchString(value)
	return err == nil && matched
}

func compileRegexp(pattern string) (jsonschema.Regexp, error) {
	if compiled, err := regexp.Compile(pattern); err == nil {
		return compiled, nil
	}
	compiled, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	compiled.MatchTimeout = time.Second
	return ecmaRegexp{compiled}, nil
}

func Compile(g resolver.Graph) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.UseLoader(closedLoader{})
	c.UseRegexpEngine(compileRegexp)
	if _, hasPlanLifecycle := g.Docs[registry.PlanLifecycleURN]; hasPlanLifecycle {
		c.AssertFormat()
	}
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

// ValidateInstance validates the structure and any registered document-level constraints.
func ValidateInstance(urn string, schema *jsonschema.Schema, instance any) error {
	if err := schema.Validate(instance); err != nil {
		return err
	}
	if urn == registry.PlanDocumentURN {
		if err := CheckPlanWorkUnitPrefixes(instance); err != nil {
			return err
		}
	}
	return nil
}

type PlanWorkUnitPrefixError struct {
	Index int
}

func (err *PlanWorkUnitPrefixError) Error() string {
	return fmt.Sprintf("work_unit[%d].id must use the same two-letter prefix as the first work unit", err.Index)
}

// CheckPlanWorkUnitPrefixes compares IDs after a Plan passes schema validation.
func CheckPlanWorkUnitPrefixes(instance any) *PlanWorkUnitPrefixError {
	const prefixStart = len("WU-")
	const prefixEnd = prefixStart + 2
	document, _ := instance.(map[string]any)
	units, _ := document["work_unit"].([]any)
	var prefix string
	for index, raw := range units {
		unit, _ := raw.(map[string]any)
		id, _ := unit["id"].(string)
		if len(id) < prefixEnd {
			continue
		}
		if prefix == "" {
			prefix = id[prefixStart:prefixEnd]
		} else if id[prefixStart:prefixEnd] != prefix {
			return &PlanWorkUnitPrefixError{Index: index}
		}
	}
	return nil
}
