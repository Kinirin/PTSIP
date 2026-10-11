use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};
use std::fs;
use std::path::{Component, Path, PathBuf};

// An explicit owner source is the input trust boundary. This verifier proves
// current bytes and declared relationships, never owner legitimacy or grants.
struct Source {
    root: PathBuf,
    hashes: BTreeMap<String, String>,
}

impl Source {
    fn read(&mut self, reference: &str) -> Result<Value, String> {
        let relative = Path::new(reference);
        if relative.is_absolute()
            || reference.contains('\\')
            || relative
                .components()
                .any(|c| !matches!(c, Component::Normal(_)))
        {
            return Err(format!("unregistered source path syntax: {reference}"));
        }
        let mut path = self.root.clone();
        for component in relative.components() {
            path.push(component);
            if fs::symlink_metadata(&path)
                .map_err(|e| e.to_string())?
                .file_type()
                .is_symlink()
            {
                return Err("source alias requires separate registered binding".into());
            }
        }
        let resolved = fs::canonicalize(&path).map_err(|e| e.to_string())?;
        if !resolved.starts_with(&self.root) || !resolved.is_file() {
            return Err("source reference escapes owner root".into());
        }
        let bytes = fs::read(&resolved).map_err(|e| e.to_string())?;
        if bytes.len() > 2_097_152 {
            return Err("source exceeds contract byte budget".into());
        }
        let hash = format!("{:x}", Sha256::digest(&bytes));
        if let Some(prior) = self.hashes.insert(reference.into(), hash.clone()) {
            if prior != hash {
                return Err("source changed during verification".into());
            }
        }
        serde_json::from_slice(&bytes)
            .map_err(|e| format!("invalid registered JSON source {reference}: {e}"))
    }

    fn bound(&mut self, reference: &Value) -> Result<Value, String> {
        let path = string(reference, "path")?;
        let value = self.read(path)?;
        if self.hashes.get(path).map(String::as_str) != Some(string(reference, "sha256")?) {
            return Err(format!("stale source digest: {path}"));
        }
        let identity = value
            .get("contract_id")
            .or_else(|| value.get("id"))
            .or_else(|| value.get("engine_id"));
        let version = value
            .get("version")
            .or_else(|| value.get("implementation_version"));
        if identity != reference.get("id") || version != reference.get("version") {
            return Err("reference identity/version mismatch".into());
        }
        Ok(value)
    }

    fn schema(&mut self, index: &Value, name: &str, value: &Value) -> Result<(), String> {
        let schema_ref = index["engine_schema_refs"][name]
            .as_str()
            .ok_or("unregistered schema route")?;
        let schema = self.read(schema_ref)?;
        jsonschema::validate(&schema, value).map_err(|e| format!("{name}: {e}"))
    }
}

fn string<'a>(v: &'a Value, key: &str) -> Result<&'a str, String> {
    v[key]
        .as_str()
        .filter(|s| !s.is_empty())
        .ok_or_else(|| format!("missing typed field: {key}"))
}
fn list<'a>(v: &'a Value, key: &str) -> Result<&'a Vec<Value>, String> {
    v[key]
        .as_array()
        .ok_or_else(|| format!("missing typed list: {key}"))
}
fn no_external_schema_refs(v: &Value) -> bool {
    match v {
        Value::Object(o) => {
            !o.contains_key("$ref")
                && !o.contains_key("$dynamicRef")
                && o.values().all(no_external_schema_refs)
        }
        Value::Array(a) => a.iter().all(no_external_schema_refs),
        _ => true,
    }
}

