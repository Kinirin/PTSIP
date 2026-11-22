package profileverify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
)

const (
	profileContractPath = ".ptsip/contracts/release/Platform/Common/profile/project-profile-contracts.yaml"
	profileRegistryPath = release.ProfileConfigPath
	profileStorageRoot  = "src/ptsip/schemas/"
)

type closedProfileSchemaLoader struct{}

func (closedProfileSchemaLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("PROFILE_EXTERNAL_SCHEMA_FORBIDDEN: %s", url)
}

// VerifyProjectProfiles checks the release version and the registered Profile
// schema bytes without consulting the historical Profile archive as a source.
func VerifyProjectProfiles(root, expectedVersion string) error {
	if err := verifyRouting(root); err != nil {
		return err
	}
	registrations, active, err := release.ReadProfileRegistrations(root)
	if err != nil {
		return err
	}
	if expectedVersion == "" || active != expectedVersion {
		return fmt.Errorf("PROFILE_RELEASE_VERSION_MISMATCH: expected %s, registered %s", expectedVersion, active)
	}
	sourceRoot, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer sourceRoot.Close()
	for _, row := range registrations {
		reference := path.Join(profileStorageRoot, row.Path)
		if err := verifyRegisteredSchema(sourceRoot, reference, row); err != nil {
			return err
		}
	}
	return nil
}

func verifyRegisteredSchema(root *os.Root, reference string, row release.ProfileRegistration) error {
	raw, err := root.ReadFile(filepath.FromSlash(reference))
	if err != nil {
		return fmt.Errorf("PROFILE_SCHEMA_MISSING: %s: %w", reference, err)
	}
	// Git may check out canonical LF JSON with CRLF on Windows.
	if release.Digest(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))) != row.Digest {
		return fmt.Errorf("PROFILE_SCHEMA_DIGEST_MISMATCH: %s", reference)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return fmt.Errorf("PROFILE_SCHEMA_INVALID: %s: %w", reference, err)
	}
	if document["$id"] != row.ID {
		return fmt.Errorf("PROFILE_SCHEMA_ID_INVALID: %s", reference)
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(closedProfileSchemaLoader{})
	if err := compiler.AddResource(row.ID, document); err != nil {
		return fmt.Errorf("PROFILE_SCHEMA_INVALID: %s: %w", reference, err)
	}
	if _, err := compiler.Compile(row.ID); err != nil {
		return fmt.Errorf("PROFILE_SCHEMA_INVALID: %s: %w", reference, err)
	}
	return nil
}
