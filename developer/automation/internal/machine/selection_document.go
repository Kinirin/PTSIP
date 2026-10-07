package machine

import (
	"reflect"
	"sort"
)

func (r *Repository) ValidateSelectionDocument(kind string, payload Object) (Object, error) {
	file := map[string]string{"request": "selection-request", "result": "selection-result", "rule": "selection-rule"}[kind]
	if file == "" {
		return nil, Fail("UNKNOWN_SELECTION_DOCUMENT", kind)
	}
	identity, err := r.Read("src/vpms/contracts/identity-registry.json")
	if err != nil {
		return nil, err
	}
	resources := []string{"src/vpms/contracts/identity-registry.json"}
	for _, path := range Strings(identity["schema_resources"]) {
		resources = append(resources, "src/vpms/contracts/"+path)
	}
	if err := r.ValidateResourceSet("src/vpms/contracts/schemas/"+file+".schema.json", payload, resources); err != nil {
		return nil, err
	}
	if kind == "result" {
		catalog, err := r.Read("src/vpms/contracts/index.json")
		if err != nil {
			return nil, err
		}
		selection, err := r.ProductContract(Text(Map(catalog["entrypoints"])["selection"]), false)
		if err != nil {
			return nil, err
		}
		semantics := Map(selection["semantics"])
		if semantics["ordering"] != "CASE_ID_ASCENDING" || !reflect.DeepEqual(Strings(semantics["diagnostic_order"]), []string{"location", "code", "reference"}) {
			return nil, Fail("UNSUPPORTED_SELECTION_ORDER", "selection contract ordering is not registered")
		}
		cases := Strings(payload["case_ids"])
		ordered := append([]string{}, cases...)
		sort.Strings(ordered)
		if !reflect.DeepEqual(cases, ordered) {
			return nil, Fail("NONDETERMINISTIC_SELECTION_ORDER", "case IDs must be ascending")
		}
		previous := ""
		for _, raw := range List(payload["diagnostics"]) {
			item := Map(raw)
			key := Text(item["location"]) + "\x00" + Text(item["code"]) + "\x00" + Text(item["reference"])
			if key < previous {
				return nil, Fail("NONDETERMINISTIC_DIAGNOSTIC_ORDER", "diagnostics must follow registered fields")
			}
			previous = key
		}
	}
	return Object{"status": "PASS", "kind": kind, "execution_performed": false}, nil
}
