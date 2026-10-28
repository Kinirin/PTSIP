package branch

import (
	"encoding/json"
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const IndexPath = ".ptsip/engine/developer/Distribution/branch/index.json"

type Contract struct {
	Schema         string              `json:"schema_version"`
	Creation       string              `json:"creation_admission"`
	Classification string              `json:"classification_authority"`
	Transition     string              `json:"profile_transition_authority"`
	Naming         string              `json:"naming_validation"`
	Commands       map[string][]string `json:"registered_commands"`
	Mutation       string              `json:"mutation_policy"`
	Legacy         bool                `json:"legacy_fallback"`
}

func Load(root string) (Contract, error) {
	var index struct {
		Schema    string   `json:"schema_version"`
		State     string   `json:"state"`
		Authority string   `json:"policy_authority"`
		Execution *bool    `json:"execution_authorized"`
		Contract  Contract `json:"contract"`
	}
	filename, err := authoring.LocalFile(root, IndexPath)
	if err != nil {
		return Contract{}, err
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		return Contract{}, err
	}
	if err = json.Unmarshal(raw, &index); err != nil {
		return Contract{}, err
	}
	c := index.Contract
	if index.Schema != "ptsip-developer-validator-domain/v1" || index.State != "STRUCTURE_ONLY" || index.Authority != "NONE" || index.Execution == nil || *index.Execution ||
		c.Schema != "ptsip-branch-control/v1" || c.Creation != "UNRESOLVED" || c.Classification != "UNRESOLVED" || c.Transition != "UNRESOLVED" || c.Naming != "GIT_CHECK_REF_FORMAT" || c.Legacy || c.Mutation != "BLOCK_UNTIL_SUCCESSOR_AUTHORITY_IS_DEFINED" {
		return c, Fail("BRANCH_CONTRACT_INVALID", "temporary authoring contract cannot grant branch authority")
	}
	expected := map[string]string{"branch-control": "commands,list,inspect,create,recreate", "branch-guard": "validate,classify-existing,profile-transition"}
	if len(c.Commands) != len(expected) {
		return c, Fail("BRANCH_COMMAND_CONTRACT_INVALID", "unexpected command namespace")
	}
	for namespace, names := range expected {
		if strings.Join(c.Commands[namespace], ",") != names {
			return c, Fail("BRANCH_COMMAND_CONTRACT_INVALID", namespace)
		}
	}
	return c, nil
}

func CheckName(root, name string) error {
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") || strings.Contains(name, "@{") {
		return Fail("INVALID_BRANCH_NAME", name)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	command := exec.Command("git", "check-ref-format", "--branch", name)
	command.Dir = absolute
	if output, err := command.CombinedOutput(); err != nil {
		return Fail("INVALID_BRANCH_NAME", strings.TrimSpace(string(output)))
	}
	return nil
}

// ValidateCreation checks the explicit request shape. It does not turn a
// caller-supplied authorization label into admitted creation authority.
func ValidateCreation(root, candidate, approved, authorization, request, mechanism string) (Object, error) {
	if _, err := Load(root); err != nil {
		return nil, err
	}
	if approved == "" {
		return nil, Fail("APPROVED_BRANCH_NAME_REQUIRED", "exact requested name is required")
	}
	if candidate != approved {
		return nil, Fail("BRANCH_NAME_NOT_EXACTLY_APPROVED", candidate)
	}
	if err := CheckName(root, candidate); err != nil {
		return nil, err
	}
	if authorization != "USER_EXPLICIT" {
		return nil, Fail("BRANCH_CREATION_REQUIRES_USER_EXPLICIT", "inferred source is not an explicit request")
	}
	if mechanism != "GITHUB_CREATE_BRANCH_API" {
		return nil, Fail("UNAUTHORIZED_BRANCH_CREATION_MECHANISM", mechanism)
	}
	if request != "DEVELOPMENT_VERSION_BRANCH" {
		return nil, Fail("UNREGISTERED_BRANCH_REQUEST_KIND", request)
	}
	return Object{"status": "STRUCTURE_VALID", "branch_name": candidate, "authorization_status": "UNVERIFIED", "scope": "REQUEST_STRUCTURE_ONLY", "branch_creation_authorized": false, "execution_authorized": false}, nil
}

func ClassifyExisting(root, name string) (Object, error) {
	if _, err := Load(root); err != nil {
		return nil, err
	}
	if err := CheckName(root, name); err != nil {
		return nil, err
	}
	return Object{"status": "UNVERIFIED", "branch_name": name, "reference_syntax": "VALID", "classification": "UNVERIFIED", "branch_creation_authorized": false}, nil
}

func ProfileTransition(root, name, previous, next string) (Object, error) {
	if _, err := Load(root); err != nil {
		return nil, err
	}
	if err := CheckName(root, name); err != nil {
		return nil, err
	}
	if previous == "" || next == "" {
		return nil, Fail("PROFILE_IDENTITIES_REQUIRED", "previous and next identities are required")
	}
	return Object{"status": "UNVERIFIED", "branch_name": name, "previous_profile": previous, "next_profile": next, "transition_authority": "UNRESOLVED", "branch_creation_authorized": false}, nil
}

func Run(root string) (any, int) {
	if _, err := Load(root); err != nil {
		return Object{"status": "FAIL", "problem": err.Error(), "execution_authorized": false}, 2
	}
	return Object{"status": "VALID", "scope": "BRANCH_MODULE_CONTRACT", "creation_admission": "UNRESOLVED", "execution_authorized": false}, 0
}
