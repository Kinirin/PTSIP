package machine

import (
	"time"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type schemaRegexp struct{ expression *regexp2.Regexp }

func (r *schemaRegexp) String() string { return r.expression.String() }
func (r *schemaRegexp) MatchString(value string) bool {
	matched, err := r.expression.MatchString(value)
	return err == nil && matched
}

func compileSchemaRegexp(pattern string) (jsonschema.Regexp, error) {
	expression, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	expression.MatchTimeout = 500 * time.Millisecond
	return &schemaRegexp{expression: expression}, nil
}
