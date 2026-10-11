use serde_json::Value;
use sha2::{Digest, Sha256};
use std::fs;
use std::path::{Component, Path};

pub fn observation(input: &Value, effect: &Value) -> Result<String, String> {
    let root = fs::canonicalize(
        input["root"]
            .as_str()
            .ok_or("explicit effect source root required")?,
    )
    .map_err(|e| e.to_string())?;
    let reference = &effect["source_ref"];
    let relative = Path::new(
        reference["path"]
            .as_str()
            .ok_or("registered effect source required")?,
    );
    if relative.is_absolute()
        || relative
            .components()
            .any(|c| !matches!(c, Component::Normal(_)))
    {
        return Err("invalid effect source reference".into());
    }
    let mut path = root.clone();
    for c in relative.components() {
        path.push(c);
        if fs::symlink_metadata(&path)
            .map_err(|e| e.to_string())?
            .file_type()
            .is_symlink()
        {
            return Err("unregistered effect source alias".into());
        }
    }
    let path = fs::canonicalize(path).map_err(|e| e.to_string())?;
    if !path.starts_with(root) {
        return Err("effect source escaped selected owner root".into());
    }
    let bytes = fs::read(path).map_err(|e| e.to_string())?;
    if bytes.len() > 2_097_152 {
        return Err("effect source exceeds byte budget".into());
    }
    if reference["sha256"] != format!("{:x}", Sha256::digest(&bytes)) {
        return Err("effect source digest stale".into());
    }
    let source: Value = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    if source["id"] != reference["id"]
        || source["version"] != reference["version"]
        || source["attempt_id"] != effect["attempt_id"]
        || source["observation"] != effect["observation"]
    {
        return Err("effect provenance/correlation mismatch".into());
    }
    let observation = source["observation"]
        .as_str()
        .ok_or("typed effect observation required")?;
    if !matches!(
        observation,
        "OBSERVED_EFFECT" | "VERIFIED_NO_EFFECT" | "UNKNOWN" | "UNVERIFIED"
    ) {
        return Err("unsupported effect taxonomy".into());
    }
    Ok(observation.to_owned())
}
