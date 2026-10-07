package release

import (
	"os"
	"path/filepath"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

func Build(repo *machine.Repository) (machine.Object, error) {
	output, code, err := run(repo, "python", "-m", "build")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, machine.Fail("RELEASE_BUILD_FAILED", output)
	}
	return machine.Object{"status": "PASS", "operation": "BUILD_DISTRIBUTIONS"}, nil
}

func VerifyDistributionMetadata(repo *machine.Repository) (machine.Object, error) {
	dist, err := repo.Path("dist")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dist)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dist, entry.Name()))
	}
	if len(files) == 0 {
		return nil, machine.Fail("RELEASE_DISTRIBUTION_MISSING", "dist contains no files")
	}
	args := append([]string{"-m", "twine", "check"}, files...)
	output, code, err := run(repo, "python", args...)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, machine.Fail("RELEASE_DISTRIBUTION_METADATA_INVALID", output)
	}
	return machine.Object{"status": "PASS", "operation": "VERIFY_DISTRIBUTION_METADATA", "file_count": len(files)}, nil
}
