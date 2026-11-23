package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGoStagesRegisteredRBACWithoutDeveloperContracts(t *testing.T) {
	root := buildRepositoryRoot(t)
	fixture := t.TempDir()
	stage := func(relative string, raw []byte) {
		t.Helper()
		target := filepath.Join(fixture, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copyAsset := func(relative string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		stage(relative, raw)
		return raw
	}
	const schemaRoot = "src/ptsip/schemas/"
	const config = schemaRoot + "config.json"
	configBytes := copyAsset(config)
	type fields map[string]string
	type storageCatalog struct {
		Storage map[string]map[string]fields `json:"storage"`
	}
	var catalog storageCatalog
	if err := json.Unmarshal(configBytes, &catalog); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{}
	for _, group := range []string{"profile", "Responsibility-Authority"} {
		for _, binding := range catalog.Storage[group] {
			relative := schemaRoot + binding["path"]
			expected[strings.TrimPrefix(relative, "src/")] = sha(copyAsset(relative))
		}
	}
	expected["ptsip/VERSION"] = sha(copyAsset("src/VERSION"))
	const candidate = "ptsip/schemas/history/profile/ptsip-profile-pp-2.00.schema.json"
	expected[candidate] = sha(copyAsset("src/" + candidate))
	stage("src/policy/index.yaml", []byte("schema_version: fixture\n"))
	stage("src/policy/ARCH/SFP-fixture.yaml", []byte("schema_version: fixture\n"))
	stage("src/policy/schemas/schema.json", []byte("{}"))
	stage("src/policy/registries/registry.json", []byte("{}"))

	executable := filepath.Join(t.TempDir(), "ptsip-build")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	build := exec.Command("go", "-C", filepath.Join(root, "src", "build"), "build", "-mod=readonly", "-o", executable, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile Go builder: %v\n%s", err, output)
	}
	runAssets := func() ([]byte, error) {
		return exec.Command(executable, "assets", "-root", fixture).CombinedOutput()
	}
	output, err := runAssets()
	if err != nil {
		t.Fatalf("stage registration from product files: %v\n%s", err, output)
	}
	var got map[string]string
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatal(err)
	}
	for name, digest := range expected {
		if got[name] != digest {
			t.Errorf("staged bytes differ for %s", name)
		}
	}
	for _, test := range []struct {
		key, field, value, problem string
	}{
		{"RA_list", "$id", "urn:ptsip:other", "identity mismatch"},
		{"RA_list", "path", "../.ptsip/contracts/compile-sealed-rbac/candidate-index.json", "registered RBAC schema path"},
		{"AEC_list", "path", catalog.Storage["Responsibility-Authority"]["RA_list"]["path"], "registered RBAC schema path"},
	} {
		if err := json.Unmarshal(configBytes, &catalog); err != nil {
			t.Fatal(err)
		}
		catalog.Storage["Responsibility-Authority"][test.key][test.field] = test.value
		stage(config, mustJSON(t, catalog))
		output, err := runAssets()
		if err == nil || !strings.Contains(string(output), test.problem) {
			t.Errorf("staged invalid %s %s: %v\n%s", test.key, test.field, err, output)
		}
	}
}
