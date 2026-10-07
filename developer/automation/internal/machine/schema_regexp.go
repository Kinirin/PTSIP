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
	// Parallel schema verification can suspend a valid match for more than
	// 500 ms. Keep a bounded timeout without treating scheduling delays as
	// invalid canonical policy identities.
	expression.MatchTimeout = 5 * time.Second
	return &schemaRegexp{expression: expression}, nil
}
