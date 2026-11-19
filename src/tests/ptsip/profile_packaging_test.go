package contracts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProfileCanonicalPackageAssets(t *testing.T) {
	root := repositoryRoot(t)
	python := filepath.Join(root, ".venv", "bin", "python")
	if runtime.GOOS == "windows" {
		python = filepath.Join(root, ".venv", "Scripts", "python.exe")
	}
	if _, err := os.Stat(python); err != nil {
		var lookupErr error
		python, lookupErr = exec.LookPath("python")
		if lookupErr != nil {
			t.Fatal("Python build runtime required for package asset verification")
		}
	}
	target := t.TempDir()
	// Stage the declared setuptools package data, then execute the production
	// Profile hook in isolation. Other setup.py hooks retain their own owners.
	script := `
import ast, hashlib, json, sys, tomllib
from pathlib import Path
from shutil import rmtree
import yaml
from setuptools import Distribution, find_packages
from setuptools.command.build_py import build_py

root, target = map(Path, sys.argv[1:])
settings = tomllib.loads((root / "pyproject.toml").read_text(encoding="utf-8"))["tool"]["setuptools"]
source = root / "src"
discovery = settings["packages"]["find"]
packages = find_packages(where=str(source), include=discovery["include"], exclude=discovery["exclude"])
dist = Distribution({"packages": packages, "package_dir": {"": str(source)},
                     "package_data": settings["package-data"],
                     "exclude_package_data": settings["exclude-package-data"],
                     "include_package_data": False})
dist.script_name = str(root / "setup.py")
command = build_py(dist)
command.build_lib = str(target)
command.ensure_finalized()
command.compile = False
command.run()

legacy = target / "ptsip" / "profiles"
(legacy / "history" / "old").mkdir(parents=True, exist_ok=True)
(legacy / "history" / "old" / "example.ptsip.yaml").write_text("stale: true\n", encoding="utf-8")
(legacy / "example.ptsip.yaml").write_text("stale: true\n", encoding="utf-8")
(legacy / "index.yaml").write_text("stale: true\n", encoding="utf-8")

setup_path = root / "setup.py"
tree = ast.parse(setup_path.read_text(encoding="utf-8"), filename=str(setup_path))
names = {"ROOT", "EMBEDDED_PP_CONTRACT_REGISTRY"}
selected = [node for node in tree.body if
            isinstance(node, ast.Assign) and any(isinstance(t, ast.Name) and t.id in names for t in node.targets)
            or isinstance(node, ast.FunctionDef) and node.name == "_project_profile_contracts"]
namespace = {"__file__": str(setup_path), "Path": Path, "hashlib": hashlib,
             "json": json, "yaml": yaml, "rmtree": rmtree}
exec(compile(ast.Module(body=selected, type_ignores=[]), str(setup_path), "exec"), namespace)
namespace["_project_profile_contracts"](target)
print("profile package assets staged")
`
	command := exec.Command(python, "-c", script, root, target)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Profile package projection failed: %v\n%s", err, output)
	}
	product := productRoot(t)
	config := readObject(t, filepath.Join(product, "schemas", "config.json"))
	registrations := config["storage"].(object)["profile"].(object)
	for key, value := range registrations {
		t.Run(key, func(t *testing.T) {
			row := value.(object)
			relative := filepath.FromSlash(row["path"].(string))
			source := filepath.Join(product, "schemas", relative)
			packaged := filepath.Join(target, "ptsip", "schemas", relative)
			if fileHash(t, source) != fileHash(t, packaged) {
				t.Fatal("packaged registered Profile schema differs from its checked source")
			}
		})
	}
	for _, relative := range []string{"schemas/config.json", "specdata/project-profile-contracts.yaml", "profiles/identity.py"} {
		if fileHash(t, filepath.Join(product, filepath.FromSlash(relative))) != fileHash(t, filepath.Join(target, "ptsip", filepath.FromSlash(relative))) {
			t.Fatalf("package projection changed or prematurely removed %s", relative)
		}
	}
	for _, retired := range []string{"profiles/index.yaml", "profiles/example.ptsip.yaml", "profiles/history"} {
		if _, err := os.Stat(filepath.Join(target, "ptsip", filepath.FromSlash(retired))); !os.IsNotExist(err) {
			t.Fatalf("retired example/catalog distribution data remains or cannot be checked: %s: %v", retired, err)
		}
	}
}
