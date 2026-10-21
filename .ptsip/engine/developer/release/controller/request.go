package releasecontroller

import (
	"fmt"
	"strings"
)

// ReleaseRequest resolves only the eight registered release types. Credential
// availability cannot alter the requested component inventory.
func ReleaseRequest(root, kind, engines, optional string) (Selection, error) {
	targets := ""
	switch kind {
	case "PTSIP_PACKAGE":
		targets = "package"
	case "TOOL_ONLY":
		targets = "tool"
	case "PROFILES_ONLY":
		targets = "profiles"
	case "TOOL_PROFILES":
		targets = "tool,profiles"
	case "ENGINE_ONLY", "MULTIPLE_ENGINES", "TOOL_ENGINES", "ENGINES_PROFILES":
		if strings.TrimSpace(engines) == "" {
			return Selection{}, fmt.Errorf("selected release type requires explicit registered engine names")
		}
		ids := []string{}
		if kind == "TOOL_ENGINES" {
			ids = append(ids, "tool")
		}
		if kind == "ENGINES_PROFILES" {
			ids = append(ids, "profiles")
		}
		for _, id := range strings.Split(engines, ",") {
			ids = append(ids, "engine:"+strings.TrimPrefix(strings.TrimSpace(id), "engine:"))
		}
		targets = strings.Join(ids, ",")
	default:
		return Selection{}, fmt.Errorf("unregistered release type: %s", kind)
	}
	if (kind == "PTSIP_PACKAGE" || kind == "TOOL_ONLY" || kind == "PROFILES_ONLY" || kind == "TOOL_PROFILES") && strings.TrimSpace(engines) != "" {
		return Selection{}, fmt.Errorf("this release type does not accept component exclusions or an engine subset")
	}
	s, err := Select(root, targets, optional)
	if err != nil {
		return s, err
	}
	if s.Type != kind {
		return Selection{}, fmt.Errorf("requested release type %s differs from resolved component type %s", kind, s.Type)
	}
	return s, nil
}

// CompletePackage checks exact identities, not a fixed engine count. The same
// registered inventory is used for full release and full installation.
func CompletePackage(s Snapshot, registered map[string]string) error {
	if err := ValidateSnapshot(s); err != nil {
		return err
	}
	expected := map[string]string{"tool": "tool", "profiles": "profiles"}
	for id := range registered {
		if id == "tool" || id == "profiles" {
			return fmt.Errorf("registered engine collides with Package identity")
		}
		expected[id] = "engine"
	}
	if len(s.Components) != len(expected) {
		return fmt.Errorf("Package must contain Tool, Profiles and every registered engine without omissions")
	}
	for _, c := range s.Components {
		if expected[c.ID] != c.Kind {
			return fmt.Errorf("Package has an unregistered or misclassified component: %s", c.ID)
		}
		delete(expected, c.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("Package registered component inventory incomplete")
	}
	return nil
}
