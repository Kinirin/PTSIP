package releasetest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func VerifyPackageJoin(root string, s release.Selection, input, platform string) (release.PackageReceipt, error) {
	output := ".artifacts/release/package/" + platform
	receipt, err := release.BuildPackageReceipt(root, s, input, output, platform)
	if err != nil {
		return receipt, err
	}
	if err := command(root, nil, "python", "-m", "pip", "install", "build>=1.2,<2", "twine>=6,<8"); err != nil {
		return receipt, err
	}
	dir, err := os.MkdirTemp("", "ptsip-complete-pip-consumer-")
	if err != nil {
		return receipt, err
	}
	defer os.RemoveAll(dir)
	if err := command(root, nil, "python", "-m", "venv", dir); err != nil {
		return receipt, err
	}
	python := filepath.Join(dir, "bin/python")
	if runtime.GOOS == "windows" {
		python = filepath.Join(dir, "Scripts/python.exe")
	}
	wheels := filepath.Join(root, filepath.FromSlash(output))
	if err := command(root, nil, python, "-m", "pip", "install", "--disable-pip-version-check", "--find-links", wheels, filepath.Join(wheels, receipt.Wheel.Name)); err != nil {
		return receipt, err
	}
	check := `import importlib.metadata as metadata, importlib.resources as resources, json, os, subprocess, sys, sysconfig, yaml
from pathlib import Path
from ptsip.engine_installation import inspect
from ptsip.constants import TOOL_VERSION
if metadata.version('PTSIP') != sys.argv[1] or TOOL_VERSION != sys.argv[2]:
 raise RuntimeError('Package and Tool versions are not independently installed')
result = inspect()
if not result['all_installed'] or result['execution_authorized']:
 raise RuntimeError('pip Package omitted or implicitly activated an engine')
if result['distribution'] != 'SINGLE_PROJECT_PLATFORM_BUNDLES':
 raise RuntimeError('Package did not install a single-project native bundle')
for row in result['entries']:
 try:
  metadata.distribution('ptsip-engine-' + row['engine'].lower())
 except metadata.PackageNotFoundError:
  pass
 else:
  raise RuntimeError('Package installed a separate engine distribution')
catalog = resources.files('ptsip.profiles')
if not catalog.joinpath('index.yaml').is_file() or catalog.joinpath('history').is_dir():
 raise RuntimeError('public Profile catalog missing or history shipped')
registry = yaml.safe_load(resources.files('ptsip').joinpath('specdata/project-profile-contracts.yaml').read_text())
if registry['current'] != 'pp.1.02':
 raise RuntimeError('Profile version differs')
contract = json.loads(resources.files('ptsip').joinpath('engines/validation/contract.json').read_text())
validation = next(row for row in result['entries'] if row['engine'] == 'VALIDATION')
if validation['installed_version'] != '0.3.2' or contract['version'] != '3.1.0':
 raise RuntimeError('b4 Validation engine or capability contract differs')
binary = Path(sysconfig.get_path('scripts')) / ('ptsip-engine-validation' + ('.exe' if os.name == 'nt' else ''))
request = dict(engine_id=validation['engine_id'], engine_version=validation['installed_version'], contract_id=contract['contract_id'], contract_version=contract['version'], operation_id='validation.schema')
for schema, instance, schema_resources, expected, error in [
 ({'$ref':'urn:ptsip:b4:string'}, 'ok', [{'$id':'urn:ptsip:b4:string','type':'string'}], True, None),
 ({'$ref':'urn:ptsip:b4:string'}, 3, [{'$id':'urn:ptsip:b4:string','type':'string'}], False, None),
 ({'$ref':'#'}, {}, [], None, 'INTERNAL_REF_UNSUPPORTED'),
 ({'$dynamicRef':'#x'}, {}, [], None, 'DYNAMIC_REF_UNSUPPORTED'),
 ({'$ref':'https://example.invalid/absent'}, {}, [], None, 'SCHEMA_RESOURCE_NOT_FOUND'),
]:
 request['input'] = dict(schema=schema, instance=instance, schema_resources=schema_resources)
 process = subprocess.run([str(binary)], input=json.dumps(request), text=True, capture_output=True, timeout=30)
 response = json.loads(process.stdout)
 if error is None:
  assert process.returncode == 0 and response['status'] == 'PASS' and response['output']['valid'] == expected, response
 else:
  assert process.returncode == 2 and response['status'] == 'FAIL' and error in response['error'], response
 assert response['normative_authority'] is False and response['execution_authorized'] is False
print('Installed b4 Validation resolves explicit external IDs and explicitly rejects internal/dynamic refs')
print('pip installed complete Package and all seven hash-verified native engines')
`
	c, err := release.ReadComposition(root)
	if err != nil {
		return receipt, err
	}
	if err := command(root, nil, python, "-c", check, c.PackagePyPI, c.Tool); err != nil {
		return receipt, fmt.Errorf("complete pip Package verification failed: %w", err)
	}
	if err := release.VerifyInstalledComponents(receipt.Snapshot, python); err != nil {
		return receipt, err
	}
	if platform == "linux-x64" {
		source, err := verifySourcePackage(root, s, wheels, check, c.PackagePyPI, c.Tool)
		if err != nil {
			return receipt, err
		}
		receipt.SourceDist = &source
	}
	args := []string{"python", "-m", "twine", "check", filepath.Join(wheels, receipt.Wheel.Name)}
	if receipt.SourceDist != nil {
		args = append(args, filepath.Join(wheels, receipt.SourceDist.Name))
	}
	if err := command(root, nil, args...); err != nil {
		return receipt, err
	}
	receipt.Status = "PASS"
	if err := release.WriteJSON(filepath.Join(wheels, "package-receipt.json"), receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}
