package lifecycle

import domainpolicy "github.com/Kinirin/PTSIP/developer/automation/policy"

type Object = domainpolicy.Object
type OperationError = domainpolicy.OperationError

type Repository interface {
	Read(string) (Object, error)
	Validate(string, any) error
	Path(string) (string, error)
	Scope(string) (string, error)
	LoadNeutralPolicyIndex() (Object, error)
	WriteYAML(string, any, *string) error
	AtomicWrite(string, []byte, *string) error
}

const DeveloperClass = domainpolicy.DeveloperClass

var rootID = domainpolicy.RootID

func Map(value any) Object  { return domainpolicy.Map(value) }
func Text(value any) string { return domainpolicy.Text(value) }
func List(value any) []any  { return domainpolicy.List(value) }
func SHA256(data []byte) string { return domainpolicy.SHA256(data) }
