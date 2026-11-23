package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestVerifierReadsRegisteredContractAtExactSource(t *testing.T) {
	root := t.TempDir()
	checkout := buildRepositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(checkout, filepath.FromSlash(releaseIndex)))
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]json.RawMessage
	if err := decode(raw, &index); err != nil {
		t.Fatal(err)
	}
	expected := sha(index["build_verification"])
	for name, data := range map[string][]byte{
		releaseIndex:                raw,
		"pyproject.toml":            []byte("[build-system]\nbackend-path = [\"src/build\"]\n"),
		"src/build/ptsip_pep517.py": []byte("# product build helper\n"),
		"src/policy/index.yaml":     []byte("policies: []\n"),
	} {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	distributionGit(t, root, "init", "--template=")
	distributionGit(t, root, "config", "core.autocrlf", "false")
	distributionGit(t, root, "add", "--", ".")
	distributionGit(t, root, "commit", "-m", "register fixture verifier")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(releaseIndex)), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := openSource(root)
	if err != nil {
		t.Fatal(err)
	}
	contract, name, observed, err := loadContract(source)
	if err != nil || name != buildVerificationReference || observed != expected || contract.Root != "src/tests/build" {
		t.Fatalf("exact release contract selection failed: %s %s %v", name, observed, err)
	}
	if problems := prebuildProblems(source, contract); len(problems) != 0 {
		t.Fatalf("registered inputs outside package staging were not recognized: %v", problems)
	}
	contract.Prebuild.RequiredFiles = append(contract.Prebuild.RequiredFiles, ".ptsip/contracts/compile-sealed-rbac/candidate-index.json")
	if problems := prebuildProblems(source, contract); len(problems) != 1 || !strings.Contains(problems[0], "outside the product and canonical projection roots") {
		t.Fatalf("unowned source input was not rejected: %v", problems)
	}
}

