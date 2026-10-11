use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::BTreeSet;
use std::fs;
use std::path::{Component, Path};

fn text<'a>(v: &'a Value, key: &str) -> Result<&'a str, String> {
    v[key]
        .as_str()
        .filter(|s| !s.is_empty())
        .ok_or_else(|| format!("missing handoff field: {key}"))
}
fn read(root: &Path, reference: &str) -> Result<(Value, String), String> {
    let path = Path::new(reference);
    if path.is_absolute()
        || reference.contains('\\')
        || path
            .components()
            .any(|c| !matches!(c, Component::Normal(_)))
    {
        return Err("invalid handoff source reference".into());
    }
    let mut p = root.to_owned();
    for c in path.components() {
        p.push(c);
        if fs::symlink_metadata(&p)
            .map_err(|e| e.to_string())?
            .file_type()
            .is_symlink()
        {
            return Err("unregistered handoff source alias".into());
        }
    }
    let p = fs::canonicalize(p).map_err(|e| e.to_string())?;
    if !p.starts_with(root) {
        return Err("handoff source escaped selected owner root".into());
    }
    let bytes = fs::read(p).map_err(|e| e.to_string())?;
    if bytes.len() > 2_097_152 {
        return Err("handoff source byte budget exceeded".into());
    }
    Ok((
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?,
        format!("{:x}", Sha256::digest(&bytes)),
    ))
}
pub fn verify(input: &Value, contract: &Value) -> Result<Value, String> {
    let root = fs::canonicalize(text(input, "root")?).map_err(|e| e.to_string())?;
    let (mapping, revision) = read(&root, text(input, "mapping_ref")?)?;
    if input["expected_mapping_sha256"] != revision
        || mapping["schema_version"] != "ptsip-engine-preserved-api-mapping/v1"
        || mapping["public_api_cutover"] != false
        || mapping["legacy_retirement"] != false
    {
        return Err("preserved public API boundary or current mapping revision invalid".into());
    }
    let catalog = read(&root, text(&mapping, "source_catalog_ref")?)?.0;
    if catalog["contract_class"] != "VPMS_PRODUCT_CONTRACT" || catalog["capability"] != "ACTIVE" {
        return Err("current source contract catalog unavailable".into());
    }
    let active = catalog["contracts"]
        .as_object()
        .ok_or("source contract registrations required")?;
    let mut covered = BTreeSet::new();
    let mut relation_count = 0;
    let mut cards = BTreeSet::new();
    let operations: BTreeSet<_> = contract["operations"]
        .as_array()
        .ok_or("target capability operations required")?
        .iter()
        .map(|o| o["operation_id"].as_str().unwrap_or("").to_owned())
        .collect();
    for relation in mapping["relations"]
        .as_array()
        .ok_or("explicit versioned relations required")?
    {
        if relation["relation_kind"] != "VALIDATION_COVERAGE_ONLY"
            || relation["public_api_equivalence"] != false
        {
            return Err("generic validation cannot assert public API equivalence".into());
        }
        let sources = relation["source_refs"]
            .as_array()
            .ok_or("relation source identities required")?;
        let targets = relation["target_operations"]
            .as_array()
            .ok_or("relation target operations required")?;
        if sources.is_empty() || targets.is_empty() {
            return Err("empty relation boundary".into());
        }
        let cardinality = match (sources.len(), targets.len()) {
            (1, 1) => "ONE_TO_ONE",
            (1, _) => "ONE_TO_MANY",
            (_, 1) => "MANY_TO_ONE",
            _ => {
                return Err("many-to-many relation needs separately declared decomposition".into())
            }
        };
        if relation["cardinality"] != cardinality {
            return Err("declared relation cardinality mismatch".into());
        }
        cards.insert(cardinality);
        let mut relation_ids = BTreeSet::new();
        for reference in sources {
            let id = text(reference, "id")?;
            if !relation_ids.insert(id.to_owned()) {
                return Err("duplicate source relation identity".into());
            }
            let registration = active
                .get(id)
                .ok_or("unregistered current public source identity")?;
            if registration["status"] != "ACTIVE" {
                return Err("source contract is not currently active".into());
            }
            let path = text(reference, "path")?;
            let expected = format!("src/vpms/contracts/{}", text(registration, "path")?);
            if path != expected {
                return Err("source path does not match its current registration".into());
            }
            let (source, hash) = read(&root, path)?;
            if reference["sha256"] != hash
                || source["id"] != reference["id"]
                || source["status"] != "ACTIVE"
                || source["revision"].to_string() != text(reference, "version")?
            {
                return Err("stale source contract version/digest".into());
            }
            let classified = relation["source_responsibility_classification"][id]
                .as_object()
                .ok_or("complete source responsibility classification required")?;
            let owned = source["owns"]
                .as_array()
                .ok_or("current source responsibility list required")?;
            if classified.len() != owned.len() {
                return Err("unmapped source responsibility".into());
            }
            for responsibility in owned {
                let scope = classified
                    .get(
                        responsibility
                            .as_str()
                            .ok_or("typed responsibility required")?,
                    )
                    .and_then(Value::as_str);
                if !matches!(
                    scope,
                    Some("VALIDATION_ONLY_PARTIAL" | "PRESERVED_LEGACY_RUNTIME")
                ) {
                    return Err("unverified or implicit responsibility reassignment".into());
                }
            }
            covered.insert(id.to_owned());
        }
        for operation in targets {
            if !operations.contains(operation.as_str().unwrap_or("")) {
                return Err("unknown target capability operation".into());
            }
        }
        if relation["target_contract_id"] != contract["contract_id"]
            || relation["target_contract_version"] != contract["version"]
        {
            return Err("stale target Validation contract".into());
        }
        relation_count += 1;
    }
    if covered.len() != active.len() {
        return Err("current source API coverage incomplete".into());
    }
    if read(&root, text(input, "mapping_ref")?)?.1 != revision {
        return Err("mapping changed during verification".into());
    }
    Ok(
        json!({"mapping_revision":revision,"verified_relation_count":relation_count,"covered_source_ids":covered,"cardinalities":cards,"public_api_equivalence":false,"cutover_authorized":false,"legacy_retired":false,"ptsip_conformance_verdict":"NOT_ASSERTED"}),
    )
}

