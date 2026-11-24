package authoring

import (
	"encoding/json"
	"fmt"

	legacy "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/check"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
)

type Document = legacy.Document
type Validator = legacy.Validator

type Catalog struct{ *legacy.Catalog }

const BoundaryPath = entry.BoundaryPath
const FamilyDirectory = entry.FamilyDirectory
const IdentityVocabularyPath = entry.IdentityVocabularyPath
const ProfilePath = legacy.ProfilePath
const DefinitionsPath = legacy.DefinitionsPath
const AuthorizationDataPath = legacy.AuthorizationDataPath
const AuthorizationDecisionDataPath = legacy.AuthorizationDecisionDataPath

func LoadResponsibilities(root string) (*legacy.Catalog, error) {
	return legacy.LoadResponsibilities(root)
}

func object(value any) Document { doc, _ := value.(Document); return doc }

func clone(input Document) Document {
	raw, _ := json.Marshal(input)
	var result Document
	_ = json.Unmarshal(raw, &result)
	if result == nil {
		result = Document{}
	}
	return result
}

func (c *Catalog) selectedFamily(doc Document, requested string) (string, error) {
	identity, present := doc["Policy_Identity"]
	if present && object(identity) == nil {
		return "", fmt.Errorf("POLICY_IDENTITY_MUST_BE_OBJECT")
	}
	declared, declaredPresent := object(identity)["root_family"]
	if declaredPresent {
		value, ok := declared.(string)
		if !ok || value == "" {
			return "", fmt.Errorf("ROOT_FAMILY_MUST_BE_NONEMPTY_STRING")
		}
		if requested != "" && requested != value {
			return "", fmt.Errorf("ROOT_FAMILY_CONFLICT: %s/%s", requested, value)
		}
		requested = value
	}
	if requested != "" && c.Contracts[requested] == nil {
		return "", fmt.Errorf("UNKNOWN_ROOT_FAMILY: %s", requested)
	}
	return requested, nil
}

func (c *Catalog) CheckRules(v Validator, doc Document) error {
	if entry.IsUnifiedPolicy(doc) {
		_, err := check.CheckUnifiedDocument(c.Root, doc)
		return err
	}
	family, err := c.selectedFamily(doc, "")
	if err != nil {
		return err
	}
	if family == "" {
		return fmt.Errorf("DECLARED_ROOT_FAMILY_REQUIRED")
	}
	if c.Common == nil {
		return fmt.Errorf("REGISTERED_POLICY_SCHEMA_REQUIRED")
	}
	selection, err := entry.SelectPolicyDocument(c.Root, doc)
	if err != nil {
		return err
	}
	if c.Common["$id"] != selection.SchemaIDs["public"] ||
		c.Contracts[family]["$ref"] != selection.ContractRefs["family_fields"] {
		return fmt.Errorf("AUTHORING_SELECTION_MISMATCH")
	}
	// The registered public schema owns the complete document and delegates
	// Family field requirements to Policy_Rules. Authoring does not reinterpret
	// retired four-block rule bindings or claim semantic approval.
	return c.ValidatePolicySchema(v, doc)
}

func (c *Catalog) GenerateRules(v Validator, input Document, requested string) (Document, string, error) {
	doc := clone(input)
	// An unspecified empty request has no declaration to serialize. It must not
	// create a policy, a Family selection, or empty required containers.
	if len(doc) == 0 && requested == "" {
		return doc, "OMITTED", nil
	}
	family, err := c.selectedFamily(doc, requested)
	if err != nil {
		return nil, "", err
	}
	if family == "" {
		return nil, "", fmt.Errorf("DECLARED_ROOT_FAMILY_REQUIRED")
	}
	if err := c.CheckRules(v, doc); err != nil {
		return nil, "", err
	}
	return doc, "GENERATED", nil
}