func TestCommonRegistryPinsPolicyAndPlanWithoutDuplicateAssets(t *testing.T) {
	root := t.TempDir()
	const product = "src/ptsip/"
	const registry = product + "common_contract/registry/manifest.json"
	const policy = "schemas/policy/policy.schema.json"
	const plan = "schemas/plan/plan.schema.json"
	policyBytes := []byte("{\"type\":\"object\"}\n")
	planBytes := []byte("{\"type\":\"array\"}\n")
	resource := func(name, digest string) map[string]string {
		return map[string]string{"resource": name, "sha256": digest}
	}
	manifest := map[string]any{
		"format": "ptsip-common-contract-registry/v1", "revision": "v2",
		"contracts":   []any{resource(policy, sha(policyBytes))},
		"diagnostics": []any{resource(policy, sha(policyBytes)), resource(plan, sha(planBytes))},
	}
	write := func(name string, data []byte) {
		t.Helper()
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(product+policy, policyBytes)
	write(product+plan, planBytes)
	distributionGit(t, root, "init", "--template=")
	distributionGit(t, root, "config", "core.autocrlf", "false")
	load := func() (registeredAssets, error) {
		t.Helper()
		write(registry, mustJSON(t, manifest))
		distributionGit(t, root, "add", "--", ".")
		distributionGit(t, root, "commit", "-m", "register common assets")
		source, err := openSource(root)
		if err != nil {
			t.Fatal(err)
		}
		assets := registeredAssets{productPrefix: product, digests: map[string]string{}}
		err = assets.loadCommonAssets(source, registry)
		return assets, err
	}
	assets, err := load()
	if err != nil || len(assets.digests) != 3 || len(assets.exactDigestPaths) != 3 ||
		assets.digests[product+policy] != sha(policyBytes) || assets.digests[product+plan] != sha(planBytes) ||
		assets.digests[registry] != assets.commonRegistryDigest {
		t.Fatalf("common assets did not resolve with exact deduplicated pins: %+v %v", assets, err)
	}
	manifest["diagnostics"] = []any{resource(policy, sha(planBytes)), resource(plan, sha(planBytes))}
	if _, err := load(); err == nil || !strings.Contains(err.Error(), "conflicting common-contract resource digests") {
		t.Fatalf("conflicting repeated resource pin was accepted: %v", err)
	}
	manifest["diagnostics"] = []any{resource(policy, sha(policyBytes)), resource(plan, sha(policyBytes))}
	if _, err := load(); err == nil || !strings.Contains(err.Error(), "common-contract resource digest mismatch") {
		t.Fatalf("incorrect diagnostic digest was accepted: %v", err)
	}
}

func TestBuildArtifactVerification(t *testing.T) {
	revision := strings.Repeat("a", 40)
	registration := `{"storage":{"Responsibility-Authority":{}}}` + "\n"
	const exactFile = "src/ptsip/schemas/plan/plan.schema.json"
	exact := []byte("{\"type\":\"object\"}\n")
	exactCRLF := bytes.ReplaceAll(exact, []byte("\n"), []byte("\r\n"))
	assets := registeredAssets{productPrefix: "src/ptsip/", digests: map[string]string{"src/ptsip/schemas/config.json": sha([]byte(registration)), exactFile: sha(exact)},
		exactDigestPaths: map[string]bool{exactFile: true},
		engines:          map[string]engineEntry{"validation": {Kind: "VALIDATION", ID: "registered-validation", Version: "1.0.0"}}, commonRegistryDigest: strings.Repeat("b", 64)}
	var rules buildContract
	rules.SDist.SourceDescriptor = "ptsip-build-source.json"
	rules.SDist.ExcludedPrefixes = []string{".ptsip/contracts/compile-sealed-rbac/", "src/ptsip/compile_sealed_rbac/"}
	rules.Wheel.EngineOriginSuffix = ".dist-info/ptsip-engines.json"
	rules.Wheel.CommonContractOrigin = "ptsip/common_contract/origin.json"
	rules.Wheel.GoEntrypoint = "ptsip-go"
	rules.Wheel.ExcludedPrefixes = []string{"ptsip/specdata/compile-sealed-rbac/", "ptsip/compile_sealed_rbac/"}
	descriptor := mustJSON(t, map[string]string{"schema_version": "ptsip-build-source/v1", "source_sha": revision})
	source := map[string][]byte{
		"ptsip-source/ptsip-build-source.json":       descriptor,
		"ptsip-source/src/ptsip/schemas/config.json": []byte(strings.ReplaceAll(registration, "\n", "\r\n")),
		"ptsip-source/" + exactFile:                  exact,
	}
	if err := verifySDist(bytes.NewReader(tarFixture(t, source)), revision, rules.SDist, assets); err != nil {
		t.Fatal(err)
	}
	if err := verifySDist(bytes.NewReader(tarFixture(t, source)), strings.Repeat("c", 40), rules.SDist, assets); err == nil {
		t.Fatal("sdist with a different source revision passed verification")
	}
	source["ptsip-source/"+exactFile] = exactCRLF
	if err := verifySDist(bytes.NewReader(tarFixture(t, source)), revision, rules.SDist, assets); err == nil || !strings.Contains(err.Error(), "sdist asset digest mismatch") {
		t.Fatalf("sdist with normalized common-contract bytes was accepted: %v", err)
	}
	source["ptsip-source/"+exactFile] = exact
	source["ptsip-source/.ptsip/contracts/compile-sealed-rbac/candidate-index.json"] = []byte("{}")
	if err := verifySDist(bytes.NewReader(tarFixture(t, source)), revision, rules.SDist, assets); err == nil {
		t.Fatal("sdist contains a developer candidate index")
	}
	delete(source, "ptsip-source/.ptsip/contracts/compile-sealed-rbac/candidate-index.json")
	source["ptsip-source/src/ptsip/compile_sealed_rbac/testdata/conformance.json"] = []byte("[]")
	if err := verifySDist(bytes.NewReader(tarFixture(t, source)), revision, rules.SDist, assets); err == nil {
		t.Fatal("sdist contains retired RBAC fixture assets")
	}

	const dist = "ptsip-1.0.0"
	const scripts = dist + ".data/scripts/"
	const origin = dist + ".dist-info/ptsip-engines.json"
	const record = dist + ".dist-info/RECORD"
	common := []byte("common-contract fixture binary")
	engine := []byte("validation fixture binary")
	payload := map[string][]byte{
		"ptsip/schemas/config.json":                   []byte(registration),
		"ptsip/schemas/plan/plan.schema.json":         exact,
		"ptsip/common_contract/ptsip-common-contract": common,
		scripts + "ptsip-engine-validation":           engine,
		scripts + "ptsip-go":                          []byte("router fixture binary"),
		scripts + "ptsip-app":                         []byte("app fixture binary"),
		scripts + "ptsip":                             []byte("router fixture binary"),
		dist + ".dist-info/METADATA":                  []byte("Name: PTSIP\nVersion: 1.0.0\n"),
		dist + ".dist-info/WHEEL":                     []byte("Wheel-Version: 1.0\n"),
	}
	payload[rules.Wheel.CommonContractOrigin] = mustJSON(t, map[string]string{
		"format": "ptsip-common-contract-wheel-origin/v1", "source_sha": revision, "binary": "ptsip-common-contract",
		"binary_sha256": sha(common), "registry_sha256": assets.commonRegistryDigest,
	})
	payload[origin] = mustJSON(t, map[string]any{
		"schema_version": "ptsip-bundled-engine-origins/v1", "platform": "linux-x64", "execution_authorized": false,
		"engines": map[string]any{"validation": map[string]any{
			"schema_version": "ptsip-independent-wheel-origin/v1", "source_sha": revision, "engine_id": "registered-validation",
			"engine_version": "1.0.0", "entrypoint": "ptsip-engine-validation", "binary_sha256": sha(engine), "execution_authorized": false,
		}},
	})
	check := func(raw []byte) error {
		return verifyWheel(bytes.NewReader(raw), int64(len(raw)), revision, rules, assets, nil)
	}
	if err := check(zipFixture(t, payload, record, "")); err != nil {
		t.Fatal(err)
	}
	payload["ptsip/schemas/plan/plan.schema.json"] = exactCRLF
	if err := check(zipFixture(t, payload, record, "")); err == nil || !strings.Contains(err.Error(), "wheel registered asset digest mismatch") {
		t.Fatalf("wheel with normalized common-contract bytes was accepted: %v", err)
	}
	payload["ptsip/schemas/plan/plan.schema.json"] = exact
	payload["ptsip/compile_sealed_rbac/pp.3.00.contract.json"] = []byte("{}")
	if err := check(zipFixture(t, payload, record, "")); err == nil || !strings.Contains(err.Error(), "excluded wheel member") {
		t.Fatalf("wheel with retired RBAC assets was accepted: %v", err)
	}
	delete(payload, "ptsip/compile_sealed_rbac/pp.3.00.contract.json")
	payload["ptsip/cli.py"] = []byte("retired CLI")
	if err := check(zipFixture(t, payload, record, "")); err == nil || !strings.Contains(err.Error(), "excluded wheel member") {
		t.Fatalf("wheel with a retired Python module was accepted: %v", err)
	}
	delete(payload, "ptsip/cli.py")
	if err := check(zipFixture(t, payload, record, scripts+"ptsip-engine-validation")); err == nil {
		t.Fatal("wheel with tampered engine bytes passed RECORD verification")
	}
	delete(payload, scripts+"ptsip-go")
	if err := check(zipFixture(t, payload, record, "")); err == nil || !strings.Contains(err.Error(), "router") {
		t.Fatalf("wheel without the runtime router was accepted: %v", err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func tarFixture(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var result bytes.Buffer
	compressor := gzip.NewWriter(&result)
	archive := tar.NewWriter(compressor)
	for name, body := range files {
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressor.Close(); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}

func zipFixture(t *testing.T, files map[string][]byte, record, tamper string) []byte {
	t.Helper()
	var result, records bytes.Buffer
	archive := zip.NewWriter(&result)
	writer := csv.NewWriter(&records)
	for name, body := range files {
		digest := sha256.Sum256(body)
		if err := writer.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(digest[:]), strconv.Itoa(len(body))}); err != nil {
			t.Fatal(err)
		}
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if name == tamper {
			body = bytes.Repeat([]byte("x"), len(body))
		}
		if _, err := entry.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Write([]string{record, "", ""}); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	entry, err := archive.Create(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(records.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}
