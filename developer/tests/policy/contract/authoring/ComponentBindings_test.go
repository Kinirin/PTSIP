package contract_authoring_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/authoring"
)

func TestUnregisteredDeveloperProfileCannotCreateOrGrantAuthority(t *testing.T) {
	root := repository(t)
	const name = "must-not-create-retired-profile.yaml"
	validate := func(any, any, []any) (bool, error) {
		t.Fatal("validator was called before resolving the responsibility owner")
		return false, nil
	}
	for _, operation := range []string{"check", "generate", "lookup-intention"} {
		t.Run(operation, func(t *testing.T) {
			input, output := "", ""
			if operation == "generate" {
				output = name
			}
			if operation == "lookup-intention" {
				input = "AUTH-UNREGISTERED"
			}
			raw, code := authoring.Run(root, "responsibility-authority", operation, "", input, output, validate)
			result, ok := raw.(map[string]any)
			if !ok || code == 0 || result["execution_authorized"] != false ||
				!strings.Contains(result["problem"].(string), "LOCAL_RESOURCE_UNAVAILABLE") {
				t.Fatalf("unregistered developer authority route was accepted: %v", raw)
			}
		})
	}
	if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
		t.Fatalf("unregistered profile generation wrote a file: %v", err)
	}
}
