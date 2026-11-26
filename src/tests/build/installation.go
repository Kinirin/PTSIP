package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const installedSmoke = `
import hashlib
from importlib import metadata
import json
from pathlib import Path
import subprocess
import sys
import sysconfig
import tomllib

request = json.loads(sys.argv[1])
root = (Path(sysconfig.get_path("purelib")) / request["package"]).resolve()
assert root.is_relative_to(Path(sys.prefix).resolve()), "product resolved outside the temporary environment"
assert not list(root.rglob("*.py")), "retired Python implementation in installed package"
for name, expected in request["assets"].items():
    data = root.joinpath(name).read_bytes()
    if name not in request["exact_asset_digests"]:
        data = data.replace(b"\r\n", b"\n")
    assert hashlib.sha256(data).hexdigest() == expected, name
scripts = Path(sysconfig.get_path("scripts"))
suffix = ".exe" if sys.platform == "win32" else ""
router = scripts / (request["router"] + suffix)
assert router.is_file(), "installed Go router is missing"
with root.joinpath("VERSION").open("rb") as stream:
    tool_version = tomllib.load(stream)["tool"]["version"]
version = subprocess.check_output([str(scripts / ("ptsip" + suffix)), "--version", str(root)], text=True, timeout=10).strip()
assert version == f"PTSIP Tool {tool_version}", "installed Tool version differs from packaged VERSION"
app_version = subprocess.check_output([str(scripts / ("ptsip-app" + suffix)), "--version"], text=True, timeout=10).strip()
assert app_version == f"PTSIP App {tool_version}", "installed App version differs from packaged VERSION"
bundle = json.loads(metadata.distribution(request["package"]).read_text("ptsip-engines.json"))
assert set(bundle["engines"]) == set(request["engines"]), "installed native inventory differs from the release source"
for kind, origin in bundle["engines"].items():
    actual = hashlib.sha256(scripts.joinpath(origin["entrypoint"]).read_bytes()).hexdigest()
    assert actual == origin["binary_sha256"], kind
common_path = root.joinpath(request["common_origin"])
common = json.loads(common_path.read_bytes())
assert hashlib.sha256(common_path.with_name(common["binary"]).read_bytes()).hexdigest() == common["binary_sha256"]
for retired in (
    "ptsip_specification/spec_identity.py", "ptsip_specification/specification_binding.py",
    "migration/__init__.py", "migration/analysis", "migration/execution", "migration/planning", "migration/proposal",
):
    assert not root.joinpath(retired).exists(), f"retired Python path is present: {retired}"
index = json.loads(root.joinpath("engines/index.json").read_bytes())
operation = "discovery.profile.contract"
matches = [entry for entry in index["official_engine_entries"]
           if entry["kind"] == "DISCOVERY" and operation in entry["operations"]]
assert len(matches) == 1, "installed engine index does not uniquely register Profile discovery"
discovery = matches[0]
source_prefix = Path("src") / request["package"]
contract = json.loads(root.joinpath(Path(discovery["contract_ref"]).relative_to(source_prefix)).read_bytes())
manifest = json.loads(root.joinpath(Path(discovery["manifest_ref"]).relative_to(source_prefix)).read_bytes())
assert manifest["engine_id"] == discovery["engine_id"] == contract["engine_id"]
assert manifest["implementation_version"] == discovery["version"]
profile_request = {
    "engine_id": discovery["engine_id"], "engine_version": discovery["version"],
    "contract_id": contract["contract_id"], "contract_version": contract["version"],
    "operation_id": operation, "input": {"root": str(root)},
}
completed = subprocess.run([str(router), "engine", str(root), str(scripts), operation],
                           input=json.dumps(profile_request), encoding="utf-8", capture_output=True, timeout=15)
assert completed.returncode == 0, f"Profile discovery failed: {completed.stderr} {completed.stdout}"
profile = json.loads(completed.stdout)
assert profile["status"] == "PASS" and profile["output"]["allowed"] is True, profile
assert profile["output"]["version"] == request["profile"], "installed Profile routing differs from registered current version"
sys.stdout.buffer.write(str(common_path.with_name(common["binary"])).encode("utf-8"))
`

