package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "github.com/Kinirin/PTSIP/developer/automation/branch"
	_ "github.com/Kinirin/PTSIP/developer/automation/release"
	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

type commandMetadata struct {
	repo       *machine.Repository
	positional []string
}

func run(arguments []string) (any, error) { return runCommand(arguments, nil) }

func runCommand(arguments []string, metadata *commandMetadata) (any, error) {
	options := map[string]string{}
	repetitions := map[string][]string{}
	positional := []string{}
	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]
		if argument == "--json" {
			continue
		}
		if strings.HasPrefix(argument, "--") {
			previous, exists := options[argument]
			if i+1 == len(arguments) || strings.HasPrefix(arguments[i+1], "--") {
				options[argument] = ""
			} else {
				i++
				options[argument] = arguments[i]
			}
			if exists {
				if len(repetitions[argument]) == 0 {
					repetitions[argument] = []string{previous}
				}
				repetitions[argument] = append(repetitions[argument], options[argument])
			}
		} else {
			positional = append(positional, argument)
		}
	}
	start := options["--repository"]
	if start == "" {
		start = options["--root"]
	}
	if start == "" {
		start = "."
	}
	repo, err := machine.Open(start)
	if err != nil {
		return nil, err
	}
	if err := repo.NormalizeCommandOptions(positional, options, repetitions); err != nil {
		return nil, err
	}
	if err := repo.AdmitCommand(positional, options); err != nil {
		return nil, err
	}
	if metadata != nil {
		metadata.repo, metadata.positional = repo, positional
	}
	if len(positional) == 0 {
		return nil, fmt.Errorf("command required: policy-resolver, root-family-entry, automation-migration")
	}
	if positional[0] == "automation-migration" && len(positional) == 2 && positional[1] == "inspect" {
		return repo.CutoverStatus()
	}
	if positional[0] == "repository-state" && positional[1] == "resolve" {
		return repo.State(options["--domain"])
	}
	if positional[0] == "planning-entry" && positional[1] == "resolve" {
		return repo.PlanningEntry(options["--branch"])
	}
	if positional[0] == "root-family-entry" && len(positional) == 2 && positional[1] == "resolve" {
		return repo.FamilyEntry(options["--policy-class"], options["--family"])
	}
	if positional[0] == "root-family-entry" && len(positional) == 3 && positional[1] == "inspect" {
		return repo.InspectFamilyID(positional[2])
	}
	if positional[0] != "policy-resolver" || len(positional) < 2 {
		return repo.DispatchOperation(positional[0], positional[1], options, positional[2:])
	}
	resolver, err := machine.NewResolver(repo)
	if err != nil {
		return nil, err
	}
	command := positional[1]
	switch command {
	case "resolve":
		if len(positional) != 2 {
			return nil, fmt.Errorf("resolve accepts only named scope and operation")
		}
		return resolver.Resolve(options["--scope"], options["--operation"])
	case "get", "explain", "rule":
		if len(positional) != 3 {
			return nil, fmt.Errorf("%s requires one registered identity", command)
		}
		if command == "get" {
			return resolver.Get(positional[2], options["--section"])
		}
		if command == "explain" {
			return resolver.Explain(positional[2])
		}
		return resolver.Rule(positional[2])
	case "validate":
		if len(positional) != 2 {
			return nil, fmt.Errorf("validate accepts no positional arguments")
		}
		if err := resolver.ValidateBindings(); err != nil {
			return nil, err
		}
		return machine.Object{"status": "PASS", "scope": "REGISTERED_DIRECT_ROOT_RESOLVER_BINDINGS"}, nil
	default:
		return nil, fmt.Errorf("unregistered policy-resolver command %q", command)
	}
}

func main() {
	metadata := &commandMetadata{}
	result, err := runCommand(os.Args[1:], metadata)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if object, ok := result.(map[string]any); ok {
		status, _ := object["status"].(string)
		code, err := metadata.repo.CommandStatusExitCode(metadata.positional, status)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if code != 0 {
			os.Exit(code)
		}
	}
}
