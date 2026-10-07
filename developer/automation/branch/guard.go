package branch

import (
	"regexp"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

func rootSection(repo *machine.Repository, id, section string) (any, error) {
	resolver, err := machine.NewResolver(repo)
	if err != nil {
		return nil, err
	}
	result, err := resolver.Get(id, section)
	if err != nil {
		return nil, err
	}
	return result["record"], nil
}

func ValidateCreation(repo *machine.Repository, candidate, approved, authorization, request, mechanism string) (machine.Object, error) {
	allowed, err := rootSection(repo, "MPD-CNTR-0001", "unit_mpd_0014_2636c6da29a8")
	if err != nil {
		return nil, err
	}
	source, err := rootSection(repo, "MPD-GOV-0001", "unit_mpd_0014_f261c3ba8099")
	if err != nil {
		return nil, err
	}
	classes, err := rootSection(repo, "MPD-GOV-0001", "unit_mpd_0014_d04f6a56972a")
	if err != nil {
		return nil, err
	}
	development := machine.Map(machine.Map(classes)["DEVELOPMENT_VERSION"])
	if mechanism != machine.Text(machine.Map(allowed)["allowed"]) {
		return nil, machine.Fail("UNAUTHORIZED_BRANCH_CREATION_MECHANISM", "branch creation must use the admitted GitHub create-ref API")
	}
	if authorization != machine.Text(source) {
		return nil, machine.Fail("BRANCH_CREATION_REQUIRES_USER_EXPLICIT", "branch creation requires the exact explicit authorization source")
	}
	if request != machine.Text(development["request_kind"]) {
		return nil, machine.Fail("UNREGISTERED_BRANCH_REQUEST_KIND", request)
	}
	if approved == "" {
		return nil, machine.Fail("APPROVED_BRANCH_NAME_REQUIRED", "exact approved name is required")
	}
	if candidate != approved {
		return nil, machine.Fail("BRANCH_NAME_NOT_EXACTLY_APPROVED", candidate+" differs from the exact approved name")
	}
	pattern, err := regexp.Compile(machine.Text(development["exact_name_pattern"]))
	if err != nil {
		return nil, err
	}
	if !pattern.MatchString(candidate) {
		return nil, machine.Fail("UNAUTHORIZED_BRANCH_NAME", "candidate does not match the registered development branch pattern")
	}
	return machine.Object{
		"status": "AUTHORIZED",
		"branch_name": candidate,
		"branch_class": "DEVELOPMENT_VERSION",
		"authorization_source": authorization,
		"request_kind": request,
		"creation_mechanism": mechanism,
	}, nil
}

func ProfileTransition(repo *machine.Repository, branch, previous, next string) (machine.Object, error) {
	classes, err := rootSection(repo, "MPD-GOV-0001", "unit_mpd_0014_d04f6a56972a")
	if err != nil {
		return nil, err
	}
	pattern, err := regexp.Compile(machine.Text(machine.Map(machine.Map(classes)["DEVELOPMENT_VERSION"])["exact_name_pattern"]))
	if err != nil {
		return nil, err
	}
	if !pattern.MatchString(branch) {
		return nil, machine.Fail("INVALID_DEVELOPMENT_BRANCH", branch)
	}
	independence, err := rootSection(repo, "MPD-CHANGE-0001", "unit_mpd_0014_103ee75460a0")
	if err != nil {
		return nil, err
	}
	profilePattern, err := regexp.Compile(machine.Text(machine.Map(independence)["project_profile_identity_pattern"]))
	if err != nil {
		return nil, err
	}
	if !profilePattern.MatchString(previous) || !profilePattern.MatchString(next) {
		return nil, machine.Fail("INVALID_PROJECT_PROFILE_IDENTITY", "invalid Project Profile identity")
	}
	return machine.Object{
		"status": "NO_BRANCH_IDENTITY_CHANGE",
		"branch_name": branch,
		"previous_profile": previous,
		"next_profile": next,
		"branch_change_required": false,
		"branch_creation_authorized": false,
	}, nil
}

func ClassifyExisting(repo *machine.Repository, branch string) (machine.Object, error) {
	classes, err := rootSection(repo, "MPD-GOV-0001", "unit_mpd_0014_d04f6a56972a")
	if err != nil {
		return nil, err
	}
	pattern, err := regexp.Compile(machine.Text(machine.Map(machine.Map(classes)["DEVELOPMENT_VERSION"])["exact_name_pattern"]))
	if err != nil {
		return nil, err
	}
	state := "UNREGISTERED_SHAPE"
	if pattern.MatchString(branch) {
		state = "AUTHORIZED_DEVELOPMENT_VERSION"
	} else {
		retention, err := rootSection(repo, "MPD-CHANGE-0001", "unit_mpd_0014_5278c1ada455")
		if err != nil {
			return nil, err
		}
		for _, raw := range machine.List(machine.Map(retention)["grandfathered_patterns"]) {
			grandfathered, err := regexp.Compile(machine.Text(raw))
			if err != nil {
				return nil, err
			}
			if grandfathered.MatchString(branch) {
				state = "GRANDFATHERED_RETENTION"
				break
			}
		}
	}
	return machine.Object{"branch_name": branch, "classification": state}, nil
}
