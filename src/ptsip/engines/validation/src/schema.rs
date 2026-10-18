//! Operation-local schema resources. URI identifiers never grant network or file access.
use jsonschema::{Retrieve, Uri, Validator};
use serde_json::Value;
use std::collections::BTreeMap;
use std::sync::Arc;

const INTERNAL: &str = "INTERNAL_REF_UNSUPPORTED: internal $ref (#, #/... or the current resource ID) is not supported in PTSIP 0.3.8b4";
const DYNAMIC: &str = "DYNAMIC_REF_UNSUPPORTED: $dynamicRef is not supported in PTSIP 0.3.8b4";

#[derive(Clone)]
struct Resources(Arc<BTreeMap<String, Value>>);
impl Retrieve for Resources {
    fn retrieve(
        &self,
        uri: &Uri<String>,
    ) -> Result<Value, Box<dyn std::error::Error + Send + Sync>> {
        self.0.get(uri.as_str()).cloned().ok_or_else(|| {
            format!("SCHEMA_RESOURCE_NOT_FOUND: {} is absent from schema_resources; network and filesystem retrieval are disabled", uri.as_str()).into()
        })
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

fn inspect(
    schema: &Value,
    inherited: Option<&str>,
    resources: &mut BTreeMap<String, Value>,
    references: &mut Vec<String>,
    remaining: &mut usize,
) -> Result<(), String> {
    *remaining = remaining
        .checked_sub(1)
        .ok_or("SCHEMA_RESOURCE_LIMIT: too many schema nodes")?;
    let Some(object) = schema.as_object() else {
        return Ok(());
    };
    let mut base = inherited.map(str::to_owned);
    if let Some(id) = object.get("$id") {
        let id = id
            .as_str()
            .ok_or("INVALID_SCHEMA_ID: $id must be a string")?;
        if id.is_empty() || id.contains('#') {
            return Err(
                "INVALID_SCHEMA_ID: $id must identify a resource without a fragment".into(),
            );
        }
        let id = resolve(id, inherited)?;
        if resources.insert(id.clone(), schema.clone()).is_some() {
            return Err(format!("DUPLICATE_SCHEMA_ID: {id}"));
        }
        base = Some(id);
    }
    if object.contains_key("$dynamicRef") {
        return Err(DYNAMIC.into());
    }
    if let Some(reference) = object.get("$ref") {
        let reference = reference
            .as_str()
            .ok_or("INVALID_SCHEMA_REF: $ref must be a string")?;
        if reference.is_empty() || reference.starts_with('#') {
            return Err(INTERNAL.into());
        }
        let target = resolve(reference, base.as_deref())?;
        let document = target.split('#').next().unwrap_or("");
        if Some(document) == base.as_deref() {
            return Err(INTERNAL.into());
        }
        references.push(document.to_owned());
    }
    // Visit schema locations only: values in examples/default/const are data.
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
                inspect(child, base.as_deref(), resources, references, remaining)?;
            }
        }
    }
    for name in ["allOf", "anyOf", "oneOf", "prefixItems"] {
        if let Some(children) = object.get(name).and_then(Value::as_array) {
            for child in children {
                inspect(child, base.as_deref(), resources, references, remaining)?;
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
                    inspect(child, base.as_deref(), resources, references, remaining)?;
                }
            } else {
                inspect(child, base.as_deref(), resources, references, remaining)?;
            }
        }
    }
    Ok(())
}

pub fn validator(schema: &Value, supplied: Option<&Value>) -> Result<Validator, String> {
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
        None,
        &mut resources,
        &mut references,
        &mut remaining,
    )?;
    for resource in supplied {
        let id = resource.get("$id").and_then(Value::as_str).ok_or(
            "SCHEMA_RESOURCE_ID_REQUIRED: every schema_resources entry requires an absolute $id",
        )?;
        resolve(id, None)?;
        inspect(
            resource,
            None,
            &mut resources,
            &mut references,
            &mut remaining,
        )?;
    }
    for reference in references {
        if !resources.contains_key(&reference) {
            return Err(format!("SCHEMA_RESOURCE_NOT_FOUND: {reference} is absent from schema_resources; network and filesystem retrieval are disabled"));
        }
    }
    let retriever = Resources(Arc::new(resources));
    // Validate the complete supplied set, including unused resources, before use.
    for resource in supplied {
        jsonschema::options()
            .with_retriever(retriever.clone())
            .build(resource)
            .map_err(|e| format!("INVALID_SCHEMA_RESOURCE: {e}"))?;
    }
    jsonschema::options()
        .with_retriever(retriever)
        .build(schema)
        .map_err(|e| format!("INVALID_SCHEMA: {e}"))
}

