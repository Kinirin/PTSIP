//! Exact Profile resource selection. Validation and presentation have separate owners.
use ptsip_engine_discovery::source::{self, Root};
use serde_json::{json, Value};
use std::collections::BTreeSet;

const PROFILE_SUPPORT: &str = "engines/discovery/profile-support.json";

pub fn algorithm(operation: &str) -> Option<&'static str> {
    match operation {
        "discovery.profile.contract" => {
            Some("SELECT_EXACT_REGISTERED_PROFILE_CONTRACT_AND_VERIFY_RESOURCE_IDENTITY")
        }
        "discovery.profile.catalog" => {
            Some("RESOLVE_EXPLICIT_LOCAL_PROFILE_CATALOG_WITHOUT_WRITES")
        }
        _ => None,
    }
}

pub fn contract(input: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "root")?)?;
    let registry_path = input["registry_path"]
        .as_str()
        .unwrap_or("schemas/config.json");
    if registry_path != "schemas/config.json" {
        return Err("Profile contracts must use the registered central schema catalog".into());
    }
    let (registry, _) = root.read(registry_path)?;
    let profiles = registry["storage"]["profile"]
        .as_object()
        .ok_or("registered Profile catalog required")?;
    let current_selection = input.get("contract_version").is_none_or(Value::is_null);
    let version = match input.get("contract_version").filter(|v| !v.is_null()) {
        Some(value) => value.as_str().ok_or("contract_version must be a string")?,
        None => profiles["ptsip.project-profile"]["version"]
            .as_str()
            .ok_or("current Profile version required")?,
    };
    let current = profiles
        .get("ptsip.project-profile")
        .ok_or("current Profile registration required")?;
    let key = format!("ptsip.project-profile.{version}");
    let registration = if current["version"] == version {
        current
    } else {
        profiles
            .get(&key)
            .ok_or("unknown registered Profile version")?
    };
    if registration["version"] != version {
        return Err("Profile registration version mismatch".into());
    }
    let lifecycle = source::text(registration, "registered_lifecycle")?;
    if !["ACTIVE", "LEGACY", "DEPRECATED"].contains(&lifecycle) {
        return Err("unsupported registered_lifecycle".into());
    }
    let relative = format!("schemas/{}", source::text(registration, "path")?);
    let (schema, digest) = registered_schema(&mut root, registration, &relative)?;
    if schema["$id"] != registration["$id"] || !schema["$id"].is_string() {
        return Err("Profile schema resource identity mismatch".into());
    }
    if schema.pointer("/properties/ptsip/properties/version/const") != Some(&json!(version)) {
        return Err("Profile schema resource version mismatch".into());
    }
    if current_selection || current["version"] == version {
        if lifecycle != "ACTIVE" || schema.get("$ref").is_some() {
            return Err("current Profile must be an active direct official schema".into());
        }
    }
    if let Some(expected) = input.get("source_revision").filter(|v| !v.is_null()) {
        if expected != &digest {
            return Err("Profile source revision mismatch".into());
        }
    }
    let (support, _) = root.read(PROFILE_SUPPORT)?;
    let operations = support
        .get(version)
        .and_then(Value::as_array)
        .ok_or("Profile support operations missing for registered version")?;
    let mut unique = BTreeSet::new();
    for operation in operations {
        let operation = operation
            .as_str()
            .ok_or("Profile operation must be a string")?;
        if ![
            "IDENTIFY",
            "VALIDATE",
            "ANALYZE",
            "MIGRATE_SOURCE",
            "CREATE_TARGET",
        ]
        .contains(&operation)
            || !unique.insert(operation)
        {
            return Err("invalid or duplicate Profile support operation".into());
        }
    }
    if operations.is_empty() {
        return Err("Profile support operations empty".into());
    }
    let allowed = !(lifecycle == "LEGACY" && input["new_registration"] == true);
    let notifications = if lifecycle == "DEPRECATED" {
        json!([{"code":"UPGRADE_REQUIRED","message":"Support retirement is scheduled; transition to the active Profile contract is required.","mandatory":true}])
    } else {
        json!([])
    };
    root.finish()?;
    Ok(
        json!({"contract_id":schema["$id"],"version":version,"schema_resource":relative,
        "source_sha256":digest,"lifecycle":lifecycle,"operations":operations,"allowed":allowed,"notifications":notifications}),
    )
}

