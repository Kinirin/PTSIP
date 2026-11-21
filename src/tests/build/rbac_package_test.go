package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSetuptoolsStagesRegisteredRBACWithoutDeveloperContracts(t *testing.T) {
	root := buildRepositoryRoot(t)
	python := filepath.Join(root, ".venv", "bin", "python")
	if runtime.GOOS == "windows" {
		python = filepath.Join(root, ".venv", "Scripts", "python.exe")
	}
	if _, err := os.Stat(python); err != nil {
		python, err = exec.LookPath("python")
		if err != nil {
			t.Fatal("Python build runtime required for package asset verification")
		}
	}
	// Extract the production RBAC hook and give it only the distributable
	// schema catalog and registered structure assets.
	const script = `
import ast, json, sys
from pathlib import Path, PurePosixPath
from shutil import rmtree

root, scratch = map(Path, sys.argv[1:])
minimal_root = scratch / "source"
schema_root = minimal_root / "src" / "ptsip" / "schemas"
source = root / "src" / "ptsip" / "schemas"
config_bytes = (source / "config.json").read_bytes()
config = json.loads(config_bytes)
registrations = config["storage"]["Responsibility-Authority"]
for row in registrations.values():
    path = row["path"]
    target = schema_root / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_bytes((source / path).read_bytes())
(schema_root / "config.json").write_bytes(config_bytes)

setup_path = root / "setup.py"
tree = ast.parse(setup_path.read_bytes(), filename=str(setup_path))
hooks = [entry for entry in tree.body if isinstance(entry, ast.FunctionDef)
         and entry.name == "_project_rbac_contract_assets"]
environment = {"ROOT": minimal_root, "Path": Path, "PurePosixPath": PurePosixPath,
               "json": json, "rmtree": rmtree}
exec(compile(ast.Module(body=hooks, type_ignores=[]), str(setup_path), "exec"), environment)
project = environment["_project_rbac_contract_assets"]

target = scratch / "build"
stale = target / "ptsip" / "specdata" / "compile-sealed-rbac" / "candidate-index.json"
stale.parent.mkdir(parents=True)
stale.write_text("{}", encoding="utf-8")
project(target)
assert not stale.parent.exists(), stale
packaged_schemas = target / "ptsip" / "schemas"
assert (packaged_schemas / "config.json").read_bytes() == config_bytes
assert {item.relative_to(packaged_schemas).as_posix()
        for item in (packaged_schemas / "storage" / "rbac").iterdir()} == {
            row["path"] for row in registrations.values()}
for row in registrations.values():
    path = row["path"]
    assert (packaged_schemas / path).read_bytes() == (source / path).read_bytes(), path

cases = (
    ("RA_list", {"$id": "urn:ptsip:other"}, "identity mismatch"),
    ("RA_list", {"path": "../.ptsip/contracts/compile-sealed-rbac/candidate-index.json"}, "registered RBAC schema path"),
    ("AEC_list", registrations["RA_list"], "registered RBAC schema path"),
)
for key, changes, reason in cases:
    mutated = json.loads(config_bytes)
    mutated["storage"]["Responsibility-Authority"][key].update(changes)
    (schema_root / "config.json").write_text(json.dumps(mutated), encoding="utf-8")
    try:
        project(scratch / "rejected")
    except RuntimeError as error:
        assert reason in str(error), (key, error)
    else:
        raise AssertionError(key + " was staged")
    assert not (scratch / "rejected").exists(), key
`
	command := exec.Command(python, "-B", "-c", script, root, t.TempDir())
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("projecting registered RBAC schemas: %v\n%s", err, output)
	}
}
