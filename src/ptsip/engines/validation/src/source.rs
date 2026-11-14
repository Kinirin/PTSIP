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

pub mod planning {
    use super::{schema, text, Root};
    use serde_json::{json, Value};
    use std::collections::{BTreeMap, BTreeSet};

    pub fn impact(input: &Value) -> Result<Value, String> {
        let mut root = Root::new(text(input, "root")?)?;
        let (graph, digest) = root.read(text(input, "graph_ref")?)?;
        if input["expected_graph_sha256"] != digest {
            return Err("stale dependency graph".into());
        }
        schema("planning_impact_graph", &graph)?;
        let mut nodes = BTreeMap::new();
        for node in graph["nodes"].as_array().ok_or("nodes required")? {
            let id = text(node, "id")?;
            if nodes.insert(id, node).is_some() {
                return Err("duplicate dependency node identity".into());
            }
            root.bound(node)?;
        }
        let changed = text(input, "changed_id")?;
        if !nodes.contains_key(changed) {
            return Err("changed node is not in graph".into());
        }
        let mut pairs = BTreeSet::new();
        let mut adjacency = BTreeMap::<&str, Vec<(&str, &Value)>>::new();
        for edge in graph["edges"].as_array().ok_or("edges required")? {
            let provider = text(edge, "provider")?;
            let consumer = text(edge, "consumer")?;
            if !nodes.contains_key(provider) || !nodes.contains_key(consumer) {
                return Err("unknown dependency endpoint".into());
            }
            if !pairs.insert((provider, consumer, text(edge, "relation")?)) {
                return Err("ambiguous dependency edge".into());
            }
            adjacency
                .entry(provider)
                .or_default()
                .push((consumer, edge));
        }

        let mut affected = BTreeSet::from([changed]);
        let mut pending = vec![changed];
        let mut verified = true;
        while let Some(provider) = pending.pop() {
            for &(consumer, edge) in adjacency.get(provider).into_iter().flatten() {
                let provider_status = evidence(
                    &mut root,
                    &edge["provider_evidence"],
                    provider,
                    edge,
                    nodes[provider],
                    nodes[consumer],
                );
                let consumer_status = evidence(
                    &mut root,
                    &edge["consumer_evidence"],
                    consumer,
                    edge,
                    nodes[provider],
                    nodes[consumer],
                );
                match (provider_status.as_deref(), consumer_status.as_deref()) {
                    (Some("COMPATIBLE"), Some("COMPATIBLE")) => continue,
                    (Some("AFFECTED"), Some("AFFECTED")) => {}
                    _ => verified = false,
                }
                if affected.insert(consumer) {
                    pending.push(consumer);
                }
            }
        }
        root.finish()?;
        let status = if !verified {
            "UNCERTAIN_UNVERIFIED"
        } else if affected.len() > 1 {
            "VERIFIED_AFFECTED"
        } else {
            "VERIFIED_COMPATIBLE_UNAFFECTED"
        };
        Ok(
            json!({"changed_id": changed, "affected_ids": affected, "status": status,
                  "verification_eligible": verified, "execution_authorized": false}),
        )
    }

    fn evidence(
        root: &mut Root,
        reference: &Value,
        producer: &str,
        edge: &Value,
        provider: &Value,
        consumer: &Value,
    ) -> Option<String> {
        let record = root.bound(reference).ok()?;
        if record["provider"] != edge["provider"]
            || record["consumer"] != edge["consumer"]
            || record["relation"] != edge["relation"]
            || record["producer_id"] != producer
            || record["provider_sha256"] != provider["sha256"]
            || record["consumer_sha256"] != consumer["sha256"]
            || record["outcome"] != "PASS"
        {
            return None;
        }
        record["compatibility"].as_str().map(str::to_owned)
    }

    pub fn prepare(input: &Value) -> Result<Value, String> {
        let mut root = Root::new(text(input, "root")?)?;
        let plan = root.bound(&input["plan_ref"])?;
        schema("planning_preparation", &plan)?;

        let mut steps = BTreeMap::new();
        let mut required = BTreeSet::new();
        for step in plan["steps"].as_array().ok_or("steps required")? {
            let id = text(step, "id")?;
            if steps.insert(id, step).is_some() {
                return Err("duplicate plan step identity".into());
            }
            required.insert(text(step, "invocation_id")?);
        }
        let mut pending = BTreeMap::new();
        let mut dependents = BTreeMap::<&str, Vec<&str>>::new();
        for (&id, step) in &steps {
            let dependencies = step["dependencies"]
                .as_array()
                .ok_or("dependencies required")?;
            pending.insert(id, dependencies.len());
            for dependency in dependencies {
                let dependency = dependency.as_str().ok_or("dependency ID required")?;
                if !steps.contains_key(dependency) {
                    return Err("unknown plan step dependency".into());
                }
                dependents.entry(dependency).or_default().push(id);
            }
        }
        let mut ready: Vec<_> = pending
            .iter()
            .filter_map(|(&id, &count)| (count == 0).then_some(id))
            .collect();
        let mut visited = 0;
        while let Some(id) = ready.pop() {
            visited += 1;
            for &dependent in dependents.get(id).into_iter().flatten() {
                let count = pending.get_mut(dependent).ok_or("unknown plan step")?;
                *count -= 1;
                if *count == 0 {
                    ready.push(dependent);
                }
            }
        }
        if visited != steps.len() {
            return Err("plan step dependency cycle".into());
        }

        let mut declared = BTreeSet::new();
        for reference in input["invocation_refs"]
            .as_array()
            .ok_or("invocation references required")?
        {
            let id = text(reference, "id")?;
            if !required.contains(id) || !declared.insert(id) {
                return Err("unused or ambiguous invocation reference".into());
            }
            let invocation = root.bound(reference)?;
            schema("planning_invocation", &invocation)?;
        }
        if declared != required {
            return Err("missing invocation reference".into());
        }
        root.finish()?;
        Ok(
            json!({"plan_ref": input["plan_ref"], "step_ids": steps.keys().collect::<Vec<_>>(),
                  "invocation_ids": declared, "prepared": true, "execution_authorized": false}),
        )
    }
}
