package branch

import "testing"

func TestValidateMinimalCreationInput(t *testing.T) {
	cases := []struct {
		name, input string
		valid       bool
	}{
		{"custom name", `{"branch_name":"feature/my-custom-branch","base_branch":"main"}`, true},
		{"preexisting name", `{"branch_name":"codex/validation-b4","base_branch":"dev/0.3.8b2"}`, true},
		{"user extra field", `{"branch_name":"MY/task","base_branch":"main","custom_value":"provided-by-user"}`, true},
		{"missing branch name", `{"base_branch":"main"}`, false},
		{"missing base branch", `{"branch_name":"feature/a"}`, false},
		{"blank value", `{"branch_name":" ","base_branch":"main"}`, false},
		{"incorrect type", `{"branch_name":25,"base_branch":"main"}`, false},
		{"not an object", `[]`, false},
		{"trailing object", `{"branch_name":"ok","base_branch":"main"}{}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateMinimalCreationInput([]byte(tc.input)); (err == nil) != tc.valid {
				t.Fatalf("unexpected result, valid=%v, err=%v", tc.valid, err)
			}
		})
	}
}

func TestValidateMinimalBranchRoster(t *testing.T) {
	cases := []struct {
		name, input string
		valid       bool
	}{
		{"empty roster", `{"schema_version":"ptsip-branch-roster/v1","active_branches":[],"exception_branches":[]}`, true},
		{"arbitrary names", `{"schema_version":"ptsip-branch-roster/v1","active_branches":["feature/solar","dev/0.3.8b2"],"exception_branches":["codex/wip"]}`, true},
		{"overlap unresolved", `{"schema_version":"ptsip-branch-roster/v1","active_branches":["feature/a"],"exception_branches":["feature/a"]}`, true},
		{"duplicate", `{"schema_version":"ptsip-branch-roster/v1","active_branches":["feature/a","feature/a"],"exception_branches":[]}`, false},
		{"bad version", `{"schema_version":"something","active_branches":[],"exception_branches":[]}`, false},
		{"missing group", `{"schema_version":"ptsip-branch-roster/v1","active_branches":[]}`, false},
		{"null group", `{"schema_version":"ptsip-branch-roster/v1","active_branches":null,"exception_branches":[]}`, false},
		{"empty name", `{"schema_version":"ptsip-branch-roster/v1","active_branches":[""],"exception_branches":[]}`, false},
		{"invalid name type", `{"schema_version":"ptsip-branch-roster/v1","active_branches":[4],"exception_branches":[]}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateMinimalBranchRoster([]byte(tc.input)); (err == nil) != tc.valid {
				t.Fatalf("unexpected result, valid=%v, err=%v", tc.valid, err)
			}
		})
	}
}
