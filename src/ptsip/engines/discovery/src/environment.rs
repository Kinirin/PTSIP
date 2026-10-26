//! Read-only diagnostics of the seven official engines in the selected installation.
//! Local source agreement is not release attestation or execution permission.
use crate::source::{self, Root};
use serde_json::{json, Value};
use std::collections::{BTreeMap, BTreeSet};
use std::fs;
use std::path::Path;

const OFFICIAL: [(&str, &str); 7] = [
    ("CONTEXT", "urn:uuid:0ca335ff-48b2-4358-99e4-3eeb48e94af3"),
    ("STORAGE", "urn:uuid:d2b72bd9-63e1-4816-b06e-f516f01aee2e"),
    (
        "VALIDATION",
        "urn:uuid:44258a79-af23-4ce9-94c9-6d23de29bd21",
    ),
    ("DISCOVERY", "urn:uuid:e51f2248-0db7-4f5c-aa4b-e278a133bc50"),
    ("PLANNING", "urn:uuid:7a19982e-90ba-4388-9da0-90f57299e7d7"),
    ("EXECUTION", "urn:uuid:b66de0f9-c454-4dbc-b43c-a2e695184671"),
    (
        "PROJECTION",
        "urn:uuid:befe7a8f-786c-4851-babe-8a154a2324d0",
    ),
];

fn text<'a>(v: &'a Value, key: &str) -> Result<&'a str, String> {
    source::text(v, key)
}
fn list<'a>(v: &'a Value, key: &str) -> Result<&'a Vec<Value>, String> {
    v[key]
        .as_array()
        .ok_or_else(|| format!("missing array: {key}"))
}
// This is a declared packaging projection, never a guessed repository path.
fn resource(reference: &str) -> Result<&str, String> {
    reference
        .strip_prefix("src/ptsip/")
        .filter(|p| p.starts_with("engines/"))
        .ok_or_else(|| "reference is outside the installed official engine resources".into())
}
fn read(package: &mut Root, reference: &str) -> Result<(Value, String), String> {
    package.read(resource(reference)?)
}
fn bound(package: &mut Root, reference: &Value) -> Result<Value, String> {
    let (v, hash) = read(package, text(reference, "path")?)?;
    let id = v
        .get("contract_id")
        .or_else(|| v.get("engine_id"))
        .or_else(|| v.get("id"));
    let version = v.get("version").or_else(|| v.get("implementation_version"));
    if id != reference.get("id")
        || version != reference.get("version")
        || reference["sha256"] != hash
    {
        return Err("installed source identity, version or hash differs".into());
    }
    Ok(v)
}
fn target() -> &'static str {
    match (std::env::consts::OS, std::env::consts::ARCH) {
        ("windows", "x86_64") => "x86_64-pc-windows-msvc",
        ("linux", "x86_64") => "x86_64-unknown-linux-gnu",
        ("macos", "aarch64") => "aarch64-apple-darwin",
        _ => "UNSUPPORTED_HOST",
    }
}
fn metadata_version(bytes: &[u8]) -> Result<String, String> {
    let raw = std::str::from_utf8(bytes).map_err(|e| e.to_string())?;
    let headers: Vec<_> = raw.lines().take_while(|s| !s.trim().is_empty()).collect();
    let names: Vec<_> = headers
        .iter()
        .filter_map(|s| s.strip_prefix("Name: "))
        .collect();
    let versions: Vec<_> = headers
        .iter()
        .filter_map(|s| s.strip_prefix("Version: "))
        .collect();
    if names.len() != 1
        || !names[0].trim().eq_ignore_ascii_case("PTSIP")
        || versions.len() != 1
        || versions[0].trim().is_empty()
    {
        return Err("selected distribution is not an unambiguous PTSIP installation".into());
    }
    Ok(versions[0].trim().to_owned())
}

struct Checked {
    row: Value,
    manifest: Value,
    contract: Value,
    contract_hash: String,
}

fn schema(package: &mut Root, index: &Value, name: &str, value: &Value) -> Result<(), String> {
    let reference = index["engine_schema_refs"][name]
        .as_str()
        .ok_or("registered engine schema unavailable")?;
    let (definition, _) = read(package, reference)?;
    source::inline(&definition)?;
    jsonschema::validate(&definition, value).map_err(|e| format!("{name}: {e}"))
}

