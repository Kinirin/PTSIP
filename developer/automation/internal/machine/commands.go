package machine

import (
	"fmt"
	"strings"
)

// AdmitCommand uses the owner-selected closed machine vocabulary before dispatch.
func (r *Repository) AdmitCommand(positional []string, options map[string]string) error {
	if len(positional) < 2 {
		return fmt.Errorf("registered command and subcommand are required")
	}
	contract, err := r.Read("developer/policy/contracts/go-automation-cutover.v1.json")
	if err != nil {
		return err
	}
	if contract["schema_version"] != "ptsip-go-automation-cutover/v1" || contract["policy_class"] != DeveloperClass {
		return fmt.Errorf("unregistered automation contract")
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return err
	}
	owner, err := resolver.Policy("MPD-CNTR-0004")
	if err != nil {
		return err
	}
	obligation := Map(Map(owner["rules"])["direct_root_automation_contract"])
	if Map(owner["policy"])["status"] != "ACTIVE" || obligation["contract_ref"] != "developer/policy/contracts/go-automation-cutover.v1.json" {
		return fmt.Errorf("automation command contract has no exact active owner")
	}
	matches := []Object{}
	for _, raw := range List(contract["runtime_commands"]) {
		record := Map(raw)
		prefix := List(record["command"])
		if len(prefix) != 2 {
			return fmt.Errorf("invalid command registration")
		}
		if prefix[0] == positional[0] && prefix[1] == positional[1] {
			matches = append(matches, record)
		}
	}
	if len(matches) != 1 {
		return fmt.Errorf("unregistered or ambiguous command %s %s", positional[0], positional[1])
	}
	command := matches[0]
	if len(positional) != 2+len(List(command["required_arguments"])) {
		return fmt.Errorf("registered command positional argument count mismatch")
	}
	allowed := map[string]bool{"--repository": true, "--root": true}
	for _, key := range []string{"required_options", "optional_options"} {
		for _, raw := range List(command[key]) {
			option := "--" + strings.ReplaceAll(Text(raw), "_", "-")
			if option == "--" {
				return fmt.Errorf("invalid registered option")
			}
			allowed[option] = true
			if key == "required_options" && strings.TrimSpace(options[option]) == "" {
				return fmt.Errorf("required registered option %s is missing", option)
			}
		}
	}
	for option := range options {
		if !allowed[option] {
			return fmt.Errorf("option %s is not registered for this command", option)
		}
	}
	return nil
}
