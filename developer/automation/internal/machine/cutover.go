package machine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (r *Repository) CutoverStatus() (Object, error) {
	contract, err := r.Read("developer/policy/contracts/go-automation-cutover.v1.json")
	if err != nil {
		return nil, err
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	for _, spec := range []struct{ ID, Section string }{
		{"MPD-CNTR-0004", "direct_root_automation_contract"},
		{"MPD-REAL-0005", "go_automation_implementation"},
		{"MPD-CHANGE-0004", "developer_legacy_retirement"},
	} {
		record, err := resolver.Policy(spec.ID)
		if err != nil {
			return nil, err
		}
		if Map(record["policy"])["status"] != "ACTIVE" || Map(record["rules"])[spec.Section] == nil {
			return nil, fmt.Errorf("cutover policy not active: %s", spec.ID)
		}
	}
	if contract["schema_version"] != "ptsip-go-automation-cutover/v1" || contract["policy_class"] != DeveloperClass {
		return nil, fmt.Errorf("cutover contract class mismatch")
	}
	inventory, err := r.Read(Text(contract["inventory_ref"]))
	if err != nil {
		return nil, err
	}
	expected := map[string]bool{}
	pending := []string{}
	implemented := []string{}
	for _, raw := range List(inventory["modules"]) {
		module := Map(raw)
		original := Text(module["python_path"])
		if expected[original] {
			return nil, fmt.Errorf("duplicate automation inventory path")
		}
		expected[original] = true
		if module["state"] == "GO_IMPLEMENTED" {
			for _, value := range List(module["go_paths"]) {
				goPath := Text(value)
				if !strings.HasPrefix(goPath, "developer/automation/") || !strings.HasSuffix(goPath, ".go") {
					return nil, fmt.Errorf("invalid Go implementation path")
				}
				if _, err := r.ReadSource(goPath); err != nil {
					return nil, err
				}
			}
			if len(List(module["go_paths"])) == 0 {
				return nil, fmt.Errorf("Go implementation requires sources")
			}
			implemented = append(implemented, original)
		} else if module["state"] == "PENDING" {
			pending = append(pending, original)
		} else {
			return nil, fmt.Errorf("unknown cutover inventory state")
		}
	}
	remaining := []string{}
	automation, err := r.Path("developer/automation")
	if err != nil {
		return nil, err
	}
	if err := filepath.WalkDir(automation, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == "__pycache__" {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(name, ".py") {
			relative, err := filepath.Rel(r.Root, name)
			if err != nil {
				return err
			}
			path := filepath.ToSlash(relative)
			if !expected[path] {
				return fmt.Errorf("unregistered Python automation file: %s", path)
			}
			remaining = append(remaining, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(remaining)
	sort.Strings(implemented)
	sort.Strings(pending)
	status := "IN_PROGRESS"
	if len(remaining) == 0 && len(pending) == 0 {
		status = "GO_SOURCE_CUTOVER_READY_FOR_FINAL_VERIFICATION"
	}
	return Object{"schema_version": "ptsip-go-automation-cutover-status/v1", "status": status,
		"implementation_complete": false, "legacy_removal_ready": false,
		"go_implemented_modules": implemented, "pending_modules": pending, "remaining_python_files": remaining,
		"projection_authority": false, "completion_contract_ref": "developer/policy/contracts/go-automation-cutover.v1.json"}, nil
}

func (r *Repository) ReadSource(input string) ([]byte, error) {
	path, err := r.Path(input)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}