fn inspect_engine(
    package: &mut Root,
    scripts: &mut Root,
    index: &Value,
    entry: &Value,
    bundle: &Value,
    progress: &mut Value,
    failed_dimension: &mut &'static str,
) -> Result<Checked, String> {
    let name = text(entry, "kind")?.to_lowercase();
    let id = text(entry, "engine_id")?;
    let version = text(entry, "version")?;
    let (manifest, _) = read(package, text(entry, "manifest_ref")?)?;
    schema(package, index, "core-manifest", &manifest)?;
    if manifest["engine_id"] != id
        || manifest["implementation_version"] != version
        || manifest["origin"] != "OFFICIAL"
    {
        return Err("official manifest identity or version differs".into());
    }
    *failed_dimension = "contract_agreement";
    let (contract, contract_hash) = read(package, text(entry, "contract_ref")?)?;
    schema(package, index, "capability-contract", &contract)?;
    if contract["engine_id"] != id
        || contract["semantic_owner"] != entry["kind"]
        || contract["contract_id"] != entry["capability_id"]
        || contract["execution_permission_from_declaration"] != false
    {
        return Err("capability identity, owner or permission boundary differs".into());
    }
    let mut matched = false;
    for r in list(&manifest, "capabilities")? {
        let c = bound(package, r)?;
        if r["path"] == entry["contract_ref"] && c == contract {
            matched = true;
        }
    }
    if !matched {
        return Err("capability is not bound to its current manifest".into());
    }
    let (bindings, _) = read(package, text(entry, "execution_binding_ref")?)?;
    let bs = list(&bindings, "bindings")?;
    if bs.len() != 1 {
        return Err("ambiguous native entrypoint binding".into());
    }
    let b = &bs[0];
    schema(package, index, "execution-binding", b)?;
    let executable = format!("ptsip-engine-{name}");
    if b["engine_id"] != id
        || b["engine_version"] != version
        || b["capability_id"] != contract["contract_id"]
        || b["contract_version"] != contract["version"]
        || b["entrypoint"] != executable
        || b["permission_grant"] != false
    {
        return Err("native binding identity, version or permission boundary differs".into());
    }
    *failed_dimension = "transport_agreement";
    let transport = bound(package, &b["transport_ref"])?;
    let (local_transport, _) = package.read(&format!("engines/{name}/transport.json"))?;
    if local_transport != transport
        || b["transport_ref"]["path"] != index["transport_contract_ref"]
        || transport["contract_id"] != "urn:ptsip:engine-transport:json-stdio:v1"
        || transport["version"] != "1.0.0"
        || transport["request_encoding"] != "ONE_UTF8_JSON_OBJECT_ON_STDIN_EOF"
        || transport["response_encoding"] != "ONE_UTF8_JSON_OBJECT_ON_STDOUT"
        || transport["execution_permission_from_binding"] != false
    {
        return Err("unsupported or inconsistent installed transport contract".into());
    }
    source::inline(&transport["request_schema"])?;
    source::inline(&transport["response_schema"])?;
    progress["transport_agreement"] = json!("VERIFIED");
    *failed_dimension = "contract_agreement";
    let mut operations = Vec::new();
    let mut ids = BTreeSet::new();
    for op in list(&contract, "operations")? {
        let operation = text(op, "operation_id")?;
        if !ids.insert(operation.to_owned()) {
            return Err("duplicate operation identity".into());
        }
        source::inline(&op["typed_input"])?;
        source::inline(&op["typed_output"])?;
        let request = json!({"engine_id":id,"engine_version":version,"contract_id":contract["contract_id"],"contract_version":contract["version"],"operation_id":operation,"input":{}});
        jsonschema::validate(&transport["request_schema"], &request).map_err(|e| e.to_string())?;
        operations.push(json!({"operation_id":operation,"required_inputs":op["typed_input"]["required"].as_array().cloned().unwrap_or_default(),"input_schema":op["typed_input"],"output_schema":op["typed_output"],"request_template":request,"template_is_executable_request":false,"runtime_probed":false,"side_effects":op["side_effects"],"declared_permission_requirements":op["declared_permission_requirements"]}));
    }
    let declared: BTreeSet<_> = list(entry, "operations")?
        .iter()
        .map(|v| v.as_str().unwrap_or("").to_owned())
        .collect();
    if declared != ids || declared.len() != list(entry, "operations")?.len() {
        return Err("operation catalog differs from its contract".into());
    }
    progress["contract_agreement"] = json!("VERIFIED");
    *failed_dimension = "installation_integrity";
    let origin = &bundle["engines"][&name];
    let filename = format!("{executable}{}", if cfg!(windows) { ".exe" } else { "" });
    if origin["schema_version"] != "ptsip-independent-wheel-origin/v1"
        || origin["engine_id"] != id
        || origin["engine_version"] != version
        || origin["host_target"] != target()
        || origin["entrypoint"] != filename
        || origin["execution_authorized"] != false
        || !origin["source_sha"].as_str().is_some_and(|s| {
            s.len() == 40
                && s.bytes()
                    .all(|c| c.is_ascii_hexdigit() && !c.is_ascii_uppercase())
        })
    {
        return Err("installed bundle origin, host or version differs".into());
    }
    let binary = scripts.artifact(&json!({"path":filename,"sha256":origin["binary_sha256"]}))?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        if fs::metadata(&binary)
            .map_err(|e| e.to_string())?
            .permissions()
            .mode()
            & 0o111
            == 0
        {
            return Err("installed native entrypoint is not executable".into());
        }
    }
    progress["installation_integrity"] = json!("VERIFIED");
    progress["version"] = json!(version);
    progress["contract_id"] = contract["contract_id"].clone();
    progress["contract_version"] = contract["version"].clone();
    progress["entrypoint"] = json!(binary);
    progress["operations"] = json!(operations);
    Ok(Checked {
        row: progress.clone(),
        manifest,
        contract,
        contract_hash,
    })
}

