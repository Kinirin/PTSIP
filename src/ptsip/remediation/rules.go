package remediation

import "encoding/json"

type RuleCapabilityState string

const (
	Supported   RuleCapabilityState = "SUPPORTED"
	Unsupported RuleCapabilityState = "UNSUPPORTED"
)

type RuleRequirementKind string

const (
	ProjectIntent RuleRequirementKind = "PROJECT_INTENT"
	ExternalFact  RuleRequirementKind = "EXTERNAL_FACT"
)

type RuleAutomationLevel string

const (
	EvaluationOnly      RuleAutomationLevel = "EVALUATION_ONLY"
	SemanticRemediation RuleAutomationLevel = "SEMANTIC_REMEDIATION"
)

type RuleEvaluationStatus string

const (
	NotApplicable RuleEvaluationStatus = "NOT_APPLICABLE"
	Conformant    RuleEvaluationStatus = "CONFORMANT"
	Violation     RuleEvaluationStatus = "VIOLATION"
	Unresolved    RuleEvaluationStatus = "UNRESOLVED"
)

type OperationalRuleCapabilityDeclaration struct {
	SupportedRemediationFamilies []string              `json:"supported_remediation_families"`
	CandidateGeneration          RuleCapabilityState   `json:"candidate_generation"`
	PostconditionVerification    RuleCapabilityState   `json:"postcondition_verification"`
	PossibleRequirements         []RuleRequirementKind `json:"possible_requirements"`
	AutomationLevel              RuleAutomationLevel   `json:"automation_level"`
}

func (v OperationalRuleCapabilityDeclaration) Validate() error {
	if err := requireIDs("OPERATIONAL_RULE", "supported_remediation_families", v.SupportedRemediationFamilies, false); err != nil {
		return err
	}
	if v.CandidateGeneration != Supported && v.CandidateGeneration != Unsupported {
		return contractError("OPERATIONAL_RULE_CANDIDATE_CAPABILITY_INVALID", "candidate-generation state is invalid")
	}
	if v.PostconditionVerification != Supported && v.PostconditionVerification != Unsupported {
		return contractError("OPERATIONAL_RULE_VERIFICATION_CAPABILITY_INVALID", "postcondition-verification state is invalid")
	}
	seen := make(map[RuleRequirementKind]bool, len(v.PossibleRequirements))
	for _, kind := range v.PossibleRequirements {
		if kind != ProjectIntent && kind != ExternalFact {
			return contractError("OPERATIONAL_RULE_REQUIREMENTS_INVALID", "unknown requirement kind")
		}
		if seen[kind] {
			return contractError("OPERATIONAL_RULE_REQUIREMENTS_DUPLICATE", "requirement kinds must be unique")
		}
		seen[kind] = true
	}
	if v.AutomationLevel != EvaluationOnly && v.AutomationLevel != SemanticRemediation {
		return contractError("OPERATIONAL_RULE_AUTOMATION_LEVEL_INVALID", "unknown automation level")
	}
	if (v.CandidateGeneration == Supported) != (len(v.SupportedRemediationFamilies) != 0) {
		return contractError("OPERATIONAL_RULE_REMEDIATION_CAPABILITY_INCONSISTENT", "candidate-generation support and remediation families must agree")
	}
	if v.AutomationLevel == SemanticRemediation && v.CandidateGeneration != Supported {
		return contractError("OPERATIONAL_RULE_AUTOMATION_CAPABILITY_INCONSISTENT", "semantic remediation requires candidate-generation capability")
	}
	return nil
}

func (v OperationalRuleCapabilityDeclaration) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	v.SupportedRemediationFamilies = array(v.SupportedRemediationFamilies)
	v.PossibleRequirements = array(v.PossibleRequirements)
	type capability OperationalRuleCapabilityDeclaration
	return json.Marshal(struct {
		capability
		Authoritative bool `json:"authoritative"`
	}{capability(v), false})
}

type OperationalRuleBinding struct {
	RuleID        string                               `json:"rule_id"`
	Specification SpecificationBinding                 `json:"specification"`
	Capability    OperationalRuleCapabilityDeclaration `json:"capability"`
}

func (v OperationalRuleBinding) Validate() error {
	if err := requireText("OPERATIONAL_RULE", "rule_id", v.RuleID); err != nil {
		return err
	}
	if err := v.Specification.Validate(); err != nil {
		return err
	}
	return v.Capability.Validate()
}

type OperationalRuleEvaluation struct {
	RuleID     string               `json:"rule_id"`
	Status     RuleEvaluationStatus `json:"status"`
	FactIDs    []string             `json:"fact_ids"`
	Constraint *NormativeConstraint `json:"constraint"`
	Detail     *string              `json:"detail"`
}

func (v OperationalRuleEvaluation) Validate() error {
	if err := requireText("OPERATIONAL_RULE", "rule_id", v.RuleID); err != nil {
		return err
	}
	if err := requireIDs("OPERATIONAL_RULE", "fact_ids", v.FactIDs, false); err != nil {
		return err
	}
	switch v.Status {
	case Conformant, Violation:
		if v.Constraint == nil {
			return contractError("OPERATIONAL_RULE_CONSTRAINT_REQUIRED", "conformant or violating evaluations require their bound constraint")
		}
	case NotApplicable:
		if v.Constraint != nil {
			return contractError("OPERATIONAL_RULE_NOT_APPLICABLE_CONSTRAINT_CONFLICT", "not-applicable evaluation cannot claim an applicable constraint")
		}
	case Unresolved:
	default:
		return contractError("OPERATIONAL_RULE_EVALUATION_STATUS_INVALID", "unknown evaluation status")
	}
	if v.Constraint != nil {
		if v.RuleID != v.Constraint.RuleID {
			return contractError("OPERATIONAL_RULE_CONSTRAINT_RULE_ID_MISMATCH", "constraint rule ID differs from the evaluation")
		}
		if err := v.Constraint.Validate(); err != nil {
			return err
		}
	}
	return optionalText("OPERATIONAL_RULE", "detail", v.Detail)
}

func (v OperationalRuleEvaluation) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	type evaluation OperationalRuleEvaluation
	v.FactIDs = array(v.FactIDs)
	return json.Marshal(evaluation(v))
}

// OperationalRule is the read-only interface for a single explicitly bound
// specification rule. The caller obtains implementations from its rule owner.
type OperationalRule interface {
	Binding() OperationalRuleBinding
	Applies(Context, []DerivedFact) (bool, error)
	Evaluate(Context, []DerivedFact) (OperationalRuleEvaluation, error)
	ProposeSolutions(Context, OperationalRuleEvaluation, []DerivedFact) ([]SemanticCandidate, error)
	Verify(Context, SemanticRemediationPlan) (PostconditionResult, error)
}
