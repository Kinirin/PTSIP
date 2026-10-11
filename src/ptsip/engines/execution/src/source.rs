// Engine-owned source verification. No imports from another engine implementation.
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::{Map, Number, Value};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fmt;
use std::fs;
use std::path::{Component, Path, PathBuf};
pub const SCHEMAS: &str = include_str!("../runtime-contracts.json");
pub fn text<'a>(value: &'a Value, key: &str) -> Result<&'a str, String> {
    value[key]
        .as_str()
        .filter(|s| !s.is_empty())
        .ok_or_else(|| format!("missing source field: {key}"))
}
pub fn hash(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}
struct Strict(Value);
impl<'de> Deserialize<'de> for Strict {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct JsonVisitor;
        impl<'de> Visitor<'de> for JsonVisitor {
            type Value = Strict;
            fn expecting(&self, f: &mut fmt::Formatter) -> fmt::Result {
                write!(f, "unambiguous JSON")
            }
            fn visit_bool<E: de::Error>(self, v: bool) -> Result<Strict, E> {
                Ok(Strict(Value::Bool(v)))
            }
            fn visit_i64<E: de::Error>(self, v: i64) -> Result<Strict, E> {
                Ok(Strict(Value::Number(v.into())))
            }
            fn visit_u64<E: de::Error>(self, v: u64) -> Result<Strict, E> {
                Ok(Strict(Value::Number(v.into())))
            }
            fn visit_f64<E: de::Error>(self, v: f64) -> Result<Strict, E> {
                Number::from_f64(v)
                    .map(|n| Strict(Value::Number(n)))
                    .ok_or_else(|| E::custom("invalid JSON number"))
            }
            fn visit_str<E: de::Error>(self, v: &str) -> Result<Strict, E> {
                Ok(Strict(Value::String(v.into())))
            }
            fn visit_string<E: de::Error>(self, v: String) -> Result<Strict, E> {
                Ok(Strict(Value::String(v)))
            }
            fn visit_unit<E: de::Error>(self) -> Result<Strict, E> {
                Ok(Strict(Value::Null))
            }
            fn visit_seq<A: SeqAccess<'de>>(self, mut a: A) -> Result<Strict, A::Error> {
                let mut values = Vec::new();
                while let Some(Strict(v)) = a.next_element()? {
                    values.push(v);
                }
                Ok(Strict(Value::Array(values)))
            }
            fn visit_map<A: MapAccess<'de>>(self, mut a: A) -> Result<Strict, A::Error> {
                let mut values = Map::new();
                while let Some(k) = a.next_key::<String>()? {
                    if values.contains_key(&k) {
                        return Err(de::Error::custom(format!("duplicate JSON key: {k}")));
                    }
                    let Strict(v) = a.next_value()?;
                    values.insert(k, v);
                }
                Ok(Strict(Value::Object(values)))
            }
        }
        d.deserialize_any(JsonVisitor)
    }
}
pub fn parse(bytes: &[u8]) -> Result<Value, String> {
    let mut decoder = serde_json::Deserializer::from_slice(bytes);
    let Strict(value) = Strict::deserialize(&mut decoder).map_err(|e| e.to_string())?;
    decoder.end().map_err(|e| e.to_string())?;
    Ok(value)
}
pub fn inline(schema: &Value) -> Result<(), String> {
    fn closed(v: &Value) -> bool {
        match v {
            Value::Object(o) => {
                !o.contains_key("$ref") && !o.contains_key("$dynamicRef") && o.values().all(closed)
            }
            Value::Array(a) => a.iter().all(closed),
            _ => true,
        }
    }
    if !closed(schema) {
        return Err("unresolved or external schema reference".into());
    }
    jsonschema::validator_for(schema).map_err(|e| e.to_string())?;
    Ok(())
}
pub fn schema(name: &str, value: &Value) -> Result<(), String> {
    let definitions: Value = serde_json::from_str(SCHEMAS).map_err(|e| e.to_string())?;
    jsonschema::validate(&definitions[name], value).map_err(|e| format!("{name}: {e}"))
}
pub struct Root {
    path: PathBuf,
    sources: BTreeMap<String, String>,
}
impl Root {
    pub fn new(path: &str) -> Result<Self, String> {
        Ok(Self {
            path: fs::canonicalize(path).map_err(|e| e.to_string())?,
            sources: BTreeMap::new(),
        })
    }
    pub fn directory(&self) -> &Path {
        &self.path
    }
    pub fn path(&self, reference: &str) -> Result<PathBuf, String> {
        let relative = Path::new(reference);
        if reference.is_empty()
            || reference.contains('\\')
            || relative.is_absolute()
            || relative
                .components()
                .any(|c| !matches!(c, Component::Normal(_)))
        {
            return Err("source must be normal owner-relative path".into());
        }
        let mut path = self.path.clone();
        for component in relative.components() {
            path.push(component);
            if fs::symlink_metadata(&path)
                .map_err(|e| e.to_string())?
                .file_type()
                .is_symlink()
            {
                return Err("unregistered source alias".into());
            }
        }
        let path = fs::canonicalize(path).map_err(|e| e.to_string())?;
        if !path.starts_with(&self.path) {
            return Err("source escapes selected owner root".into());
        }
        Ok(path)
    }
    pub fn bytes(&self, reference: &str, max: u64) -> Result<Vec<u8>, String> {
        let path = self.path(reference)?;
        let metadata = fs::metadata(&path).map_err(|e| e.to_string())?;
        if !metadata.is_file() || metadata.len() > max {
            return Err("source file or byte budget invalid".into());
        }
        fs::read(path).map_err(|e| e.to_string())
    }
    pub fn read(&mut self, reference: &str) -> Result<(Value, String), String> {
        let bytes = self.bytes(reference, 2_097_152)?;
        let digest = hash(&bytes);
        if self.sources.get(reference).is_some_and(|h| h != &digest) {
            return Err("source changed during resolution".into());
        }
        self.sources.insert(reference.into(), digest.clone());
        Ok((parse(&bytes)?, digest))
    }
    pub fn file(&mut self, reference: &Value) -> Result<Value, String> {
        let (value, digest) = self.read(text(reference, "path")?)?;
        if reference["sha256"] != digest {
            return Err("source digest mismatch".into());
        }
        Ok(value)
    }
    pub fn bound(&mut self, reference: &Value) -> Result<Value, String> {
        let value = self.file(reference)?;
        let id = value
            .get("id")
            .or_else(|| value.get("contract_id"))
            .or_else(|| value.get("engine_id"));
        let version = value
            .get("version")
            .or_else(|| value.get("implementation_version"));
        if id != Some(&reference["id"]) || version != Some(&reference["version"]) {
            return Err("source identity or version mismatch".into());
        }
        Ok(value)
    }
    pub fn artifact(&mut self, reference: &Value) -> Result<PathBuf, String> {
        let name = text(reference, "path")?;
        let bytes = self.bytes(name, 268_435_456)?;
        let digest = hash(&bytes);
        if reference["sha256"] != digest {
            return Err("native artifact digest mismatch".into());
        }
        self.sources.insert(name.into(), digest);
        self.path(name)
    }
    pub fn finish(&self) -> Result<(), String> {
        for (name, digest) in &self.sources {
            if hash(&self.bytes(name, 268_435_456)?) != *digest {
                return Err(format!("source changed: {name}"));
            }
        }
        Ok(())
    }
}

