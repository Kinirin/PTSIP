package machine

import (
	"fmt"
	"os"
	"strings"
)

// Frozen approval scopes retain their original paths and hashes. The admitted
// Developer cutover contract selects explicit verified Go implementation paths;
// this mapping neither rewrites approval provenance nor admits Support authority.
func (r *Repository) VerifyAuditImplementationTarget(reference string) error {
	targets := []string{reference}
	if strings.HasPrefix(reference, "developer/automation/") && strings.HasSuffix(reference, ".py") {
		resolver, err := NewResolver(r)
		if err != nil {
			return err
		}
		owner, err := resolver.Policy("MPD-CNTR-0004")
		if err != nil {
			return err
		}
		const contractRef = "developer/policy/contracts/go-automation-cutover.v1.json"
		if Map(owner["policy"])["status"] != "ACTIVE" || Map(Map(owner["rules"])["direct_root_automation_contract"])["contract_ref"] != contractRef {
			return fmt.Errorf("Go implementation mapping contract is not active")
		}
		contract, err := r.Read(contractRef)
		if err != nil {
			return err
		}
		inventory, err := r.Read(Text(contract["inventory_ref"]))
		if err != nil {
			return err
		}
		matches := []Object{}
		for _, raw := range List(inventory["modules"]) {
			module := Map(raw)
			if module["python_path"] == reference {
				matches = append(matches, module)
			}
		}
		if len(matches) != 1 {
			return fmt.Errorf("audit implementation requires one registered cutover mapping: %s", reference)
		}
		module := matches[0]
		if module["python_entrypoint_removed"] == true {
			verified := module["equivalence_verification"] == "NATIVE_GO_REGRESSION_AND_COMMAND_SURFACE_VERIFIED" || module["equivalence_verification"] == "REGISTERED_DIRECT_ROOT_ENTRY_PROTOCOL_VERIFIED"
			if module["state"] != "GO_IMPLEMENTED" || !verified {
				return fmt.Errorf("audit implementation equivalence is not verified: %s", reference)
			}
			oldPath, err := r.Path(reference)
			if err != nil {
				return err
			}
			if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
				return fmt.Errorf("retired Python automation entrypoint is present: %s", reference)
			}
			targets = Strings(module["go_paths"])
			if len(targets) == 0 {
				return fmt.Errorf("audit implementation has no verified Go targets: %s", reference)
			}
			for _, target := range targets {
				scope, err := r.Scope(target)
				if err != nil || scope != target || !strings.HasPrefix(target, "developer/automation/") || !strings.HasSuffix(target, ".go") {
					return fmt.Errorf("audit implementation target is outside Go automation: %s", target)
				}
			}
		}
	}
	for _, target := range targets {
		path, err := r.Path(target)
		if err != nil {
			return err
		}
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("audit implementation target missing: %s", target)
		}
	}
	return nil
}
