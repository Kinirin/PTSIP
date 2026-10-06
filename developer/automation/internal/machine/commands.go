package machine

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (r *Repository) registeredCommand(positional []string) (Object, error) {
	if len(positional) < 2 {
		return nil, fmt.Errorf("registered command and subcommand are required")
	}
	contract, err := r.Read("developer/policy/contracts/go-automation-cutover.v1.json")
	if err != nil {
		return nil, err
	}
	if contract["schema_version"] != "ptsip-go-automation-cutover/v1" || contract["policy_class"] != DeveloperClass {
		return nil, fmt.Errorf("unregistered automation contract")
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	owner, err := resolver.Policy("MPD-CNTR-0004")
	if err != nil {
		return nil, err
	}
	obligation := Map(Map(owner["rules"])["direct_root_automation_contract"])
	if Map(owner["policy"])["status"] != "ACTIVE" || obligation["contract_ref"] != "developer/policy/contracts/go-automation-cutover.v1.json" {
		return nil, fmt.Errorf("automation command contract has no exact active owner")
	}
	matches := []Object{}
	for _, raw := range List(contract["runtime_commands"]) {
		record := Map(raw)
		prefix := List(record["command"])
		if len(prefix) != 2 {
			return nil, fmt.Errorf("invalid command registration")
		}
		if prefix[0] == positional[0] && prefix[1] == positional[1] {
			matches = append(matches, record)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("unregistered or ambiguous command %s %s", positional[0], positional[1])
	}
	return matches[0], nil
}

func (r *Repository) CommandStatusExitCode(positional []string, status string) (int, error) {
	command, err := r.registeredCommand(positional)
	if err != nil {
		return 2, err
	}
	for _, value := range List(command["nonfatal_statuses"]) {
		if Text(value) == status {
			return 0, nil
		}
	}
	switch status {
	case "FAIL", "BLOCKED", "UNRESOLVED":
		return 2, nil
	default:
		return 0, nil
	}
}

// NormalizeCommandOptions admits repetition only through the selected contract.
func (r *Repository) NormalizeCommandOptions(positional []string, options map[string]string, repetitions map[string][]string) error {
	command, err := r.registeredCommand(positional)
	if err != nil {
		return err
	}
	allowed := map[string]bool{}
	for _, raw := range List(command["repeatable_options"]) {
		name := "--" + strings.ReplaceAll(Text(raw), "_", "-")
		if name == "--" {
			return fmt.Errorf("invalid registered repeatable option")
		}
		allowed[name] = true
	}
	for name := range repetitions {
		if !allowed[name] {
			return fmt.Errorf("duplicate option %s", name)
		}
	}
	for name := range allowed {
		value, present := options[name]
		if !present {
			continue
		}
		values := repetitions[name]
		if len(values) == 0 {
			values = []string{value}
		}
		for _, value := range values {
			if value == "" {
				return fmt.Errorf("option %s requires a value", name)
			}
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return err
		}
		options[name] = string(encoded)
	}
	return nil
}

// AdmitCommand uses the owner-selected closed machine vocabulary before dispatch.
func (r *Repository) AdmitCommand(positional []string, options map[string]string) error {
	command, err := r.registeredCommand(positional)
	if err != nil {
		return err
	}
	for _, raw := range List(command["boolean_options"]) {
		option := "--" + strings.ReplaceAll(Text(raw), "_", "-")
		if value, present := options[option]; present {
			if value == "" {
				options[option] = "true"
			} else if value != "true" && value != "false" {
				return fmt.Errorf("option %s requires boolean", option)
			}
		}
	}
	if len(positional) != 2+len(List(command["required_arguments"])) {
		return fmt.Errorf("registered command positional argument count mismatch")
	}
	allowed := map[string]bool{"--repository": true, "--root": true}
	for _, key := range []string{"required_options", "optional_options", "boolean_options"} {
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
		if options[option] == "" {
			return fmt.Errorf("option %s requires a value", option)
		}
	}
	return nil
}
