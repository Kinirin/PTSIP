//! Reference evaluation is owned by the JSON Schema evaluator; acquisition is explicit.
use crate::retrieval::{Document, Loader, Resolution};
use jsonschema::{Retrieve, Uri, Validator};
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::sync::Arc;

#[derive(Clone)]
struct Resources(Arc<BTreeMap<String, Value>>);
impl Retrieve for Resources {
    fn retrieve(
        &self,
        uri: &Uri<String>,
    ) -> Result<Value, Box<dyn std::error::Error + Send + Sync>> {
        self.0
            .get(uri.as_str())
            .cloned()
            .ok_or_else(|| format!("SCHEMA_RESOURCE_NOT_FOUND: {}", uri.as_str()).into())
    }
}
fn resolve(reference: &str, base: Option<&str>) -> Result<String, String> {
    let uri = if let Some(base) = base {
        let base = Uri::parse(base).map_err(|e| format!("INVALID_SCHEMA_ID: {e}"))?;
        jsonschema::uri::resolve_against(&base, reference).map_err(|e| e.to_string())?
    } else {
        Uri::parse(reference.to_owned())
            .map_err(|(e, _)| format!("ABSOLUTE_SCHEMA_URI_REQUIRED: {reference}: {e}"))?
            .normalize()
    };
    Ok(uri.as_str().to_owned())
}
fn insert(
    resources: &mut BTreeMap<String, Value>,
    id: String,
    schema: &Value,
    identical_ok: bool,
) -> Result<(), String> {
    if let Some(previous) = resources.get(&id) {
        if identical_ok && previous == schema {
            return Ok(());
        }
        return Err(format!("DUPLICATE_SCHEMA_ID: {id}"));
    }
    resources.insert(id, schema.clone());
    Ok(())
}
fn inspect(
    schema: &Value,
    inherited: &str,
    resources: &mut BTreeMap<String, Value>,
    references: &mut Vec<String>,
    remaining: &mut usize,
    identical_ok: bool,
) -> Result<(), String> {
    *remaining = remaining
        .checked_sub(1)
        .ok_or("SCHEMA_RESOURCE_LIMIT: too many schema nodes")?;
    if schema.is_boolean() {
        return Ok(());
    }
    let object = schema
        .as_object()
        .ok_or("INVALID_SCHEMA: object or boolean schema required")?;
    let mut base = inherited.to_owned();
    if let Some(id) = object.get("$id") {
        let id = id
            .as_str()
            .ok_or("INVALID_SCHEMA_ID: $id must be a string")?;
        if id.is_empty() || id.contains('#') {
            return Err("INVALID_SCHEMA_ID: fragment-free resource ID required".into());
        }
        base = resolve(id, Some(inherited))?;
        insert(resources, base.clone(), schema, identical_ok)?;
    }
    for keyword in ["$ref", "$dynamicRef"] {
        if let Some(reference) = object.get(keyword) {
            let reference = reference
                .as_str()
                .ok_or("INVALID_SCHEMA_REF: URI-reference string required")?;
            let target = resolve(reference, Some(&base))?;
            references.push(target.split('#').next().unwrap_or("").to_owned());
        }
    }
    for name in [
        "$defs",
        "definitions",
        "properties",
        "patternProperties",
        "dependentSchemas",
        "dependencies",
    ] {
        if let Some(children) = object.get(name).and_then(Value::as_object) {
            for child in children.values() {
                if name == "dependencies" && child.is_array() {
                    continue;
                }
                inspect(child, &base, resources, references, remaining, identical_ok)?;
            }
        }
    }
    for name in ["allOf", "anyOf", "oneOf", "prefixItems"] {
        if let Some(children) = object.get(name).and_then(Value::as_array) {
            for child in children {
                inspect(child, &base, resources, references, remaining, identical_ok)?;
            }
        }
    }
    for name in [
        "items",
        "additionalItems",
        "contains",
        "additionalProperties",
        "unevaluatedProperties",
        "unevaluatedItems",
        "propertyNames",
        "not",
        "if",
        "then",
        "else",
        "contentSchema",
    ] {
        if let Some(child) = object.get(name) {
            if let Some(children) = child.as_array() {
                for child in children {
                    inspect(child, &base, resources, references, remaining, identical_ok)?;
                }
            } else {
                inspect(child, &base, resources, references, remaining, identical_ok)?;
            }
        }
    }
    Ok(())
}
pub struct Prepared {
    pub validator: Validator,
    pub evidence: Vec<Value>,
}
pub fn prepare(
    schema: &Value,
    supplied: Option<&Value>,
    resolution: Option<&Value>,
) -> Result<Prepared, String> {
    let config: Resolution = match resolution {
        None => Resolution::default(),
        Some(value) => serde_json::from_value(value.clone())
            .map_err(|e| format!("SCHEMA_RESOLUTION_INVALID: {e}"))?,
    };
    let base = resolve(
        config
            .base_uri
            .as_deref()
            .unwrap_or("urn:ptsip:validation:root"),
        None,
    )?;
    if base.contains('#') {
        return Err("INVALID_SCHEMA_ID: base_uri must not contain a fragment".into());
    }
    let root_uri = match schema.get("$id").and_then(Value::as_str) {
        Some(id) => resolve(id, Some(&base))?,
        None => base.clone(),
    };
    let supplied =
        match supplied {
            None => &[][..],
            Some(Value::Array(values)) if values.len() <= 128 => values.as_slice(),
            _ => return Err(
                "SCHEMA_RESOURCE_LIMIT: schema_resources must be an array of at most 128 schemas"
                    .into(),
            ),
        };
    let mut resources = BTreeMap::new();
    let mut references = Vec::new();
    let mut remaining = 16384;
    inspect(
        schema,
        &base,
        &mut resources,
        &mut references,
        &mut remaining,
        false,
    )?;
    insert(&mut resources, root_uri.clone(), schema, true)?;
    let mut documents = vec![(root_uri.clone(), schema.clone())];
    let mut evidence = vec![
        json!({"uri":root_uri,"sha256":format!("{:x}",Sha256::digest(serde_json::to_vec(schema).map_err(|e| e.to_string())?)),"source":"INPUT_JSON_COMPACT"}),
    ];
    for resource in supplied {
        let id = resource
            .get("$id")
            .and_then(Value::as_str)
            .ok_or("SCHEMA_RESOURCE_ID_REQUIRED: explicit resource requires an absolute $id")?;
        let uri = resolve(id, None)?;
        inspect(
            resource,
            &uri,
            &mut resources,
            &mut references,
            &mut remaining,
            false,
        )?;
        documents.push((uri.clone(), resource.clone()));
        evidence.push(json!({"uri":uri,"sha256":format!("{:x}",Sha256::digest(serde_json::to_vec(resource).map_err(|e| e.to_string())?)),"source":"INPUT_JSON_COMPACT"}));
    }
    let mut loader = Loader::new(&config)?;
    let discovered = loader.discover()?;
    let add = |doc: Document,
               resources: &mut BTreeMap<String, Value>,
               references: &mut Vec<String>,
               remaining: &mut usize,
               documents: &mut Vec<(String, Value)>,
               evidence: &mut Vec<Value>|
     -> Result<(), String> {
        inspect(
            &doc.schema,
            &doc.uri,
            resources,
            references,
            remaining,
            true,
        )?;
        insert(resources, doc.uri.clone(), &doc.schema, true)?;
        documents.push((doc.uri, doc.schema));
        evidence.push(doc.evidence);
        Ok(())
    };
    for doc in discovered {
        add(
            doc,
            &mut resources,
            &mut references,
            &mut remaining,
            &mut documents,
            &mut evidence,
        )?;
    }
    let mut cursor = 0;
    while cursor < references.len() {
        if documents.len() > 128 {
            return Err("SCHEMA_RESOURCE_LIMIT: at most 128 schema documents".into());
        }
        let uri = references[cursor].clone();
        cursor += 1;
        if !resources.contains_key(&uri) && !referencing::SPECIFICATIONS.contains_resource(&uri) {
            let doc = loader.retrieve(&uri)?;
            insert(&mut resources, uri, &doc.schema, true)?;
            add(
                doc,
                &mut resources,
                &mut references,
                &mut remaining,
                &mut documents,
                &mut evidence,
            )?;
        }
    }
    if documents.len() > 128 {
        return Err("SCHEMA_RESOURCE_LIMIT: at most 128 schema documents".into());
    }
    let retriever = Resources(Arc::new(resources));
    for (uri, document) in documents.iter().skip(1) {
        jsonschema::options()
            .with_base_uri(uri)
            .with_retriever(retriever.clone())
            .build(document)
            .map_err(|e| format!("INVALID_SCHEMA_RESOURCE: {e}"))?;
    }
    let validator = jsonschema::options()
        .with_base_uri(base)
        .with_retriever(retriever)
        .build(schema)
        .map_err(|e| format!("INVALID_SCHEMA: {e}"))?;
    Ok(Prepared {
        validator,
        evidence,
    })
}
pub fn validator(schema: &Value, resources: Option<&Value>) -> Result<Validator, String> {
    Ok(prepare(schema, resources, None)?.validator)
}
pub fn validate_with_resolution(
    schema: &Value,
    instance: &Value,
    resources: Option<&Value>,
    resolution: Option<&Value>,
) -> Result<(Vec<String>, Vec<Value>), String> {
    let prepared = prepare(schema, resources, resolution)?;
    let errors = prepared
        .validator
        .iter_errors(instance)
        .map(|e| e.to_string())
        .collect();
    Ok((errors, prepared.evidence))
}