func verifyInstalled(file io.ReadSeeker, artifact, python, wheelhouse string, contract buildContract, assets registeredAssets, digest string) error {
	if !filepath.IsAbs(python) {
		return fmt.Errorf("isolated verification requires an absolute Python interpreter path")
	}
	if wheelhouse != "" {
		var err error
		wheelhouse, err = filepath.Abs(wheelhouse)
		if err != nil {
			return err
		}
		info, err := os.Stat(wheelhouse)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("wheelhouse must be an existing directory")
		}
	}
	directory, err := os.MkdirTemp("", "ptsip-build-verification-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	wheel := filepath.Join(directory, filepath.Base(artifact))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	copy, err := os.OpenFile(wheel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(copy, file)
	closeErr := copy.Close()
	if copyErr != nil || closeErr != nil {
		return fmt.Errorf("copy wheel into isolated environment: %v %v", copyErr, closeErr)
	}
	copy, err = os.Open(wheel)
	if err != nil {
		return err
	}
	stagedDigest, err := digestReader(copy)
	_ = copy.Close()
	if err != nil || stagedDigest != digest {
		return fmt.Errorf("wheel changed before isolated installation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	run := func(executable string, input []byte, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, executable, args...)
		cmd.Dir = directory
		cmd.Stdin = bytes.NewReader(input)
		for _, variable := range os.Environ() {
			key, _, _ := strings.Cut(variable, "=")
			switch key := strings.ToUpper(key); {
			case key == "PYTHONPATH", key == "PYTHONHOME", strings.HasPrefix(key, "PTSIP_"):
				continue
			}
			cmd.Env = append(cmd.Env, variable)
		}
		output, err := cmd.Output()
		if err != nil {
			if failure, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("isolated installed verification: %w: %s%s", err, output, failure.Stderr)
			}
			return nil, fmt.Errorf("isolated installed verification: %w", err)
		}
		return output, nil
	}
	venv := filepath.Join(directory, "venv")
	if _, err := run(python, nil, "-I", "-m", "venv", venv); err != nil {
		return err
	}
	isolated := filepath.Join(venv, "bin", "python")
	if runtime.GOOS == "windows" {
		isolated = filepath.Join(venv, "Scripts", "python.exe")
	}
	args := []string{"-I", "-m", "pip", "--isolated", "--disable-pip-version-check", "install"}
	if wheelhouse != "" {
		args = append(args, "--no-index", "--find-links", wheelhouse)
	}
	if _, err := run(isolated, nil, append(args, wheel)...); err != nil {
		return err
	}
	assetDigests := map[string]string{}
	exactAssetDigests := map[string]bool{}
	for name, digest := range assets.digests {
		relative := strings.TrimPrefix(name, assets.productPrefix)
		assetDigests[relative] = digest
		if assets.exactDigestPaths[name] {
			exactAssetDigests[relative] = true
		}
	}
	options, err := json.Marshal(map[string]any{
		"package":             contract.Installation.Package,
		"profile":             assets.activeProfile,
		"assets":              assetDigests,
		"exact_asset_digests": exactAssetDigests,
		"engines":             assets.engines,
		"router":              contract.Wheel.GoEntrypoint,
		"common_origin":       strings.TrimPrefix(contract.Wheel.CommonContractOrigin, contract.Installation.Package+"/"),
	})
	if err != nil {
		return err
	}
	binary, err := run(isolated, nil, "-I", "-c", installedSmoke, string(options))
	if err != nil {
		return err
	}
	return verifyInstalledPolicyAndPlan(string(binary), run)
}

func verifyInstalledPolicyAndPlan(binary string, run func(string, []byte, ...string) ([]byte, error)) error {
	const policy = "urn:ptsip:contract:public-structure-policy:v4"
	const plan = "urn:ptsip:contract:plan:public-structure"
	const version = "SCH.1.00"
	checks := []struct {
		name, urn, want string
		instance        map[string]any
	}{
		{
			name: "Policy schema and user version", urn: policy, want: "VALID",
			instance: map[string]any{"Policy_Identity": map[string]any{"SchemaVersion": version, "Version": "v1.2.3"}},
		},
		{
			name: "Plan with exact rule reference", urn: plan, want: "VALID",
			instance: map[string]any{
				"SchemaVersion": version,
				"work_unit":     []any{map[string]any{"id": "WU-AB-001", "task_id": []any{}}},
				"Plan_lifecycle": map[string]any{
					"status": "Draft.Proposed", "Creation_date": "2026-11-26T09:00:00+09:00", "completion_date": nil,
				},
				"policy_refs": []string{"SFP-CNTR-0009#SFP-CNTR-0009/R-02"},
			},
		},
		{
			name: "Plan with a rule outside the owning policy", urn: plan, want: "UNRESOLVED",
			instance: map[string]any{"SchemaVersion": version, "policy_refs": []string{"SFP-CNTR-0001#SFP-CNTR-0009/R-02"}},
		},
	}
	for _, check := range checks {
		request, err := json.Marshal(map[string]any{
			"format": "ptsip-common-contract/v1", "operation": "validate", "urn": check.urn, "instance": check.instance,
		})
		if err != nil {
			return err
		}
		output, err := run(binary, request)
		if err != nil {
			return err
		}
		var response struct {
			Status   string `json:"status"`
			Contract struct {
				Version string `json:"version"`
			} `json:"contract"`
		}
		if err := json.Unmarshal(output, &response); err != nil {
			return fmt.Errorf("invalid installed common contract output: %w", err)
		}
		if response.Status != check.want || response.Contract.Version != version {
			return fmt.Errorf("installed %s expected %s at %s: %s", check.name, check.want, version, output)
		}
	}
	return nil
}
