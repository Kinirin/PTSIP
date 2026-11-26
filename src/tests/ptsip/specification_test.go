package contracts_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
	specification "github.com/Kinirin/PTSIP/src/ptsip/ptsip_specification"
)

func TestSpecificationResolvesSourceAndInstalledContracts(t *testing.T) {
	root := repositoryRoot(t)
	current, err := specification.OpenSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := current.VerifyRegisteredSchemas(); err != nil {
		t.Fatal(err)
	}
	versions := current.Versions()
	if versions.RBACVersion != "ver.1.0" || versions.Schemas["RA_list"] != "SCH.1.00" ||
		versions.Schemas[versionsource.PolicyKey] != "SCH.1.00" || versions.ProfileVersion != "pp.3.00" {
		t.Fatalf("unexpected version owners: %+v", versions)
	}

	installed := t.TempDir()
	write := func(relative string, data []byte) {
		t.Helper()
		file := filepath.Join(installed, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copyFile := func(origin, relative string) {
		t.Helper()
		raw, err := os.ReadFile(origin)
		if err != nil {
			t.Fatal(err)
		}
		write(relative, raw)
	}
	copyFile(filepath.Join(root, filepath.FromSlash(versionsource.Path)), "VERSION")
	for _, relative := range []string{"schemas/config.json", "common_contract/registry/manifest.json"} {
		copyFile(filepath.Join(productRoot(t), filepath.FromSlash(relative)), relative)
	}
	for key := range versions.Schemas {
		schema, raw, err := current.RegisteredSchema(key)
		if err != nil {
			t.Fatal(err)
		}
		write("schemas/"+schema.Path, raw)
	}
	reader, err := specification.OpenInstalled(installed)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.VerifyRegisteredSchemas(); err != nil {
		t.Fatal(err)
	}
	policy, raw, err := reader.RegisteredSchema(versionsource.PolicyKey)
	if err != nil {
		t.Fatal(err)
	}
	write("schemas/"+policy.Path, append(raw, ' '))
	if err := reader.VerifyRegisteredSchemas(); err == nil {
		t.Fatal("installed specification accepted a policy resource whose digest changed")
	}
}
