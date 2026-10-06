package pp

import "testing"

func TestProjectProfileRegistryPlaneAndCanonicalProjectionAreValid(t *testing.T) {
	binary := ppBuildAutomation(t)
	result, err, output := ppAutomation(t, binary, "profile-registry", "validate")
	if err != nil {
		t.Fatalf("profile-registry validate: %v\n%s", err, output)
	}
	if result["status"] != "PASS" {
		t.Fatalf("Project Profile registry validation failed: %#v", result)
	}
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) != 0 {
		t.Fatalf("Project Profile registry validation errors: %#v", result["errors"])
	}
}
