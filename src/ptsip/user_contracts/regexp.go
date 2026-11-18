package usercontracts

import (
	"time"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type schemaRegexp struct {
	regexp *regexp2.Regexp
}

func (pattern schemaRegexp) MatchString(value string) bool {
	matched, err := pattern.regexp.MatchString(value)
	return err == nil && matched
}

func (pattern schemaRegexp) String() string {
	return pattern.regexp.String()
}

func compileRegexp(pattern string) (jsonschema.Regexp, error) {
	compiled, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	compiled.MatchTimeout = time.Second
	return schemaRegexp{regexp: compiled}, nil
}
