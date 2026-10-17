package lifecycle_test

// Shared fixture for lifecycle transition tests. Legacy version-rule tests
// were retired because version format and update semantics are undecided.
type object = map[string]any

func versionPolicy(version, status string) object {
	payload := object{
		"schema_version": "ptsip-developer-root-family-policy/v1", "policy_class": "PTSIP_DEVELOPER_POLICY", "responsibility_family": "GOV",
		"authority_subject": object{"id": "FIXTURE_SUBJECT", "statement": "Fixture version contract"},
		"exclusive_kernel":  []any{"FIXTURE_KERNEL"},
		"policy":            object{"id": "MPD-GOV-9999", "version": version, "title": "Fixture", "status": status, "substatus": "Default"},
		"rules":             object{"fixture": object{"enabled": true}},
		"relations":         object{"supersedes": []any{}, "amends": []any{}, "extends": []any{}, "depends_on": []any{}},
	}
	if status == "APPROVED" {
		payload["transition"] = object{"target_status": "APPROVED", "target_substatus": "Applying", "state": "PENDING", "requirements": []any{object{"id": "TR-001", "type": "FIXTURE", "state": "OPEN", "next_action": object{"action": "FIXTURE", "execution": "PARALLEL_ALLOWED", "completion_check": "FIXTURE_PASS"}}}}
	}
	return payload
}
