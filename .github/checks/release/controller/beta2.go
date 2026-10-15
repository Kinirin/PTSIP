package releasecontroller

import (
	"fmt"
	"strings"
)

// This is the owner's bounded release instruction, not a successor Family policy
// resolver or a policy-validation PASS. Retired developer controls stay retired.
const Beta2AuthorizationPath = ".github/checks/release/beta2-authorization.json"

func Beta2Branch(root string) (string, error) {
	var a struct {
		Schema     string   `json:"schema_version"`
		Branch     string   `json:"branch"`
		Package    string   `json:"package_pypi_version"`
		Tool       string   `json:"tool_identity"`
		Merge      bool     `json:"merge_to_main"`
		PolicyPass bool     `json:"automated_policy_validation_available"`
		Engines    []string `json:"mandatory_engines"`
	}
	if err := ReadJSON(root, Beta2AuthorizationPath, &a); err != nil {
		return "", err
	}
	if a.Schema != "ptsip-owner-bounded-release/v1" || a.Branch != "dev/0.3.8b2" || a.Package != "0.3.8b3" || a.Tool != "T.0.3.8.b3" || a.Merge || a.PolicyPass || strings.Join(a.Engines, ",") != "context,storage,validation,discovery,planning,execution,projection" {
		return "", fmt.Errorf("owner's bounded beta2 release instruction differs")
	}
	return a.Branch, nil
}

func AdmitBeta2(root string, s Selection) error {
	if _, err := Beta2Branch(root); err != nil {
		return err
	}
	c, err := ReadComposition(root)
	if err != nil {
		return err
	}
	if c.PackagePyPI != "0.3.8b3" || c.Tool != "T.0.3.8.b3" || s.Type != "PTSIP_PACKAGE" || len(s.Components) != 9 || len(c.Engines) != 7 || strings.Join(s.Platforms, ",") != "windows-x64,linux-x64" {
		return fmt.Errorf("beta2 requires complete Tool/Profile/seven-engine Windows and Linux selection")
	}
	return ValidateSelection(root, s)
}