fn dependency(
    package: &mut Root,
    consumer: &str,
    dep: &Value,
    all: &BTreeMap<String, Checked>,
) -> Result<&'static str, String> {
    let provider = text(dep, "provider_engine_id")?;
    let p = all
        .get(provider)
        .ok_or("declared provider is unavailable or unverified")?;
    let c = all.get(consumer).ok_or("declared consumer unavailable")?;
    if dep["capability_id"] != p.contract["contract_id"]
        || dep["contract_version"] != p.contract["version"]
    {
        return Ok("INCOMPATIBLE");
    }
    let evidence = bound(package, &dep["compatibility_ref"])?;
    if evidence["provider_engine_id"] != provider
        || evidence["consumer_engine_id"] != consumer
        || evidence["outcome"] != "PASS"
    {
        return Err("applicable dependency compatibility evidence is missing or failed".into());
    }
    let refs = list(&evidence, "source_refs")?;
    for checked in [p, c] {
        let matches: Vec<_> = refs
            .iter()
            .filter(|r| r["id"] == checked.contract["contract_id"])
            .collect();
        if matches.len() != 1
            || matches[0]["version"] != checked.contract["version"]
            || matches[0]["sha256"] != checked.contract_hash
            || bound(package, matches[0])? != checked.contract
        {
            return Err("dependency evidence is not bound to both current contracts".into());
        }
    }
    Ok("VERIFIED")
}

