package lifecycle

import "sort"

// CheckMigrationSourceCatalog compares audit-only catalog membership with the
// admitted migration graph. It never reads archived bodies or grants authority
// to a source identity, even when its preserved lifecycle status is ACTIVE.
func CheckMigrationSourceCatalog(index, graph Object) error {
	sources := map[string]Object{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		if entry["authority_role"] != "MIGRATION_SOURCE" {
			continue
		}
		id := Text(entry["id"])
		if _, found := sources[id]; found {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", id+": duplicate audit-only catalog identity")
		}
		sources[id] = entry
	}
	if graph == nil {
		if len(sources) > 0 || Text(index["migration_registry_ref"]) != "" {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", "admitted migration registry is required")
		}
		return nil
	}
	if graph["policy_class"] != DeveloperClass {
		return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", "migration registry class mismatch")
	}
	seen := map[string]bool{}
	for _, raw := range List(graph["sources"]) {
		source := Map(raw)
		id := Text(source["source_policy_id"])
		if seen[id] {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", id+": duplicate migration registry identity")
		}
		seen[id] = true
		entry, found := sources[id]
		if !found {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", id+": migration registry source is missing from the audit-only catalog")
		}
		header := Map(source["header"])
		identity := Map(header["policy"])
		if identity["id"] != id || identity["status"] != source["source_status"] || header["policy_class"] != DeveloperClass {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", id+": inconsistent migration source header")
		}
		if entry["policy_class"] != header["policy_class"] || entry["status"] != source["source_status"] || entry["path"] != "developer/policy/"+Text(source["archive_path"]) {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", id+": catalog class/status/path differs from migration evidence")
		}
		delete(sources, id)
	}
	if len(sources) > 0 {
		ids := []string{}
		for id := range sources {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", ids[0]+": audit-only catalog identity is not registered in migration evidence")
	}
	return nil
}

func checkRegisteredMigrationSources(r Repository, index Object) error {
	var graph Object
	if reference := Text(index["migration_registry_ref"]); reference != "" {
		var err error
		graph, err = r.Read("developer/policy/" + reference)
		if err != nil {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", err.Error())
		}
		if err := r.Validate("developer/policy/schemas/root-family-migration.schema.json", graph); err != nil {
			return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", err.Error())
		}
	}
	return CheckMigrationSourceCatalog(index, graph)
}
