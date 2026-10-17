// Package policy loads current policy-domain vocabulary from the declared
// developer contract identity sources. It does not determine policy authority,
// routing completeness, document structure, or normative conformance.
package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const routingContract = "developer/contracts/routing/policy_routes.json"

// Domain contains only the currently declared identity vocabularies.
// Its membership methods do not grant policy or execution authority.
type Domain struct {
	families []string
	kinds    []string
	family   map[string]struct{}
	kind     map[string]struct{}
}

// Load reads the routing contract's declared identity sources. It never
// imports historical policy code or guesses policy routes from filenames.
func Load(root string) (*Domain, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("repository root: %w", err)
	}
	raw, err := os.ReadFile(filepath.Join(absolute, filepath.FromSlash(routingContract)))
	if err != nil {
		return nil, fmt.Errorf("routing contract unavailable: %w", err)
	}
	var routing struct {
		IdentitySources map[string]string `json:"identity_sources"`
	}
	if err := json.Unmarshal(raw, &routing); err != nil {
		return nil, fmt.Errorf("routing contract malformed: %w", err)
	}
	if len(routing.IdentitySources) == 0 {
		return nil, errors.New("routing contract identity_sources not declared")
	}
	families, familySet, err := readVocabulary(absolute, routing.IdentitySources["root_family"])
	if err != nil {
		return nil, fmt.Errorf("root_family: %w", err)
	}
	kinds, kindSet, err := readVocabulary(absolute, routing.IdentitySources["policy_kind"])
	if err != nil {
		return nil, fmt.Errorf("policy_kind: %w", err)
	}
	return &Domain{
		families: families,
		kinds:    kinds,
		family:   familySet,
		kind:     kindSet,
	}, nil
}

// RootFamilies returns a copy in contract order; no Family names are coded here.
func (d *Domain) RootFamilies() []string {
	if d == nil {
		return nil
	}
	return append([]string(nil), d.families...)
}

// PolicyKinds returns a copy of the declared policy-class vocabulary.
func (d *Domain) PolicyKinds() []string {
	if d == nil {
		return nil
	}
	return append([]string(nil), d.kinds...)
}

// HasRootFamily answers membership only, not whether a route or policy exists.
func (d *Domain) HasRootFamily(value string) bool {
	if d == nil {
		return false
	}
	_, found := d.family[value]
	return found
}

// HasPolicyKind answers membership only, not whether a policy is authorized.
func (d *Domain) HasPolicyKind(value string) bool {
	if d == nil {
		return false
	}
	_, found := d.kind[value]
	return found
}

func readVocabulary(root, reference string) ([]string, map[string]struct{}, error) {
	path, err := confinedFile(root, reference)
	if err != nil {
		return nil, nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("identity source unavailable: %w", err)
	}
	var contract struct {
		ID   string   `json:"$id"`
		Type string   `json:"type"`
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		return nil, nil, fmt.Errorf("identity source malformed: %w", err)
	}
	if contract.ID == "" || contract.Type != "string" || len(contract.Enum) == 0 {
		return nil, nil, errors.New("identity vocabulary must have $id, string type, and nonempty enum")
	}
	seen := make(map[string]struct{}, len(contract.Enum))
	for _, value := range contract.Enum {
		if value == "" || strings.TrimSpace(value) != value {
			return nil, nil, errors.New("empty or noncanonical identity token")
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, nil, fmt.Errorf("duplicate identity token: %s", value)
		}
		seen[value] = struct{}{}
	}
	return append([]string(nil), contract.Enum...), seen, nil
}

func confinedFile(root, reference string) (string, error) {
	if reference == "" || strings.Contains(reference, "\") ||
		strings.HasPrefix(reference, "/") {
		return "", errors.New("missing or invalid declared relative source")
	}
	parts := strings.Split(reference, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("unsafe identity source path")
		}
	}
	path := filepath.Join(root, filepath.FromSlash(reference))
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("repository root unresolved: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("declared identity source not found: %w", err)
	}
	relative, err := filepath.Rel(realRoot, realPath)
	if err != nil || relative == ".." || filepath.IsAbs(relative) ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("identity source escapes repository")
	}
	return realPath, nil
}
