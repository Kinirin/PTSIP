package lifecycle

import (
	"crypto/sha1"
	"fmt"
	"os"
	"sort"
)

const approvedMigrationBaseline = "developer/policy/registries/root-family-approved-baseline.json"
const approvedMigrationBaselineSchema = "developer/policy/schemas/root-family-approved-baseline.schema.json"

// CheckMigrationSourceCatalog validates a frozen, approved historical partition.
// A historical-only identity is not granted current catalog or execution authority.
func CheckMigrationSourceCatalog(index, graph, baseline Object) error {
	fail := func(s string) error { return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", s) }
	if index["migration_registry_ref"] != "registries/root-family-migration.json" {
		return fail("approved migration registry is not admitted by catalog")
	}
	if graph == nil || baseline == nil ||
		graph["policy_class"] != DeveloperClass ||
		Map(graph["approval"])["decision"] != "APPROVED" ||
		graph["base_head"] != baseline["approved_migration_base_head"] ||
		baseline["authority"] != "HISTORICAL_EVIDENCE_ONLY" ||
		baseline["migration_registry_ref"] != "developer/policy/registries/root-family-migration.json" ||
		baseline["source_classification"] != "APPROVED_PRE_MIGRATION_CATALOG_PATH_AND_FROZEN_MIGRATION_GRAPH" {
		return fail("historical migration evidence is not bound to approved baseline")
	}
	frozen := map[string]Object{}
	for _, raw := range List(baseline["source_records"]) {
		item := Map(raw)
		id := Text(item["id"])
		if id == "" || frozen[id] != nil {
			return fail(id + ": missing/duplicate approved baseline identity")
		}
		if item["role"] != "AUDIT_ONLY_CATALOG" && item["role"] != "HISTORICAL_ONLY" {
			return fail(id + ": unapproved historical classification")
		}
		frozen[id] = item
	}
	if len(frozen) == 0 { return fail("empty approved migration baseline") }
	actual := map[string]Object{}
	for _, raw := range List(graph["sources"]) {
		source := Map(raw)
		id := Text(source["source_policy_id"])
		if actual[id] != nil || frozen[id] == nil {
			return fail(id + ": unregistered/duplicate historical graph source")
		}
		actual[id] = source
	}
	registered := map[string]Object{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		if entry["authority_role"] != "MIGRATION_SOURCE" { continue }
		id := Text(entry["id"])
		if registered[id] != nil || frozen[id] == nil {
			return fail(id + ": unapproved or duplicate catalog source")
		}
		registered[id] = entry
	}
	ids := make([]string, 0, len(frozen))
	for id := range frozen { ids = append(ids, id) }
	sort.Strings(ids)
	for _, id := range ids {
		item, src := frozen[id], actual[id]
		if src == nil { return fail(id + ": approved historical evidence missing") }
		header := Map(src["header"])
		identity := Map(header["policy"])
		if src["source_status"] != item["source_status"] ||
			src["original_path"] != item["original_path"] ||
			src["archive_path"] != item["archive_path"] ||
			src["source_sha256"] != item["source_sha256"] ||
			src["record_sha256"] != item["record_sha256"] ||
			identity["id"] != id ||
			identity["status"] != item["source_status"] ||
			header["policy_class"] != DeveloperClass {
			return fail(id + ": immutable source metadata differs from approved snapshot")
		}
		entry := registered[id]
		if item["role"] == "HISTORICAL_ONLY" {
			if entry != nil { return fail(id + ": historical-only evidence granted catalog membership") }
			continue
		}
		if entry == nil {
			return fail(id + ": required audit-only catalog identity omitted")
		}
		if entry["policy_class"] != header["policy_class"] ||
			entry["status"] != item["source_status"] ||
			entry["path"] != "developer/policy/"+Text(item["archive_path"]) {
			return fail(id + ": audit-only catalog membership differs from frozen source")
		}
	}
	return nil
}

func checkRegisteredMigrationSources(r Repository, index Object) error {
	fail := func(s string) error { return policyFailure("MIGRATION_SOURCE_CATALOG_MISMATCH", s) }
	baseline, err := r.Read(approvedMigrationBaseline)
	if err != nil { return fail(err.Error()) }
	if err := r.Validate(approvedMigrationBaselineSchema, baseline); err != nil { return fail(err.Error()) }
	if baseline["schema_version"] != "ptsip-root-family-approved-migration-baseline/v1" ||
		baseline["policy_class"] != DeveloperClass {
		return fail("invalid approved historical classification contract")
	}
	path, err := r.Path(Text(baseline["migration_registry_ref"]))
	if err != nil { return fail(err.Error()) }
	blob, err := os.ReadFile(path)
	if err != nil { return fail(err.Error()) }
	digest := sha1.Sum(append([]byte(fmt.Sprintf("blob %d\x00", len(blob))), blob...))
	if fmt.Sprintf("%x", digest) != Text(baseline["approved_migration_graph_git_blob"]) {
		return fail("migration registry differs from approved immutable graph snapshot")
	}
	graph, err := r.Read(Text(baseline["migration_registry_ref"]))
	if err != nil { return fail(err.Error()) }
	if err := r.Validate("developer/policy/schemas/root-family-migration.schema.json", graph); err != nil { return fail(err.Error()) }
	return CheckMigrationSourceCatalog(index, graph, baseline)
}
