// PTSIP developer-only validator orchestrator.
// Register future validator modules here; implementation remains in their own packages.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	contract_authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/contract_authoring"
	ctrl_admission "github.com/Kinirin/PTSIP/.ptsip/engine/developer/ctrl_admission"
	release_check "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release"
	schema_check "github.com/Kinirin/PTSIP/.ptsip/engine/developer/schema_check"
)

type Validator func(string) (any, int)

func runSchemaCheck(root string) (any, int) {
	result, code := schema_check.Run(root)
	return result, code
}

func runReleaseAdmissionCheck(root string) (any, int) {
	return release_check.RunAdmission(root)
}

// runVersionDiff routes the Release responsibility through the developer
// entrypoint; no nested command package or publication admission is created.
func runVersionDiff(root, operation, base, head, component, target, digest string, explicit bool) (any, int) {
	reject := func(err error) (any, int) {
		return map[string]any{"status": "BLOCKED", "reason": err.Error(), "publication_authorized": false}, 2
	}
	var result any
	var err error
	switch operation {
	case "preview":
		result, err = release_check.PreviewVersionsDifferent(root, base, head)
	case "check":
		result, err = release_check.CheckVersionsDifferent(root, base, head)
	case "apply":
		err = release_check.ApplySelectedVersion(root, base, head, component, target, digest, explicit)
		if err == nil {
			result = map[string]any{"status": "SELECTED_VERSION_APPLIED", "component": component, "target": target, "publication_authorized": false}
		}
	default:
		err = fmt.Errorf("unsupported release version operation: %s", operation)
	}
	if err != nil {
		return reject(err)
	}
	if strings.TrimSpace(operation) == "apply" {
		return result, 0
	}
	return map[string]any{"status": "NON_NORMATIVE_VERSION_PREVIEW", "operation": operation, "report": result, "publication_authorized": false}, 0
}

var validators = map[string]Validator{
	"release-admission-check":      runReleaseAdmissionCheck,
	"schema-check":                 runSchemaCheck,
	"ctrl-release-admission-check": ctrl_admission.Run,
}

func main() {
	root := flag.String("root", ".", "repository root")
	selected := flag.String("validator", "schema-check", "registered developer validator")
	operation := flag.String("operation", "check", "authoring: check, generate, describe; release-version-diff: preview, check, apply")
	family := flag.String("family", "", "explicit Root Family for policy-rules generation or description")
	input := flag.String("input", "", "repository-relative JSON/YAML input, or - for stdin")
	output := flag.String("output", "", "new repository-relative JSON/YAML output (existing files are preserved)")
	base := flag.String("base", "", "exact baseline commit SHA for release-version-diff")
	head := flag.String("head", "", "exact checked-out commit SHA for release-version-diff")
	component := flag.String("component", "", "explicitly selected component")
	target := flag.String("target", "", "explicitly selected version")
	digest := flag.String("version-sha256", "", "exact version source SHA256")
	explicit := flag.Bool("explicit-user-decision", false, "version apply requires explicit decision")
	flag.Parse()

	names := make([]string, 0, len(validators))
	for name := range validators {
		names = append(names, name)
	}
	names = append(names, "release-version-diff")
	names = append(names, "policy-rules", "responsibility-authority")
	sort.Strings(names)
	if *selected == "policy-rules" || *selected == "responsibility-authority" {
		validate, err := schema_check.InstalledValidator()
		if err != nil {
			emit(map[string]any{"status": "FAIL", "problem": err.Error(), "execution_authorized": false})
			os.Exit(2)
		}
		report, code := contract_authoring.Run(*root, *selected, *operation, *family, *input, *output, validate)
		emit(report)
		os.Exit(code)
	}
	if *selected == "release-version-diff" {
		absolute, err := filepath.Abs(*root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		result, code := runVersionDiff(absolute, *operation, *base, *head, *component, *target, *digest, *explicit)
		emit(result)
		if code != 0 {
			os.Exit(code)
		}
		return
	}
	runner, registered := validators[*selected]
	if !registered {
		emit(map[string]any{
			"status":    "UNRESOLVED",
			"reason":    "UNREGISTERED_DEVELOPER_VALIDATOR",
			"available": names,
		})
		os.Exit(2)
	}
	absolute, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	report, code := runner(absolute)
	emit(report)
	os.Exit(code)
}

func emit(report any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
