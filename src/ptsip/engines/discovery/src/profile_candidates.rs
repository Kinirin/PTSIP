use crate::source::{self, Root};
use serde_json::{json, Value};
use std::collections::BTreeSet;
use std::fs;

pub const ALGORITHM: &str =
    "READ_EXPLICIT_PROFILE_BINDINGS_AND_CONTENT_HASHES_WITHOUT_LINEAGE_INFERENCE";

fn diagnostic(code: &str, message: impl Into<String>, path: &str) -> Value {
    json!({"code":code,"message":message.into(),"path":path})
}

fn binding(root: &Root, reference: &str, temporary: bool) -> Result<Value, Value> {
    if reference.contains(['\\', ':'])
        || reference
            .split('/')
            .any(|part| matches!(part, "" | "." | ".."))
    {
        return Err(diagnostic(
            "PROFILE_PATH_ERROR",
            "Profile path must be canonical and relative to the selected root",
            reference,
        ));
    }
    let raw = root
        .bytes(reference, 2_097_152)
        .map_err(|error| diagnostic("PROFILE_READ_ERROR", error, reference))?;
    let payload = source::parse_yaml(raw.strip_prefix(b"\xef\xbb\xbf").unwrap_or(&raw))
        .map_err(|error| diagnostic("INVALID_PROFILE_YAML", error, reference))?;
    if !payload.is_object() {
        return Err(diagnostic(
            "INVALID_PROFILE_YAML",
            "Profile root must be a mapping",
            reference,
        ));
    }
    let version = payload
        .pointer("/ptsip/version")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .ok_or_else(|| {
            diagnostic(
                "MISSING_PROFILE_VERSION",
                "Profile has no ptsip.version identity",
                reference,
            )
        })?;
    let revision = payload
        .pointer("/ptsip/specification/revision")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .ok_or_else(|| {
            diagnostic(
                "MISSING_SPEC_REVISION",
                "Profile has no specification revision",
                reference,
            )
        })?;
    let specification = payload
        .pointer("/ptsip/specification/source")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty());
    Ok(
        json!({"path":reference,"declared_version":version,"specification_revision":revision,
        "specification_source":specification,"content_sha256":source::hash(&raw),"temporary":temporary}),
    )
}

fn root_candidates(root: &Root) -> Result<Vec<String>, String> {
    let mut paths = Vec::new();
    for entry in fs::read_dir(root.directory()).map_err(|error| error.to_string())? {
        let entry = entry.map_err(|error| error.to_string())?;
        let name = entry.file_name();
        let Some(name) = name.to_str() else {
            continue;
        };
        if name.starts_with("ptsip_") && name.ends_with(".yaml") {
            let kind = entry.file_type().map_err(|error| error.to_string())?;
            if !kind.is_dir() {
                paths.push(name.to_string());
            }
        }
    }
    paths.sort();
    Ok(paths)
}

pub fn discover(input: &Value) -> Result<Value, String> {
    let root = Root::new(source::text(input, "root")?)?;
    let source_path = input["source_path"].as_str().unwrap_or("ptsip.yaml");
    let mut paths = match input["candidate_paths"].as_array() {
        Some(values) => values
            .iter()
            .map(|value| {
                value
                    .as_str()
                    .map(str::to_owned)
                    .ok_or("candidate path must be a string")
            })
            .collect::<Result<Vec<_>, _>>()?,
        None => root_candidates(&root)?,
    };
    paths.sort();
    let mut diagnostics = Vec::new();
    let mut seen = BTreeSet::from([source_path.to_owned()]);
    let mut source_binding = match root.exists(source_path) {
        Ok(true) => binding(&root, source_path, false),
        Ok(false) => Err(diagnostic(
            "NO_CANONICAL_PROFILE",
            "Selected Profile source is missing",
            source_path,
        )),
        Err(error) => Err(diagnostic("PROFILE_PATH_ERROR", error, source_path)),
    }
    .map_err(|error| diagnostics.push(error))
    .ok();
    let mut candidates = Vec::new();
    for path in paths {
        if !seen.insert(path.clone()) {
            diagnostics.push(diagnostic(
                "DUPLICATE_PROFILE_PATH",
                "Profile path was supplied more than once",
                &path,
            ));
            continue;
        }
        match binding(&root, &path, true) {
            Ok(value) => candidates.push(value),
            Err(error) => diagnostics.push(error),
        }
    }
    let mut unchanged = |value: &Value| {
        let path = value["path"].as_str().expect("bound Profile path");
        match root.bytes(path, 2_097_152) {
            Ok(raw) if value["content_sha256"] == source::hash(&raw) => true,
            _ => {
                diagnostics.push(diagnostic(
                    "PROFILE_CONTENT_CHANGED",
                    "Profile changed during discovery",
                    path,
                ));
                false
            }
        }
    };
    source_binding = source_binding.filter(&mut unchanged);
    candidates.retain(unchanged);
    Ok(json!({"source":source_binding,"candidates":candidates,"diagnostics":diagnostics}))
}
