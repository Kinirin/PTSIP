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
if not result['all_installed'] or result['execution_authorized']:
 raise RuntimeError('pip Package omitted or implicitly activated an engine')
print('pip installed complete Package and all seven hash-verified native engines')
`
	c, err := release.ReadComposition(root)
	if err != nil {
		return receipt, err
	}
	if err := command(root, nil, python, "-c", check, c.PackagePyPI, c.Tool); err != nil {
		return receipt, fmt.Errorf("complete pip Package verification failed: %w", err)
	}
	receipt.Status = "PASS"
	if err := release.WriteJSON(filepath.Join(wheels, "package-receipt.json"), receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}