pub fn invocation(
    root: &mut Root,
    registry_reference: &Value,
    id: &str,
) -> Result<(Value, Value, std::path::PathBuf), String> {
    let registry = root.bound(registry_reference)?;
    schema("invocation_registry", &registry)?;
    let matches: Vec<_> = registry["invocations"]
        .as_array()
        .ok_or("invocation registrations required")?
        .iter()
        .filter(|v| v["id"] == id)
        .collect();
    if matches.len() != 1 {
        return Err("invocation registration missing or ambiguous".into());
    }
    let invocation = matches[0].clone();
    let manifest = root.bound(&invocation["manifest_ref"])?;
    let contract = root.bound(&invocation["contract_ref"])?;
    let bindings = root.file(&invocation["binding_ref"])?;
    let stored = root.bound(&invocation["request_ref"])?;
    schema("request", &stored)?;
    let request = stored["request"].clone();
    if manifest["engine_id"] != invocation["engine_id"]
        || manifest["implementation_version"] != invocation["engine_version"]
        || contract["contract_id"] != invocation["capability_id"]
        || contract["version"] != invocation["contract_version"]
    {
        return Err("current invocation owner identity/version mismatch".into());
    }
    let capabilities = manifest["capabilities"]
        .as_array()
        .ok_or("manifest capabilities required")?;
    if capabilities
        .iter()
        .filter(|r| *r == &invocation["contract_ref"])
        .count()
        != 1
    {
        return Err("contract is not exactly bound by selected engine manifest".into());
    }
    for (k, field) in [
        ("engine_id", "engine_id"),
        ("engine_version", "engine_version"),
        ("contract_id", "capability_id"),
        ("contract_version", "contract_version"),
        ("operation_id", "operation_id"),
    ] {
        if request[k] != invocation[field] {
            return Err(format!("registered request identity mismatch: {k}"));
        }
    }
    let matching: Vec<_> = bindings["bindings"]
        .as_array()
        .ok_or("entrypoint bindings required")?
        .iter()
        .filter(|b| {
            b["engine_id"] == invocation["engine_id"]
                && b["engine_version"] == invocation["engine_version"]
                && b["capability_id"] == invocation["capability_id"]
                && b["contract_version"] == invocation["contract_version"]
        })
        .collect();
    if matching.len() != 1 {
        return Err("native entrypoint binding missing or ambiguous".into());
    }
    let binding = matching[0];
    let path = root.artifact(&invocation["artifact_ref"])?;
    let filename = path
        .file_name()
        .and_then(|v| v.to_str())
        .ok_or("native entrypoint filename required")?;
    let base = filename.strip_suffix(".exe").unwrap_or(filename);
    if binding["entrypoint"] != base || binding["permission_grant"] != false {
        return Err("artifact does not match declared native entrypoint".into());
    }
    let transport = root.bound(&binding["transport_ref"])?;
    if transport["request_encoding"] != "ONE_UTF8_JSON_OBJECT_ON_STDIN_EOF"
        || transport["response_encoding"] != "ONE_UTF8_JSON_OBJECT_ON_STDOUT"
    {
        return Err("unsupported registered transport".into());
    }
    jsonschema::validate(&transport["request_schema"], &request).map_err(|e| e.to_string())?;
    for reference in manifest["extensions"]
        .as_array()
        .ok_or("manifest extensions required")?
    {
        let extension = root.bound(reference)?;
        if extension["engine_id"] != manifest["engine_id"] {
            return Err("runtime extension belongs to another engine".into());
        }
    }
    let operations: Vec<_> = contract["operations"]
        .as_array()
        .ok_or("capability operations required")?
        .iter()
        .filter(|o| o["operation_id"] == invocation["operation_id"])
        .collect();
    if operations.len() != 1 {
        return Err("registered capability operation missing or ambiguous".into());
    }
    inline(&operations[0]["typed_input"])?;
    jsonschema::validate(&operations[0]["typed_input"], &request["input"])
        .map_err(|e| format!("native request input: {e}"))?;
    Ok((invocation, request, path))
}
