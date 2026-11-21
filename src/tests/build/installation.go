package main

import (
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
from importlib import metadata, resources
import json
from pathlib import Path
import sys
import sysconfig

request = json.loads(sys.argv[1])
package = resources.files(request["package"])
root = Path(str(package)).resolve()
assert root.is_relative_to(Path(sys.prefix).resolve()), "product resolved outside the temporary environment"
for name, expected in request["assets"].items():
    assert hashlib.sha256(root.joinpath(name).read_bytes().replace(b"\r\n", b"\n")).hexdigest() == expected, name
scripts = Path(sysconfig.get_path("scripts"))
suffix = ".exe" if sys.platform == "win32" else ""
assert scripts.joinpath(request["router"] + suffix).is_file(), "installed Go router is missing"
bundle = json.loads(metadata.distribution(request["package"]).read_text("ptsip-engines.json"))
assert set(bundle["engines"]) == set(request["engines"]), "installed native inventory differs from the release source"
for kind, origin in bundle["engines"].items():
    actual = hashlib.sha256(scripts.joinpath(origin["entrypoint"]).read_bytes()).hexdigest()
    assert actual == origin["binary_sha256"], kind
common_path = root.joinpath(request["common_origin"])
common = json.loads(common_path.read_bytes())
assert hashlib.sha256(common_path.with_name(common["binary"]).read_bytes()).hexdigest() == common["binary_sha256"]
from ptsip.migration import analyze_source_migration
from ptsip.ptsip_specification.spec_identity import current_project_profile_version
assert callable(analyze_source_migration)
assert current_project_profile_version() == request["profile"], "installed Profile routing differs from registered current version"
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
	run := func(executable string, args ...string) error {
		cmd := exec.CommandContext(ctx, executable, args...)
		cmd.Dir = directory
		for _, variable := range os.Environ() {
			key, _, _ := strings.Cut(variable, "=")
			switch key := strings.ToUpper(key); {
			case key == "PYTHONPATH", key == "PYTHONHOME", strings.HasPrefix(key, "PTSIP_"):
				continue
			}
			cmd.Env = append(cmd.Env, variable)
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("isolated installed verification: %w: %s", err, output)
		}
		return nil
	}
	venv := filepath.Join(directory, "venv")
	if err := run(python, "-I", "-m", "venv", venv); err != nil {
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
	if err := run(isolated, append(args, wheel)...); err != nil {
		return err
	}
	assetDigests := map[string]string{}
	for name, digest := range assets.digests {
		assetDigests[strings.TrimPrefix(name, assets.productPrefix)] = digest
	}
	options, err := json.Marshal(map[string]any{
		"package":       contract.Installation.Package,
		"profile":       assets.activeProfile,
		"assets":        assetDigests,
		"engines":       assets.engines,
		"router":        contract.Wheel.GoEntrypoint,
		"common_origin": strings.TrimPrefix(contract.Wheel.CommonContractOrigin, contract.Installation.Package+"/"),
	})
	if err != nil {
		return err
	}
	return run(isolated, "-I", "-c", installedSmoke, string(options))
}
