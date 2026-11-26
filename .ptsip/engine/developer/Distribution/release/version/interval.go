package version

import (
	"fmt"
	"math/big"
	"strings"
)

type VersionChangeClass uint8

const (
	VersionUnchanged VersionChangeClass = iota
	VersionPatch
	VersionMinor
	VersionMajor
)

// PublicationIntervalClass retains the greatest established change since the
// last completed distribution-platform publication. The affected contract
// owner supplies the approved numeric grammar and transition.
func PublicationIntervalClass(applied VersionChangeClass, changes ...VersionChangeClass) (VersionChangeClass, error) {
	if applied > VersionMajor {
		return 0, fmt.Errorf("unknown applied version change class: %d", applied)
	}
	for _, change := range changes {
		if change > VersionMajor {
			return 0, fmt.Errorf("unknown version change class: %d", change)
		}
		applied = max(applied, change)
	}
	return applied, nil
}

type EngineVersionTrigger string

const (
	ContractOnlyChange                   EngineVersionTrigger = "CONTRACT_ONLY_CHANGE"
	StandaloneResponsibilityDeletion     EngineVersionTrigger = "STANDALONE_RESPONSIBILITY_DELETION"
	FullEngineRemoval                    EngineVersionTrigger = "FULL_ENGINE_REMOVAL"
	ExistingEngineFunctionChange         EngineVersionTrigger = "EXISTING_ENGINE_FUNCTION_CHANGE"
	NewEngineFunctionAddition            EngineVersionTrigger = "NEW_ENGINE_FUNCTION_ADDITION"
	ResponsibilityTransferOut            EngineVersionTrigger = "RESPONSIBILITY_TRANSFER_OUT"
	ResponsibilityTransferIn             EngineVersionTrigger = "RESPONSIBILITY_TRANSFER_IN"
	NewDomainWithRelatedContract         EngineVersionTrigger = "NEW_DOMAIN_RESPONSIBILITY_ADDITION_WITH_RELATED_CONTRACT_CREATED"
	NewDomainWithoutRelatedContract      EngineVersionTrigger = "NEW_DOMAIN_RESPONSIBILITY_DECLARATION_WITHOUT_RELATED_CONTRACT"
	ResponsibilityScopeChangeWithCode    EngineVersionTrigger = "EXISTING_RESPONSIBILITY_EXPANSION_OR_REDUCTION_WITH_CODE_SOURCE_CHANGE"
	ResponsibilityScopeChangeWithoutCode EngineVersionTrigger = "EXISTING_RESPONSIBILITY_EXPANSION_OR_REDUCTION_WITHOUT_CODE_SOURCE_CHANGE"
	ResponsibilityCompatibilityBreak     EngineVersionTrigger = "RESPONSIBILITY_COMPATIBILITY_BREAK"
	FunctionCompatibilityBreak           EngineVersionTrigger = "FUNCTION_COMPATIBILITY_BREAK"
)

// SuggestEngineVersion calculates an engine's proposed version from its last
// version confirmed published by the distribution platform. The caller supplies
// that evidence and established change triggers. The result never edits the
// source version and still requires the applicable exact target approval.
func SuggestEngineVersion(lastPublished, current string, triggers ...EngineVersionTrigger) (string, error) {
	if !engineVersionRE.MatchString(lastPublished) || !engineVersionRE.MatchString(current) {
		return "", fmt.Errorf("numeric engine versions and a last published version are required")
	}
	base := strings.Split(lastPublished, ".")
	increment := func(number string) string {
		value, _ := new(big.Int).SetString(number, 10)
		return value.Add(value, big.NewInt(1)).String()
	}
	versions := [...]string{
		VersionUnchanged: lastPublished,
		VersionPatch:     strings.Join([]string{base[0], base[1], increment(base[2])}, "."),
		VersionMinor:     strings.Join([]string{base[0], increment(base[1]), "0"}, "."),
		VersionMajor:     strings.Join([]string{increment(base[0]), "0", "0"}, "."),
	}
	var applied VersionChangeClass
	for index, value := range versions {
		if value == current {
			applied = VersionChangeClass(index)
			break
		}
	}
	if versions[applied] != current {
		return "", fmt.Errorf("engine version %s does not match the publication interval from %s", current, lastPublished)
	}
	var next VersionChangeClass
	retired := false
	for _, trigger := range triggers {
		switch trigger {
		case FullEngineRemoval:
			retired = true
		case ContractOnlyChange, StandaloneResponsibilityDeletion, ExistingEngineFunctionChange, ResponsibilityScopeChangeWithCode:
			next = max(next, VersionPatch)
		case NewEngineFunctionAddition, ResponsibilityTransferOut, ResponsibilityTransferIn, NewDomainWithRelatedContract, FunctionCompatibilityBreak:
			next = max(next, VersionMinor)
		case ResponsibilityCompatibilityBreak:
			next = VersionMajor
		case NewDomainWithoutRelatedContract, ResponsibilityScopeChangeWithoutCode:
		default:
			return "", fmt.Errorf("unknown engine version trigger: %q", trigger)
		}
	}
	if retired {
		return current, nil
	}
	class, err := PublicationIntervalClass(applied, next)
	if err != nil {
		return "", err
	}
	return versions[class], nil
}

type RBACSystemVersionTrigger string

const (
	RBACListFieldAddition       RBACSystemVersionTrigger = "RBAC_LIST_FIELD_ADDITION"
	RBACOrdinaryChange          RBACSystemVersionTrigger = "ORDINARY_RBAC_CHANGE"
	RBACInferenceFunctionChange RBACSystemVersionTrigger = "SUBSTANTIAL_CHANGE_TO_RBAC_INFERENCE_FUNCTIONALITY"
)

// ClassifyRBACSystemVersion returns the approved automatic classification.
// Substantial inference changes need a direct user decision and version edit.
func ClassifyRBACSystemVersion(triggers ...RBACSystemVersionTrigger) (VersionChangeClass, error) {
	class := VersionUnchanged
	for _, trigger := range triggers {
		switch trigger {
		case RBACListFieldAddition, RBACOrdinaryChange:
			class = VersionMinor
		case RBACInferenceFunctionChange:
			return VersionUnchanged, fmt.Errorf("RBAC_MAJOR_USER_DECISION_AND_INCREMENT_REQUIRED")
		default:
			return VersionUnchanged, fmt.Errorf("unknown RBAC system version trigger: %q", trigger)
		}
	}
	return class, nil
}