pub fn verify(input: &Value, select: bool) -> Result<Value, String> {
    let root = fs::canonicalize(string(input, "root")?).map_err(|e| e.to_string())?;
    let mut source = Source {
        root,
        hashes: BTreeMap::new(),
    };
    let index_ref = string(input, "index_ref")?;
    let index = source.read(index_ref)?;
    if index["schema_version"] != "ptsip-official-engine-index/v1"
        || index["policy_class"] != "PTSIP_SUPPORT_FEATURE"
    {
        return Err("unsupported owner index contract".into());
    }
    for (field, id) in [
        ("architecture_owner", "SFP-ARCH-0004"),
        ("contract_owner", "SFP-CNTR-0005"),
    ] {
        let policy = source.read(string(&index, field)?)?;
        if policy["policy_class"] != "PTSIP_SUPPORT_FEATURE"
            || policy["policy"]["id"] != id
            || policy["policy"]["status"] != "ACTIVE"
        {
            return Err("current Support owner unavailable".into());
        }
    }
    let federation = &index["federation"];
    let statics = source.read(string(federation, "static_registry_ref")?)?;
    let execution = source.read(string(federation, "execution_registry_ref")?)?;
    if statics["owner_id"] != federation["owner_id"]
        || execution["owner_id"] != federation["owner_id"]
        || statics["execution_permission"] != false
        || execution["execution_permission"] != false
    {
        return Err("federated owner or permission boundary mismatch".into());
    }
    let entries = list(&index, "official_engine_entries")?;
    let mut declared_ids = BTreeSet::new();
    let mut declared_capabilities = BTreeSet::new();
    for entry in entries {
        if !declared_ids.insert(string(entry, "engine_id")?.to_owned())
            || !declared_capabilities.insert(string(entry, "capability_id")?.to_owned())
        {
            return Err("ambiguous index owner identity".into());
        }
    }
    let mut scope = BTreeSet::new();
    if select {
        let mut pending = vec![string(input, "engine_id")?.to_owned()];
        while let Some(id) = pending.pop() {
            if scope.insert(id.clone()) {
                let entry = entries
                    .iter()
                    .find(|e| e["engine_id"] == id)
                    .ok_or("unregistered required engine")?;
                let manifest = source.read(string(entry, "manifest_ref")?)?;
                for dependency in list(&manifest, "dependencies")? {
                    pending.push(string(dependency, "provider_engine_id")?.to_owned());
                }
            }
        }
    } else {
        scope = declared_ids;
    }
    let mut ids = BTreeSet::new();
    let mut capabilities = BTreeMap::new();
    let mut registered = Vec::new();
    let mut manifests = BTreeMap::new();
    let mut invocable_count = 0;
    for entry in entries
        .iter()
        .filter(|e| scope.contains(e["engine_id"].as_str().unwrap_or("")))
    {
        let id = string(entry, "engine_id")?;
        if !ids.insert(id.to_owned()) {
            return Err("ambiguous engine identity".into());
        }
        let manifest = source.read(string(entry, "manifest_ref")?)?;
        source.schema(&index, "core-manifest", &manifest)?;
        if manifest["engine_id"] != entry["engine_id"]
            || manifest["implementation_version"] != entry["version"]
            || manifest["origin"] != statics["origin"]
        {
            return Err("manifest origin or implementation identity mismatch".into());
        }
        let matches: Vec<_> = list(&statics, "entries")?
            .iter()
            .filter(|s| {
                s["engine_id"] == entry["engine_id"] && s["manifest_ref"] == entry["manifest_ref"]
            })
            .collect();
        if matches.len() != 1 {
            return Err("missing or ambiguous owner static registration".into());
        }
        let contract = source.read(string(entry, "contract_ref")?)?;
        source.schema(&index, "capability-contract", &contract)?;
        if contract["engine_id"] != entry["engine_id"]
            || contract["contract_id"] != entry["capability_id"]
            || contract["semantic_owner"] != entry["kind"]
            || contract["execution_permission_from_declaration"] != false
        {
            return Err("capability normative owner mismatch".into());
        }
        let mut operations = BTreeSet::new();
        for op in list(&contract, "operations")? {
            if !operations.insert(string(op, "operation_id")?.to_owned()) {
                return Err("ambiguous operation identity".into());
            }
            for schema in ["typed_input", "typed_output"] {
                if !no_external_schema_refs(&op[schema]) {
                    return Err("unregistered schema reference".into());
                }
                jsonschema::validator_for(&op[schema]).map_err(|e| e.to_string())?;
            }
        }
        let declared: BTreeSet<_> = list(entry, "operations")?
            .iter()
            .map(|v| v.as_str().unwrap_or("").to_owned())
            .collect();
        if declared != operations {
            return Err("index operation projection drift".into());
        }
        let mut matched = false;
        for reference in list(&manifest, "capabilities")? {
            let capability = source.bound(reference)?;
            if capabilities
                .insert(
                    string(&capability, "contract_id")?.to_owned(),
                    (id.to_owned(), string(&capability, "version")?.to_owned()),
                )
                .is_some()
            {
                return Err("duplicate capability normative owner".into());
            }
            if reference["path"] == entry["contract_ref"] && capability == contract {
                matched = true;
            }
        }
        if !matched {
            return Err("current capability not bound by the owner manifest".into());
        }
        for extension in list(&manifest, "extensions")? {
            let value = source.bound(extension)?;
            if value["engine_id"] != entry["engine_id"] {
                return Err("extension redefines another engine".into());
            }
        }
        let invocable = !operations.is_empty();
        let routes: Vec<_> = list(&execution, "bindings")?
            .iter()
            .filter(|r| r["engine_id"] == entry["engine_id"])
            .collect();
        if invocable && routes.len() != 1 {
            return Err("required invocation route missing or ambiguous".into());
        }
        if !invocable && !routes.is_empty() {
            return Err("static-only engine has unexpected invocation route".into());
        }
        let mut endpoint = Value::Null;
        if invocable {
            invocable_count += 1;
            if routes[0]["binding_ref"] != entry["execution_binding_ref"] {
                return Err("execution route projection drift".into());
            }
            let bindings = source.read(string(routes[0], "binding_ref")?)?;
            let values = list(&bindings, "bindings")?;
            if values.len() != 1 {
                return Err("ambiguous capability invocation binding".into());
            }
            let binding = &values[0];
            source.schema(&index, "execution-binding", binding)?;
            if binding["engine_id"] != entry["engine_id"]
                || binding["engine_version"] != manifest["implementation_version"]
                || binding["capability_id"] != contract["contract_id"]
                || binding["contract_version"] != contract["version"]
                || binding["permission_grant"] != false
            {
                return Err("stale capability invocation binding".into());
            }
            source.bound(&binding["transport_ref"])?;
            endpoint = binding["entrypoint"].clone();
        }
        registered.push(json!({"engine_id":entry["engine_id"],"origin":manifest["origin"],"implementation_version":manifest["implementation_version"],"capability_id":contract["contract_id"],"contract_version":contract["version"],"entrypoint":endpoint,"registration_eligible":true,"execution_authorized":false}));
        manifests.insert(id.to_owned(), manifest);
    }
    if !select
        && (list(&statics, "entries")?.len() != ids.len()
            || list(&execution, "bindings")?.len() != invocable_count)
    {
        return Err("unmapped owner registration".into());
    }
    for (consumer, manifest) in &manifests {
        for dependency in list(manifest, "dependencies")? {
            let provider = string(dependency, "provider_engine_id")?;
            let capability = string(dependency, "capability_id")?;
            if capabilities.get(capability)
                != Some(&(
                    provider.to_owned(),
                    string(dependency, "contract_version")?.to_owned(),
                ))
            {
                return Err("unknown or version-incompatible dependency".into());
            }
            let evidence = source.bound(&dependency["compatibility_ref"])?;
            if evidence["provider_engine_id"] != provider
                || evidence["consumer_engine_id"] != consumer.as_str()
                || evidence["outcome"] != "PASS"
            {
                return Err("required provider/consumer compatibility evidence unavailable".into());
            }
            for reference in list(&evidence, "source_refs")? {
                source.bound(reference)?;
            }
        }
    }
    // Re-read every admitted source before issuing a revision-bound derived result.
    let refs: Vec<_> = source.hashes.keys().cloned().collect();
    for path in &refs {
        source.read(path)?;
    }
    let revision = format!(
        "{:x}",
        Sha256::digest(serde_json::to_vec(&source.hashes).map_err(|e| e.to_string())?)
    );
    if let Some(expected) = input["expected_revision"].as_str() {
        if expected != revision {
            return Err("cached source revision is no longer current".into());
        }
    }
    let source_refs: Vec<_> = source
        .hashes
        .iter()
        .map(|(path, sha)| json!({"path":path,"sha256":sha}))
        .collect();
    let selected = if select {
        let matches: Vec<_> = registered
            .iter()
            .filter(|r| {
                r["engine_id"] == input["engine_id"]
                    && r["capability_id"] == input["capability_id"]
                    && r["implementation_version"] == input["version"]
                    && r["capability_id"] == input["contract_id"]
            })
            .collect();
        if matches.len() != 1 {
            return Err("unknown or ambiguous current selected capability".into());
        }
        matches[0].clone()
    } else {
        Value::Null
    };
    Ok(
        json!({"source_revision":revision,"registered_engines":registered,"source_refs":source_refs,"selected":selected,"derived_authority":false,"owner_legitimacy_asserted":false,"execution_authorized":false}),
    )
}