pub fn verify_context(input: &Value) -> Result<Value, String> {
    let root = fs::canonicalize(text(input, "root")?).map_err(|e| e.to_string())?;
    let (plan, revision) = read(&root, text(input, "mapping_ref")?)?;
    if input["expected_mapping_sha256"] != revision
        || plan["schema_version"] != "ptsip-context-projection-handoff/v1"
        || plan["source_semantics_owner"] != "CONTEXT"
        || plan["output_transformation_owner"] != "PROJECTION"
        || plan["legacy_contract_retired"] != false
    {
        return Err("Context/Projection boundary or mapping revision invalid".into());
    }
    if !plan["consumer_cutovers"]
        .as_array()
        .ok_or("explicit consumer cutover list required")?
        .is_empty()
    {
        return Err("consumer cutover is outside the preserved-API implementation scope".into());
    }
    let mut refs = Vec::new();
    for (field, owner) in [
        ("new_context_contract_ref", "CONTEXT"),
        ("new_projection_contract_ref", "PROJECTION"),
    ] {
        let path = text(&plan, field)?;
        let (contract, hash) = read(&root, path)?;
        if contract["semantic_owner"] != owner
            || contract["execution_permission_from_declaration"] != false
        {
            return Err("source or output semantic owner mismatch".into());
        }
        refs.push(json!({"id":contract["contract_id"],"version":contract["version"],"path":path,"sha256":hash}));
    }
    if read(&root, text(input, "mapping_ref")?)?.1 != revision {
        return Err("Context/Projection plan changed during verification".into());
    }
    Ok(
        json!({"mapping_revision":revision,"source_semantics_owner":"CONTEXT","output_transformation_owner":"PROJECTION","verified_contract_refs":refs,"cutover_authorized":false,"legacy_retired":false,"public_api_changed":false}),
    )
}