fn registered_schema(
    root: &mut Root,
    registration: &Value,
    reference: &str,
) -> Result<(Value, String), String> {
    let expected = source::text(registration, "schema_sha256")?;
    if expected.len() != 64
        || !expected
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
    {
        return Err("registered Profile schema SHA256 must be a lowercase digest".into());
    }
    let (schema, digest) = root.read(reference)?;
    let raw = root.bytes(reference, 2_097_152)?;
    if source::hash(&raw) != digest {
        return Err("Profile schema changed during integrity verification".into());
    }
    let canonical = std::str::from_utf8(&raw)
        .map_err(|e| e.to_string())?
        .replace("\r\n", "\n");
    if source::hash(canonical.as_bytes()) != expected {
        return Err("registered Profile schema SHA256 mismatch".into());
    }
    Ok((schema, digest))
}

pub(crate) struct Catalog {
    pub reference: String,
    pub digest: String,
    pub payload: Value,
}

pub(crate) fn load_catalog(root: &Root, explicit: Option<&str>) -> Result<Option<Catalog>, String> {
    let canonical_exists = root.exists(".ptsip/profiles/index.json")?;
    let legacy_exists = root.exists(".ptsip/profiles/index.yaml")?;
    if canonical_exists && legacy_exists {
        return Err("canonical and legacy Profile catalogs conflict".into());
    }
    let reference = explicit.unwrap_or(if canonical_exists || !legacy_exists {
        ".ptsip/profiles/index.json"
    } else {
        ".ptsip/profiles/index.yaml"
    });
    if ![".ptsip/profiles/index.json", ".ptsip/profiles/index.yaml"].contains(&reference) {
        return Err("unregistered Profile catalog reference".into());
    }
    // This probes every existing ancestor before treating a missing leaf as absent.
    if !root.exists(reference)? {
        return Ok(None);
    }
    let bytes = root.bytes(reference, 2_097_152)?;
    let catalog = if reference.ends_with(".json") {
        source::parse(&bytes)?
    } else {
        source::parse_yaml(&bytes)?
    };
    if catalog["schema_version"] != "ptsip-local-profile-catalog/v1" {
        return Err("unsupported Profile catalog schema version".into());
    }
    let mut ids = BTreeSet::new();
    let mut resources = BTreeSet::new();
    let entries = catalog["profiles"]
        .as_array()
        .filter(|a| !a.is_empty())
        .ok_or("Profile catalog entries required")?;
    for row in entries {
        let id = source::text(row, "id")?;
        let name = source::text(row, "resource")?;
        if name.contains(['/', '\\', ':'])
            || name == ".ptsip.yaml"
            || !name.ends_with(".ptsip.yaml")
        {
            return Err("Profile resource must be one relative *.ptsip.yaml filename".into());
        }
        if !ids.insert(id) || !resources.insert(name) {
            return Err("duplicate Profile ID or resource".into());
        }
    }
    if !ids.contains(source::text(&catalog, "default_profile")?) {
        return Err("default Profile ID is not registered".into());
    }
    Ok(Some(Catalog {
        reference: reference.into(),
        digest: source::hash(&bytes),
        payload: catalog,
    }))
}

pub(crate) fn selected_resource(
    catalog: &Catalog,
    requested: Option<&str>,
) -> Result<String, String> {
    let selected = requested.unwrap_or(source::text(&catalog.payload, "default_profile")?);
    let entries = catalog.payload["profiles"]
        .as_array()
        .ok_or("Profile catalog entries required")?;
    let row = entries
        .iter()
        .find(|row| row["id"] == selected)
        .ok_or("requested Profile ID is not registered")?;
    let resource = source::text(row, "resource")?;
    Ok(format!(".ptsip/profiles/{resource}"))
}

pub fn catalog(input: &Value) -> Result<Value, String> {
    let root = Root::new(source::text(input, "repository_root")?)?;
    let Some(catalog) = load_catalog(&root, input["catalog_ref"].as_str())? else {
        return Ok(
            json!({"selected_id":null,"resource_path":null,"default_new_profile_path":".ptsip/profiles/main.ptsip.yaml","catalog_valid":false,"catalog_sha256":null}),
        );
    };
    let selected = input["requested_profile_id"]
        .as_str()
        .unwrap_or(source::text(&catalog.payload, "default_profile")?);
    let relative = selected_resource(&catalog, Some(selected))?;
    root.bytes(&relative, 2_097_152)?;
    if source::hash(&root.bytes(&catalog.reference, 2_097_152)?) != catalog.digest {
        return Err("Profile catalog changed during selection".into());
    }
    Ok(
        json!({"selected_id":selected,"resource_path":relative,"default_new_profile_path":".ptsip/profiles/main.ptsip.yaml","catalog_valid":true,"catalog_sha256":catalog.digest}),
    )
}
