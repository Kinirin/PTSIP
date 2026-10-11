use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::{Component, Path, PathBuf};

fn text<'a>(v: &'a Value, k: &str) -> Result<&'a str, String> {
    v[k].as_str()
        .filter(|s| !s.is_empty())
        .ok_or_else(|| format!("missing provider field: {k}"))
}
fn read(root: &Path, reference: &str) -> Result<(Value, String), String> {
    let relative = Path::new(reference);
    if relative.is_absolute()
        || reference.contains('\\')
        || relative
            .components()
            .any(|c| !matches!(c, Component::Normal(_)))
    {
        return Err("provider source path must be registered relative to its owner root".into());
    }
    let mut path = PathBuf::from(root);
    for c in relative.components() {
        path.push(c);
        if fs::symlink_metadata(&path)
            .map_err(|e| e.to_string())?
            .file_type()
            .is_symlink()
        {
            return Err("unregistered provider source alias".into());
        }
    }
    let resolved = fs::canonicalize(path).map_err(|e| e.to_string())?;
    if !resolved.starts_with(root) {
        return Err("provider source escapes registered root".into());
    }
    let bytes = fs::read(resolved).map_err(|e| e.to_string())?;
    if bytes.len() > 2_097_152 {
        return Err("provider source exceeds byte budget".into());
    }
    let digest = format!("{:x}", Sha256::digest(&bytes));
    Ok((
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?,
        digest,
    ))
}
fn bound(root: &Path, r: &Value) -> Result<Value, String> {
    let (value, hash) = read(root, text(r, "path")?)?;
    if r["sha256"] != hash || value["id"] != r["id"] || value["version"] != r["version"] {
        return Err("provider source identity/version/digest mismatch".into());
    }
    Ok(value)
}
pub fn mediate(input: &Value, manifest: &Value, contract: &Value) -> Result<Value, String> {
    let root = fs::canonicalize(text(input, "root")?).map_err(|e| e.to_string())?;
    let registry_path = text(input, "provider_registry_ref")?;
    let (registry, hash) = read(&root, registry_path)?;
    if input["expected_registry_sha256"] != hash
        || registry["schema_version"] != "ptsip-storage-provider-registry/v1"
        || registry["engine_id"] != manifest["engine_id"]
    {
        return Err("current Storage owner registry proof unavailable".into());
    }
    if input["capability_id"] != contract["contract_id"]
        || input["contract_version"] != contract["version"]
    {
        return Err("current Storage capability identity mismatch".into());
    }
    let candidates: Vec<_> = registry["providers"]
        .as_array()
        .ok_or("registered provider list required")?
        .iter()
        .filter(|p| {
            p["operation_id"] == input["operation_id"]
                && p["storage_capability_id"] == input["capability_id"]
                && p["storage_contract_version"] == input["contract_version"]
        })
        .collect();
    if candidates.len() != 1 {
        return Err("provider operation registration missing or ambiguous".into());
    }
    let p = candidates[0];
    let provider = bound(&root, &p["source_ref"])?;
    let evidence = bound(&root, &p["compatibility_ref"])?;
    if provider["provider_id"] != p["provider_id"]
        || provider["provider_owner"] != p["provider_owner"]
        || provider["status"] != "ACTIVE"
    {
        return Err("provider identity or owner is not current".into());
    }
    let operations = provider["operations"]
        .as_array()
        .ok_or("provider operation declarations required")?;
    if operations
        .iter()
        .filter(|o| o["operation_id"] == p["provider_operation_id"])
        .count()
        != 1
    {
        return Err("provider operation declaration missing or ambiguous".into());
    }
    if evidence["provider_id"] != p["provider_id"]
        || evidence["storage_engine_id"] != manifest["engine_id"]
        || evidence["provider_operation_id"] != p["provider_operation_id"]
        || evidence["storage_operation_id"] != input["operation_id"]
        || evidence["outcome"] != "PASS"
    {
        return Err("applicable provider compatibility proof unavailable".into());
    }
    let refs = evidence["source_refs"]
        .as_array()
        .ok_or("current provider/consumer evidence references required")?;
    if refs.len() != 2 {
        return Err("both current provider and Storage contract references required".into());
    }
    let provider_ref = refs
        .iter()
        .find(|r| r["id"] == provider["id"])
        .ok_or("provider evidence link missing")?;
    if provider_ref != &p["source_ref"] {
        return Err("provider evidence is not bound to the selected registered source".into());
    }
    bound(&root, provider_ref)?;
    let consumer_ref = refs
        .iter()
        .find(|r| r["id"] == contract["contract_id"])
        .ok_or("Storage evidence link missing")?;
    let (consumer, consumer_hash) = read(&root, text(consumer_ref, "path")?)?;
    if consumer != *contract
        || consumer_ref["sha256"] != consumer_hash
        || consumer_ref["version"] != contract["version"]
    {
        return Err("stale Storage consumer contract evidence".into());
    }
    // Confirm registry and evidence have not changed while selecting a plan.
    if read(&root, registry_path)?.1 != hash
        || bound(&root, &p["source_ref"])? != provider
        || bound(&root, &p["compatibility_ref"])? != evidence
        || read(&root, text(consumer_ref, "path")?)?.1 != consumer_hash
    {
        return Err("provider sources changed during mediation".into());
    }
    Ok(
        json!({"provider_id":p["provider_id"],"provider_owner":p["provider_owner"],"provider_operation_id":p["provider_operation_id"],"registry_revision":hash,"execution_authorized":false,"provider_invoked":false}),
    )
}
