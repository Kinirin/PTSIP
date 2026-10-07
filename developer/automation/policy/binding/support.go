package binding

import domainpolicy "github.com/Kinirin/PTSIP/developer/automation/policy"

type Object = domainpolicy.Object

type Repository interface {
	Read(string) (Object, error)
	Validate(string, any) error
	Path(string) (string, error)
	Scope(string) (string, error)
}

const DeveloperClass = domainpolicy.DeveloperClass

var rootID = domainpolicy.RootID

func Map(value any) Object  { return domainpolicy.Map(value) }
func Text(value any) string { return domainpolicy.Text(value) }
func List(value any) []any  { return domainpolicy.List(value) }
func Strings(value any) []string { return domainpolicy.Strings(value) }
