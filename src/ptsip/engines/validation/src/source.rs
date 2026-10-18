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
    crate::schema::validator(schema, None)?;
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
    pub fn finish(&self) -> Result<(), String> {
        for (name, digest) in &self.sources {
            if hash(&self.bytes(name, 268_435_456)?) != *digest {
                return Err(format!("source changed: {name}"));
            }
        }
        Ok(())
    }
}
