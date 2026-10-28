// Package layout checks developer module routing without loading any policy
// authority, legacy implementation, or temporary work-record file.
package layout

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

const directory = ".ptsip/engine/developer/"

type boundary struct {
	Lifecycle   string `json:"lifecycle"`
	Replacement bool   `json:"replacement_required"`
	Authority   string `json:"policy_authority"`
	Execution   *bool  `json:"execution_authorized"`
	Publication *bool  `json:"publication_authorized"`
}

func (b boundary) check() error {
	if b.Lifecycle != "TEMPORARY_REPLACEABLE" || !b.Replacement || b.Authority != "NONE" ||
		b.Execution == nil || *b.Execution || b.Publication == nil || *b.Publication {
		return fmt.Errorf("TEMPORARY_VALIDATOR_BOUNDARY_INVALID")
	}
	return nil
}

func read(root, relative string, target any) error {
	file, err := authoring.LocalFile(root, directory+relative)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("INVALID_VALIDATOR_INDEX: %s: %w", relative, err)
	}
	return nil
}

func Run(root string) (any, int) {
	report := map[string]any{"status": "FAIL", "scope": "DEVELOPER_MODULE_LAYOUT", "policy_authority": "NONE", "execution_authorized": false, "publication_authorized": false}
	fail := func(err error) (any, int) { report["problem"] = err.Error(); return report, 2 }
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fail(err)
	}
	var index struct {
		boundary
		Schema  string `json:"schema_version"`
		Domains map[string]struct {
			Index string `json:"index"`
		} `json:"domains"`
		Validators map[string]string `json:"validators"`
	}
	if err := read(absolute, "index.json", &index); err != nil {
		return fail(err)
	}
	if err := index.check(); err != nil {
		return fail(err)
	}
	if index.Schema != "ptsip-developer-validator-index/v1" || len(index.Domains) != 3 {
		return fail(fmt.Errorf("VALIDATOR_ROOT_INDEX_INVALID"))
	}
	domains := []string{"Distribution", "policy", "Responsibility-Authority"}
	knownIndexes := map[string]bool{"index.json": true}
	counts := map[string]int{}
	for _, name := range domains {
		ref := name + "/index.json"
		if index.Domains[name].Index != ref {
			return fail(fmt.Errorf("DOMAIN_INDEX_MISMATCH: %s", name))
		}
		var domain struct {
			boundary
			Schema     string `json:"schema_version"`
			Domain     string `json:"domain"`
			Components []struct {
				ID             string `json:"id"`
				Root           string `json:"root"`
				State          string `json:"state"`
				Implementation string `json:"implementation"`
				Index          string `json:"index"`
			} `json:"components"`
		}
		if err := read(absolute, ref, &domain); err != nil {
			return fail(err)
		}
		if err := domain.check(); err != nil {
			return fail(err)
		}
		if domain.Schema != "ptsip-developer-validator-domain/v1" || domain.Domain != name || len(domain.Components) == 0 {
			return fail(fmt.Errorf("DOMAIN_CONTRACT_INVALID: %s", name))
		}
		seen := map[string]bool{}
		for _, component := range domain.Components {
			if component.ID == "" || seen[component.ID] || component.Root == "" {
				return fail(fmt.Errorf("COMPONENT_ID_OR_ROOT_INVALID: %s", name))
			}
			seen[component.ID] = true
			if !map[string]bool{"IMPLEMENTED": true, "STRUCTURE_ONLY": true, "COMPATIBILITY_BLOCKED": true, "PLANNED": true, "INTEGRATION_SUITE": true}[component.State] {
				return fail(fmt.Errorf("UNKNOWN_IMPLEMENTATION_STATE: %s", component.ID))
			}
			counts[component.State]++
			if component.Implementation != "" {
				if !strings.HasPrefix(component.Implementation, component.Root+"/") {
					return fail(fmt.Errorf("IMPLEMENTATION_OUTSIDE_COMPONENT: %s", component.ID))
				}
				if _, err := authoring.LocalFile(absolute, directory+name+"/"+component.Implementation); err != nil {
					return fail(err)
				}
			} else if component.State != "PLANNED" && component.State != "INTEGRATION_SUITE" {
				return fail(fmt.Errorf("IMPLEMENTATION_REQUIRED: %s", component.ID))
			}
			if component.Index != "" {
				if component.Index != component.Root+"/index.json" {
					return fail(fmt.Errorf("COMPONENT_INDEX_MISMATCH: %s", component.ID))
				}
				var child struct {
					boundary
					State string `json:"state"`
				}
				if err := read(absolute, name+"/"+component.Index, &child); err != nil {
					return fail(err)
				}
				if err := child.check(); err != nil {
					return fail(err)
				}
				if child.State != component.State {
					return fail(fmt.Errorf("IMPLEMENTATION_STATE_MISMATCH: %s", component.ID))
				}
			}
		}
		knownIndexes[ref] = true
	}
	names := make([]string, 0, len(index.Validators))
	for name, ref := range index.Validators {
		if name == "" || !knownIndexes[ref] {
			return fail(fmt.Errorf("UNREGISTERED_VALIDATOR_DOMAIN: %s", name))
		}
		names = append(names, name)
	}
	sort.Strings(names)
	report["status"], report["domains"], report["validators"], report["implementation_states"] = "VALID", domains, names, counts
	report["validation_claim"] = "LAYOUT_ONLY"
	return report, 0
}
