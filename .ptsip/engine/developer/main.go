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

	branch "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/branch"
	release_check "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release"
	ctrl_admission "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/admission"
	releasenote "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/releasenote"
	release_version "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/version"
	responsibilityauthority "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Responsibility-Authority/verification"
	contract_authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/layout"
	schema_check "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
	contractentry "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractentry"
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
		result, err = release_version.PreviewVersionsDifferent(root, base, head)
	case "check":
		result, err = release_version.CheckVersionsDifferent(root, base, head)
	case "apply":
		err = release_version.ApplySelectedVersion(root, base, head, component, target, digest, explicit)
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
	"developer-layout-check":       layout.Run,
	"branch-check":                 branch.Run,
	"release-admission-check":      runReleaseAdmissionCheck,
	"schema-check":                 runSchemaCheck,
	"ctrl-release-admission-check": ctrl_admission.Run,
}

func main() {
	root := flag.String("root", ".", "repository root")
	selected := flag.String("validator", "schema-check", "registered developer validator")
	operation := flag.String("operation", "check", "authoring: check, generate, describe; release-version-diff: preview, check, apply; release-documentation: check, preview, generate")
	family := flag.String("family", "", "explicit Root Family for policy-rules generation or description")
	input := flag.String("input", "", "repository-relative JSON/YAML input, or - for stdin")
	dryRun := flag.Bool("dry-run", false, "resolve contract entry without invoking validation")
	output := flag.String("output", "", "new repository-relative JSON/YAML output (existing files are preserved)")
	base := flag.String("base", "", "exact baseline commit SHA for version comparison and documentation")
	head := flag.String("head", "", "exact checked-out commit SHA for version comparison and documentation")
	component := flag.String("component", "", "explicitly selected component")
	target := flag.String("target", "", "version target or exact developer-policy path for contract-entry")
	digest := flag.String("version-sha256", "", "exact version source SHA256")
	explicit := flag.Bool("explicit-user-decision", false, "version apply requires explicit decision")
	flag.Parse()

	names := make([]string, 0, len(validators))
	for name := range validators {
		names = append(names, name)
	}
	names = append(names, "contract-entry", "release-version-diff")
	names = append(names, "release-documentation")
	names = append(names, "policy-rules", "responsibility-authority")
	names = append(names, "policy-check", "policy-identity-check", "policy-status-check", "plan-status-check")
	sort.Strings(names)
	if *selected == "contract-entry" {
		if *operation != "check" || *target == "" || *input != "" || *output != "" || *family != "" {
			emit(map[string]any{"status": "UNRESOLVED", "problem": "CONTRACT_ENTRY_REQUIRES_EXACT_TARGET_ONLY", "execution_authorized": false, "mutation_authorized": false})
			os.Exit(2)
		}
		absolute, err := filepath.Abs(*root)
		if err != nil {
			emit(map[string]any{"status": "UNRESOLVED", "problem": err.Error(), "execution_authorized": false, "mutation_authorized": false})
			os.Exit(2)
		}
		selection, err := contractentry.Resolve(absolute, *target)
		if err != nil {
			emit(map[string]any{"status": "UNRESOLVED", "problem": err.Error(), "execution_authorized": false, "mutation_authorized": false})
			os.Exit(2)
		}
		if *dryRun {
			emit(map[string]any{"status": "SELECTED_NOT_EXECUTED", "selection": selection, "execution_authorized": false, "mutation_authorized": false})
			return
		}
		verified, code := schema_check.RunDocument(absolute, selection.Validator, selection.Target)
		status := "CHECK_COMPLETED"
		if code != 0 { status = "CHECK_FAILED" }
		emit(map[string]any{"status": status, "selection": selection, "result": verified, "execution_authorized": false, "mutation_authorized": false})
		if code != 0 { os.Exit(code) }
		return
	}
	if *selected == "release-documentation" {
		if *output != "" || *family != "" {
			emit(map[string]any{"status": "FAIL", "problem": "DOCUMENTATION_USES_CONTRACT_OUTPUT_PATHS", "publication_authorized": false})
			os.Exit(2)
		}
		validate, err := schema_check.InstalledValidator()
		if err != nil {
			emit(map[string]any{"status": "FAIL", "problem": err.Error(), "publication_authorized": false})
			os.Exit(2)
		}
		report, code := releasenote.Run(*root, *operation, *base, *head, *input, validate)
		emit(report)
		os.Exit(code)
	}
	if *selected == "policy-check" || *selected == "policy-identity-check" || *selected == "policy-status-check" || *selected == "plan-status-check" {
		if *operation != "check" || *output != "" || *family != "" {
			emit(map[string]any{"status": "FAIL", "problem": "POLICY_VERIFICATION_REQUIRES_CHECK_WITH_INPUT", "execution_authorized": false})
			os.Exit(2)
		}
		report, code := schema_check.RunDocument(*root, *selected, *input)
		emit(report)
		os.Exit(code)
	}
	if *selected == "policy-rules" || *selected == "responsibility-authority" {
		validate, err := schema_check.InstalledValidator()
		if err != nil {
			emit(map[string]any{"status": "FAIL", "problem": err.Error(), "execution_authorized": false})
			os.Exit(2)
		}
		var report any
		var code int
		if *selected == "responsibility-authority" {
			if *family != "" {
				emit(map[string]any{"status": "FAIL", "problem": "FAMILY_ONLY_APPLIES_TO_POLICY_RULES", "execution_authorized": false})
				os.Exit(2)
			}
			report, code = responsibilityauthority.Run(*root, *operation, *input, *output, validate)
		} else {
			report, code = contract_authoring.Run(*root, *selected, *operation, *family, *input, *output, validate)
		}
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