pub fn assess(input: &Value) -> Result<Value, String> {
    if target() == "UNSUPPORTED_HOST" {
        return Err("unsupported diagnostic host".into());
    }
    let package_path = fs::canonicalize(text(input, "package_root")?).map_err(|e| e.to_string())?;
    let distribution_path =
        fs::canonicalize(text(input, "distribution_root")?).map_err(|e| e.to_string())?;
    if package_path.parent() != distribution_path.parent() {
        return Err(
            "package and distribution metadata must belong to one selected installation".into(),
        );
    }
    let mut package = Root::new(text(input, "package_root")?)?;
    let mut distribution = Root::new(text(input, "distribution_root")?)?;
    let mut scripts = Root::new(text(input, "scripts_root")?)?;
    let metadata = distribution.bytes("METADATA", 2_097_152)?;
    let package_version = metadata_version(&metadata)?;
    let (bundle, _) = distribution.read("ptsip-engines.json")?;
    let expected_keys: BTreeSet<_> = OFFICIAL
        .iter()
        .map(|(kind, _)| kind.to_lowercase())
        .collect();
    let keys: BTreeSet<_> = bundle["engines"]
        .as_object()
        .ok_or("official engine origins absent")?
        .keys()
        .cloned()
        .collect();
    let platform = match target() {
        "x86_64-pc-windows-msvc" => "windows-x64",
        "x86_64-unknown-linux-gnu" => "linux-x64",
        "aarch64-apple-darwin" => "macos-arm64",
        _ => "UNSUPPORTED_HOST",
    };
    if bundle["schema_version"] != "ptsip-bundled-engine-origins/v1"
        || bundle["package_version"] != package_version
        || bundle["platform"] != platform
        || bundle["execution_authorized"] != false
        || keys != expected_keys
    {
        return Err("selected Package bundle is incomplete or host-incompatible".into());
    }
    let (index, _) = package.read("engines/index.json")?;
    if index["schema_version"] != "ptsip-official-engine-index/v1"
        || list(&index, "official_engine_entries")?.len() != 7
    {
        return Err("exact seven-engine official index required".into());
    }
    let mut all = BTreeMap::new();
    let mut failed = BTreeMap::new();
    for (kind, id) in OFFICIAL {
        let mut progress = json!({"engine":kind,"engine_id":id,"installation_integrity":"UNVERIFIED","contract_agreement":"UNVERIFIED","transport_agreement":"UNVERIFIED","dependency_compatibility":"UNVERIFIED","issues":[],"operations":[]});
        let mut failed_dimension = "installation_integrity";
        let entries: Vec<_> = list(&index, "official_engine_entries")?
            .iter()
            .filter(|e| e["kind"] == kind && e["engine_id"] == id)
            .collect();
        let checked = if entries.len() == 1 {
            inspect_engine(
                &mut package,
                &mut scripts,
                &index,
                entries[0],
                &bundle,
                &mut progress,
                &mut failed_dimension,
            )
        } else {
            Err("missing or ambiguous official registration".into())
        };
        match checked {
            Ok(c) => {
                all.insert(id.to_owned(), c);
            }
            Err(e) => {
                progress[failed_dimension] = json!("FAILED");
                progress["issues"] = json!([e]);
                failed.insert(id.to_owned(), progress);
            }
        }
    }
    let mut rows = Vec::new();
    let mut relationships = Vec::new();
    for (_, id) in OFFICIAL {
        if let Some(failure) = failed.get(id) {
            rows.push(failure.clone());
            continue;
        }
        let c = &all[id];
        let mut row = c.row.clone();
        let deps = list(&c.manifest, "dependencies")?;
        if deps.is_empty() {
            row["issues"]=json!(["No engine dependency relationships are declared; pairwise compatibility is unverified."]);
        } else {
            let mut verified = true;
            let mut seen = BTreeSet::new();
            for d in deps {
                let result = if seen.insert((
                    d["provider_engine_id"].to_string(),
                    d["capability_id"].to_string(),
                )) {
                    dependency(&mut package, id, d, &all)
                } else {
                    Err("duplicate dependency relationship".into())
                };
                let (status, reason) = match result {
                    Ok("VERIFIED") => ("VERIFIED", Value::Null),
                    Ok(_) => {
                        verified = false;
                        ("INCOMPATIBLE",json!("Declared provider contract version does not match the installed provider."))
                    }
                    Err(e) => {
                        verified = false;
                        ("UNVERIFIED", json!(e))
                    }
                };
                if status == "INCOMPATIBLE" {
                    row["dependency_compatibility"] = json!("INCOMPATIBLE");
                }
                relationships.push(json!({"consumer_engine_id":id,"provider_engine_id":d["provider_engine_id"],"capability_id":d["capability_id"],"status":status,"reason":reason}));
            }
            if verified {
                row["dependency_compatibility"] = json!("VERIFIED");
            }
        }
        rows.push(row);
    }
    package.finish()?;
    distribution.finish()?;
    scripts.finish()?;
    if distribution.bytes("METADATA", 2_097_152)? != metadata {
        return Err("distribution metadata changed during diagnostics".into());
    }
    let verified = failed.is_empty()
        && rows
            .iter()
            .all(|r| r["dependency_compatibility"] == "VERIFIED");
    let incompatible =
        !failed.is_empty() || relationships.iter().any(|r| r["status"] == "INCOMPATIBLE");
    Ok(
        json!({"schema_version":"ptsip-official-engine-compatibility/v1","scope":"SEVEN_OFFICIAL_ENGINES_DECLARED_RELATIONSHIPS_ONLY","status":if incompatible{"INCOMPATIBLE"}else if verified{"VERIFIED"}else{"UNVERIFIED"},"package_version":package_version,"host_target":target(),"package_root":package_path,"scripts_root":Path::new(text(input,"scripts_root")?),"engine_count":7,"installation_integrity":if failed.is_empty(){"VERIFIED"}else{"UNVERIFIED"},"entries":rows,"relationships":relationships,"pairwise_compatibility_guaranteed":false,"release_attested":false,"execution_authorized":false,"evidence_authentication":"LOCAL_SOURCE_AGREEMENT_ONLY"}),
    )
}
