package releasetest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
)

func VerifyPackageJoin(root string, s release.Selection, input, platform string) (release.PackageReceipt, error) {
	output := ".artifacts/release/package/" + platform
	receipt, err := release.BuildPackageReceipt(root, s, input, output, platform)
	if err != nil {
		return receipt, err
	}
	// Match the b5 release test toolchain; neither operation publishes assets.
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
	check := `import importlib.metadata as metadata, sys
from ptsip.engine_installation import inspect
from ptsip.constants import TOOL_VERSION
if metadata.version('PTSIP') != sys.argv[1] or TOOL_VERSION != sys.argv[2]:
 raise RuntimeError('Package and Tool versions are not independently installed')
result = inspect()
if not result['all_installed'] or result['execution_authorized'] or result['release_attested']:
 raise RuntimeError('pip Package omitted engines or implicitly granted authority')
if result['distribution'] != 'SINGLE_PROJECT_PLATFORM_BUNDLES_ONLY':
 raise RuntimeError('not the integrated platform-specific PTSIP Package')
for item in result['entries']:
 try:
  metadata.distribution('ptsip-engine-' + item['engine'].lower())
 except metadata.PackageNotFoundError:
  pass
 else:
  raise RuntimeError('independent engine pip package still required')
print('pip installed complete PTSIP Package with all seven hash-verified native engines')
`
	c, err := release.ReadComposition(root)
	if err != nil {
		return receipt, err
	}
	if err := command(root, nil, python, "-c", check, c.PackagePyPI, c.Tool); err != nil {
		return receipt, fmt.Errorf("complete pip Package verification failed: %w", err)
	}
	// Native binaries must match the exact original component snapshot, not
	// merely be present inside the current Python environment.
	if err := release.VerifyInstalledComponents(receipt.Snapshot, python); err != nil {
		return receipt, err
	}
	// b5's exact-source sdist verification is required on the Linux host.
	if platform == "linux-x64" {
		source, err := verifySourcePackage(root, s, wheels, check, c.PackagePyPI, c.Tool)
		if err != nil {
			return receipt, err
		}
		receipt.SourceDist = &source
	}
	// Without both successful checks, the source and Wheel cannot be admitted.
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
