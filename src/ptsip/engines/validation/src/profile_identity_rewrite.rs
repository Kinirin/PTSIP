use crate::{profile, profile_path, profile_yaml, schema, source};
use serde_json::{json, Value};
use std::fs::File;
use std::io::Read;
use std::path::PathBuf;

pub const ALGORITHM: &str = "VERIFY_IDENTITY_ONLY_PROFILE_REWRITE_AGAINST_REGISTERED_TARGET_SCHEMA";
const RESOURCE_LIMIT: u64 = 2_097_152;

fn read(root: &str, path: &str) -> Result<Vec<u8>, String> {
    let location = profile_path::resolve(&json!({"root": root,"path": path}))?;
    if location["normalized_path"] != path {
        return Err("Profile resource path must already be normalized".into());
    }
    let path = PathBuf::from(source::text(&location, "path_on_disk")?);
    let file = File::open(path).map_err(|error| error.to_string())?;
    let mut bytes = Vec::new();
    file.take(RESOURCE_LIMIT + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 > RESOURCE_LIMIT {
        return Err("Profile resource exceeds the native read limit".into());
    }
    Ok(bytes)
}

fn profile_payload(bytes: &[u8]) -> Result<Value, String> {
    let bytes = bytes.strip_prefix(b"\xef\xbb\xbf").unwrap_or(bytes);
    profile_yaml::parse_yaml(bytes)
}

fn target_schema(package: &str, target: &str) -> Result<(Value, String, Vec<u8>), String> {
    let config = read(package, "schemas/config.json")?;
    let data = source::parse(&config)?;
    let mut matches = data["storage"]["profile"]
        .as_object()
        .ok_or("registered Profile schema catalog required")?
        .iter()
        .filter(|(_, entry)| entry["version"] == target);
    let (selector, registration) = matches
        .next()
        .ok_or("unregistered target Profile contract")?;
    if matches.next().is_some() {
        return Err("ambiguous target Profile schema registration".into());
    }
    let path = source::text(registration, "path")?;
    let expected_path = if selector == "ptsip.project-profile" {
        "storage/profile/project-profile.schema.json".to_owned()
    } else {
        format!("history/profile/project-profile-{target}.schema.json")
    };
    if path != expected_path {
        return Err("target does not resolve to a registered Profile schema path".into());
    }
    let relative = format!("schemas/{path}");
    let bytes = read(package, &relative)?;
    let canonical = std::str::from_utf8(&bytes)
        .map_err(|error| error.to_string())?
        .replace("\r\n", "\n");
    if registration["schema_sha256"] != source::hash(canonical.as_bytes()) {
        return Err("target Profile schema SHA-256 mismatch".into());
    }
    let schema = source::parse(&bytes)?;
    if schema["$id"] != registration["$id"]
        || !schema["$id"].is_string()
        || schema.pointer("/properties/ptsip/properties/version/const") != Some(&json!(target))
    {
        return Err("target Profile schema identity mismatch".into());
    }
    if read(package, "schemas/config.json")? != config {
        return Err("Profile catalog changed during target schema resolution".into());
    }
    Ok((schema, relative, bytes))
}

fn check_change(before: &[u8], after: &[u8], plan: &Value) -> Result<Value, String> {
    let from = source::text(plan, "source_declared_version")?;
    let target = source::text(plan, "target_contract")?;
    let revision = source::text(plan, "specification_revision")?;
    let family = source::text(plan, "specification_family")?;
    let transition = profile::transition(&json!({
        "source_contract_version":from,"target_contract_version":target,
        "source_revision":revision,"declared_transition":"IDENTITY_ONLY"
    }))?;
    if transition["allowed"] != true || from != family {
        return Err("no exact registered identity-only transition matches the plan".into());
    }
    let mut expected = profile_payload(before)?;
    let after = profile_payload(after)?;
    let ptsip = expected["ptsip"]
        .as_object_mut()
        .ok_or("source Profile is missing its ptsip mapping")?;
    if ptsip.get("version").and_then(Value::as_str) != Some(from) {
        return Err("source Profile version differs from the rewrite plan".into());
    }
    let spec = ptsip
        .get_mut("specification")
        .and_then(Value::as_object_mut)
        .ok_or("source Profile is missing its specification mapping")?;
    if spec.get("revision").and_then(Value::as_str) != Some(revision) {
        return Err("source specification revision differs from the rewrite plan".into());
    }
    if !transition["legacy_target_filename"].is_null() {
        if spec.contains_key("family") {
            return Err("historical source already declares the specification family".into());
        }
        spec.insert("family".into(), json!(family));
    }
    ptsip.insert("version".into(), json!(target));
    if expected != after {
        return Err(
            "rewrite changed Profile fields beyond the registered identity transition".into(),
        );
    }
    Ok(after)
}

pub fn verify(input: &Value) -> Result<Value, String> {
    let plan = &input["plan"];
    let repository = source::text(input, "root")?;
    let package = source::text(input, "package_root")?;
    let path = source::text(plan, "source_path")?;
    let before = source::text(input, "before_content")?.as_bytes();
    let observed = read(repository, path)?;
    let proposed = input["phase"] == "PROPOSED";
    let after = if proposed {
        source::text(input, "after_content")?.as_bytes()
    } else {
        observed.as_slice()
    };
    let after_sha256 = source::hash(after);
    let observed_sha256 = source::hash(&observed);
    let errors = (|| {
        if plan["source_sha256"] != source::hash(before)
            || (proposed && observed != before)
            || input["after_sha256"] != after_sha256
        {
            return Err("Profile rewrite byte evidence differs from the planned sources".into());
        }
        let payload = check_change(before, after, plan)?;
        let (schema, schema_path, schema_bytes) =
            target_schema(package, source::text(plan, "target_contract")?)?;
        let (mut diagnostics, _) = schema::validate_with_resolution(&schema, &payload, None, None)?;
        if payload["ptsip"]["profile_role"] == "DISTRIBUTED_EXAMPLE" {
            diagnostics.push("Project Profile rewrite requires a project-owned Profile".into());
        }
        if read(repository, path)? != observed || read(package, &schema_path)? != schema_bytes {
            return Err("Profile or target schema changed during rewrite verification".into());
        }
        Ok(diagnostics)
    })()
    .unwrap_or_else(|error| vec![error]);
    Ok(
        json!({"valid":errors.is_empty(),"errors":errors,"warnings":[],"observed_sha256":observed_sha256}),
    )
}
