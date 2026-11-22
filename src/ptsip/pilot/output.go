package pilot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/internal/localstate"
)

// WriteReport persists previously collected evidence. When destination is
// empty, the retained default lives under the caller's PTSIP state directory.
func WriteReport(report *Report, destination string) (string, error) {
	if report == nil || report.repositoryRoot == "" || report.Format != "ptsip-pilot-report/v2" {
		return "", fmt.Errorf("a completed Pilot report is required")
	}
	if destination == "" {
		directory, err := localstate.Directory(report.repositoryRoot, "pilots")
		if err != nil {
			return "", err
		}
		destination = filepath.Join(directory, "report.json")
	}
	absolute, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(absolute)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", err
	}
	parent, err = filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	output := filepath.Join(parent, filepath.Base(absolute))
	info, err := os.Lstat(output)
	if err == nil && !info.Mode().IsRegular() {
		return "", fmt.Errorf("Pilot output must be a regular file: %s", output)
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	stored := *report
	stored.NonIntrusion.Status = report.observationStatus()
	stored.NonIntrusion.OutputInsideRepository = false
	relative, err := filepath.Rel(report.repositoryRoot, output)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		stored.NonIntrusion.Status = "OUTPUT_PATH_INSIDE_REPOSITORY"
		stored.NonIntrusion.OutputInsideRepository = true
	}
	file, err := os.CreateTemp(parent, ".ptsip-pilot-*.json")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	encodeErr := encoder.Encode(stored)
	closeErr := file.Close()
	if encodeErr != nil {
		return "", encodeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err := os.Rename(file.Name(), output); err != nil {
		return "", err
	}
	*report = stored
	return output, nil
}
