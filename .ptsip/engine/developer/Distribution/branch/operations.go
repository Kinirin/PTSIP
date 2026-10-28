package branch

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type Object = map[string]any
type OperationError struct{ Code, Message string }

func (e *OperationError) Error() string { return e.Code + ": " + e.Message }
func Fail(code, message string) error   { return &OperationError{code, message} }
func object(value any) Object           { result, _ := value.(map[string]any); return result }
func text(value any) string             { result, _ := value.(string); return result }

// Execute is the single closed command interface. It performs no policy
// discovery and remains usable without any legacy automation directories.
func Execute(args []string) (any, error) {
	positional := []string{}
	options := map[string]string{}
	for i := 0; i < len(args); i++ {
		value := args[i]
		if value == "--json" {
			continue
		}
		if strings.HasPrefix(value, "--") {
			if _, present := options[value]; present {
				return nil, Fail("DUPLICATE_OPTION", value)
			}
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "--") {
				return nil, Fail("OPTION_VALUE_REQUIRED", value)
			}
			i++
			options[value] = args[i]
		} else {
			positional = append(positional, value)
		}
	}
	if len(positional) != 2 {
		return nil, Fail("BRANCH_COMMAND_REQUIRED", "branch-control or branch-guard plus one operation required")
	}
	root := options["--repository"]
	if root == "" {
		root = options["--root"]
	}
	if root == "" {
		root = "."
	}
	if options["--root"] != "" && options["--repository"] != "" {
		return nil, Fail("AMBIGUOUS_REPOSITORY_ROOT", "supply one repository root option")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	contract, err := Load(absolute)
	if err != nil {
		return nil, err
	}
	namespace, command := positional[0], positional[1]
	registered := false
	for _, name := range contract.Commands[namespace] {
		if name == command {
			registered = true
		}
	}
	if !registered {
		return nil, Fail("UNREGISTERED_BRANCH_COMMAND", strings.Join(positional, " "))
	}
	allowed := map[string]bool{"--repository": true, "--root": true}
	fields := []string{}
	switch namespace + " " + command {
	case "branch-control inspect":
		fields = []string{"--github-repository", "--branch"}
	case "branch-control list":
		fields = []string{"--github-repository"}
	case "branch-control create", "branch-control recreate":
		fields = []string{"--github-repository", "--branch", "--approved-name", "--base-ref"}
	case "branch-guard validate":
		fields = []string{"--candidate", "--approved-name", "--authorization-source", "--request-kind", "--creation-mechanism"}
	case "branch-guard classify-existing":
		fields = []string{"--branch"}
	case "branch-guard profile-transition":
		fields = []string{"--branch", "--from-profile", "--to-profile"}
	}
	for _, field := range fields {
		allowed[field] = true
		if options[field] == "" {
			return nil, Fail("REQUIRED_BRANCH_OPTION", field)
		}
	}
	for field := range options {
		if !allowed[field] {
			return nil, Fail("UNREGISTERED_BRANCH_OPTION", field)
		}
	}
	if namespace == "branch-control" {
		return Control(absolute, command, options)
	}
	switch command {
	case "validate":
		return ValidateCreation(absolute, options["--candidate"], options["--approved-name"], options["--authorization-source"], options["--request-kind"], options["--creation-mechanism"])
	case "classify-existing":
		return ClassifyExisting(absolute, options["--branch"])
	case "profile-transition":
		return ProfileTransition(absolute, options["--branch"], options["--from-profile"], options["--to-profile"])
	}
	return nil, fmt.Errorf("unreachable branch operation")
}

func CLI(args []string, out io.Writer) int {
	result, err := Execute(args)
	code := 0
	if err != nil {
		status := "FAIL"
		if operation, ok := err.(*OperationError); ok && operation.Code == "BRANCH_MUTATION_ADMISSION_UNRESOLVED" {
			status = "UNVERIFIED"
			code = 4
		} else {
			code = 2
		}
		result = Object{"status": status, "problem": err.Error(), "execution_authorized": false}
	}
	if code == 0 && object(result)["status"] == "UNVERIFIED" {
		code = 4
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return 2
	}
	return code
}