pub fn validate(
    schema: &Value,
    instance: &Value,
    resources: Option<&Value>,
) -> Result<Vec<String>, String> {
    Ok(validator(schema, resources)?
        .iter_errors(instance)
        .map(|e| e.to_string())
        .collect())
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn external_ids_resolve_without_manual_uri_mapping() {
        let schema = json!({"$ref":"https://example.invalid/address"});
        let resources = json!([{"$id":"https://example.invalid/address","type":"object","properties":{"zip":{"type":"integer"}},"required":["zip"]}]);
        assert!(validate(&schema, &json!({"zip":123}), Some(&resources))
            .unwrap()
            .is_empty());
        assert!(
            !validate(&schema, &json!({"zip":"wrong"}), Some(&resources))
                .unwrap()
                .is_empty()
        );
    }

    #[test]
    fn relative_transitive_external_refs_and_external_fragments_resolve() {
        let root = json!({"$id":"https://example.invalid/main","$ref":"parts/customer"});
        let resources = json!([
            {"$id":"https://example.invalid/parts/customer","$ref":"types#/$defs/name"},
            {"$id":"https://example.invalid/parts/types","$defs":{"name":{"type":"string","minLength":2}}}
        ]);
        assert!(validate(&root, &json!("ok"), Some(&resources))
            .unwrap()
            .is_empty());
        assert!(!validate(&root, &json!("x"), Some(&resources))
            .unwrap()
            .is_empty());
    }

    #[test]
    fn embedded_schema_ids_and_anchors_are_discovered() {
        let root = json!({"$id":"https://example.invalid/main","$ref":"other#name","$defs":{"embedded":{"$id":"other","$anchor":"name","type":"string"}}});
        assert!(validate(&root, &json!("ok"), None).unwrap().is_empty());
        assert!(!validate(&root, &json!(1), None).unwrap().is_empty());
    }

    #[test]
    fn duplicate_missing_and_invalid_resource_ids_fail_closed() {
        for resources in [
            json!([{"type":"string"}]),
            json!([{"$id":"relative","type":"string"}]),
            json!([{"$id":"https://example.invalid/a"},{"$id":"https://example.invalid/a"}]),
            json!([{"$id":"https://example.invalid/a#fragment"}]),
            json!([{"$id":"https://example.invalid/a","type":123}]),
        ] {
            assert!(validator(&json!({}), Some(&resources)).is_err());
        }
        for uri in [
            "https://example.invalid/unregistered",
            "file:///unregistered.json",
            "urn:missing",
        ] {
            let error = validator(&json!({"$ref":uri}), None).err().unwrap();
            assert!(error.contains("SCHEMA_RESOURCE_NOT_FOUND"), "{error}");
        }
    }

    #[test]
    fn internal_and_dynamic_refs_report_explicit_limits_everywhere() {
        for schema in [
            json!({"$ref":"#"}),
            json!({"$ref":"#/$defs/x","$defs":{"x":{}}}),
            json!({"$id":"https://example.invalid/a","$ref":"https://example.invalid/a"}),
            json!({"properties":{"x":{"$ref":"#"}}}),
            json!({"$schema":"http://json-schema.org/draft-07/schema#","items":[{"$ref":"#"}]}),
            json!({"$dynamicRef":"https://example.invalid/a"}),
            json!({"$defs":{"x":{"$dynamicRef":"#x"}}}),
        ] {
            let error = validator(&schema, None).err().unwrap();
            assert!(error.contains("REF_UNSUPPORTED"), "{error}");
        }
        let resources = json!([{"$id":"urn:bad","$ref":"#"}]);
        assert!(validator(&json!({}), Some(&resources))
            .err()
            .unwrap()
            .contains("INTERNAL_REF_UNSUPPORTED"));
    }

    #[test]
    fn annotations_are_data_and_resources_do_not_leak_between_requests() {
        let literal = json!({"$ref":"#","$dynamicRef":"#x"});
        let schema = json!({"const":literal,"examples":[literal]});
        assert!(validate(&schema, &literal, None).unwrap().is_empty());
        let schema = json!({"$ref":"urn:only-one-request"});
        let resources = json!([{"$id":"urn:only-one-request","type":"integer"}]);
        assert!(validate(&schema, &json!(3), Some(&resources)).is_ok());
        assert!(validate(&schema, &json!(3), None).is_err());
    }
}
