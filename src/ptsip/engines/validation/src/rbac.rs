// Compile-Sealed RBAC profile preflight, owned by the distributable Validation Engine.
// This module does not grant authority, validate binding legitimacy, or activate permissions.
// Registered engine operation/version cutover is separate from this internal implementation.
use serde_json::{json, Value};

use crate::source::Root;

const SCHEMA_ROOT: &str = ".ptsip/contracts/schemas/compile-sealed-rbac/";
const PROFILE_RBAC_ROOT: &str = ".ptsip/profiles/rbac/";
const SCHEMAS: [&str; 6] = [
    "responsibility.schema.json",
    "condition-atom.schema.json",
    "permission-atom.schema.json",
    "binding.schema.json",
    "sealed-contract.schema.json",
    "activation.schema.json",
];

fn ensure_local_schema_refs(value: &Value) -> Result<(), String> {
    match value {
        Value::Object(items) => {
            for (key, child) in items {
                if key == "$ref" || key == "$dynamicRef" {
                    if key == "$dynamicRef"
                        || !child.as_str().is_some_and(|s| s.starts_with("#/"))
                    {
                        return Err("RBAC schema uses an unregistered reference".into());
                    }
                } else {
                    ensure_local_schema_refs(child)?;
                }
            }
        }
        Value::Array(items) => {
            for child in items {
                ensure_local_schema_refs(child)?;
            }
        }
        _ => {}
    }
    Ok(())
}

/// Validate the selected, already-decoded pp.2.00 profile declaration.
/// JSON is the native engine transport. YAML decoding is a separate input-adapter concern.
/// This is a structural preflight only: a completed Registry/Binding/Seal proof is NOT asserted.
pub fn preflight(input: &Value) -> Result<Value, String> {
    let profile = input
        .get("profile")
        .and_then(Value::as_object)
        .ok_or("profile must be an explicitly supplied object")?;
    let identity = profile
        .get("ptsip")
        .and_then(Value::as_object)
        .ok_or("ptsip identity is required")?;
    if identity.get("version").and_then(Value::as_str) != Some("pp.2.00") {
        return Err("RBAC declaration requires pp.2.00; legacy profile semantics unchanged".into());
    }
    let rbac = profile
        .get("rbac")
        .and_then(Value::as_object)
        .ok_or("pp.2.00 requires explicit rbac declaration")?;
    let enabled = rbac
        .get("enabled")
        .and_then(Value::as_bool)
        .ok_or("rbac.enabled must be an explicit boolean")?;
    let registry_ref = rbac.get("registry_ref");
    if !enabled {
        // Optional means optional, not a trigger inferred from presence or file existence.
        if registry_ref.is_some_and(|v| !v.is_string()) {
            return Err("rbac.registry_ref must be a string when present".into());
        }
        return Ok(json!({
            "declaration_valid": true,
            "rbac_enabled": false,
            "rbac_usable": false,
            "verified_schema_count": 0,
            "validation_stage": "DECLARATION_ONLY",
            "diagnostics": [],
            "ptsip_conformance_verdict": "NOT_ASSERTED",
            "execution_authorized": false
        }));
    }
    let reference = registry_ref
        .and_then(Value::as_str)
        .filter(|s| !s.is_empty())
        .ok_or("rbac.registry_ref is required when rbac.enabled is true")?;
    if !reference.starts_with(PROFILE_RBAC_ROOT)
        || reference.len() <= PROFILE_RBAC_ROOT.len()
    {
        return Err("RBAC registry must be located under .ptsip/profiles/rbac/".into());
    }
    let root = Root::new(
        input
            .get("root")
            .and_then(Value::as_str)
            .filter(|s| !s.is_empty())
            .ok_or("repository root is required for enabled RBAC")?,
    )?;
    // Root::bytes rejects absolute/parent traversal and symlink escapes.
    let registry_bytes = root.bytes(reference, 2_097_152)?;
    if registry_bytes.is_empty() {
        return Err("RBAC registry cannot be empty".into());
    }
    for name in SCHEMAS {
        let path = format!("{SCHEMA_ROOT}{name}");
        let schema_bytes = root.bytes(&path, 2_097_152)?;
        let schema = crate::source::parse(&schema_bytes)?;
        if !schema.is_object()
            || schema.get("$schema").and_then(Value::as_str)
                != Some("https://json-schema.org/draft/2020-12/schema")
        {
            return Err(format!("{path}: expected JSON Schema draft 2020-12"));
        }
        ensure_local_schema_refs(&schema)?;
        jsonschema::validator_for(&schema)
            .map_err(|e| format!("{path}: invalid JSON Schema: {e}"))?;
    }
    // The registry record grammar and six domain schemas have not yet been approved.
    // Existence and syntactic validation MUST NOT be mistaken for RBAC readiness.
    Ok(json!({
        "declaration_valid": true,
        "rbac_enabled": true,
        "rbac_usable": false,
        "verified_schema_count": 6,
        "validation_stage": "STRUCTURAL_PREFLIGHT_ONLY",
        "diagnostics": ["RBAC_REGISTRY_BINDING_SEAL_VALIDATION_NOT_YET_DEFINED"],
        "ptsip_conformance_verdict": "NOT_ASSERTED",
        "execution_authorized": false
    }))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn explicit_disabled_declaration_needs_no_registry_or_schemas() {
        let input = json!({"profile": {"ptsip":{"version":"pp.2.00"}, "rbac":{"enabled":false}}});
        let result = preflight(&input).unwrap();
        assert_eq!(result["declaration_valid"], true);
        assert_eq!(result["rbac_enabled"], false);
        assert_eq!(result["rbac_usable"], false);
        assert_eq!(result["execution_authorized"], false);
        let ignored = json!({"profile": {"ptsip":{"version":"pp.2.00"}, "rbac":{"enabled":false, "registry_ref":"ignored"}}});
        assert_eq!(preflight(&ignored).unwrap()["rbac_enabled"], false);
    }

    #[test]
    fn false_or_missing_enabled_does_not_enable_rbac() {
        for value in [
            json!({"profile":{"ptsip":{"version":"pp.2.00"},"rbac":{}}}),
            json!({"profile":{"ptsip":{"version":"pp.2.00"},"rbac":{"enabled":"true"}}}),
            json!({"profile":{"ptsip":{"version":"pp.1.02"},"rbac":{"enabled":false}}}),
            json!({"profile":{"ptsip":{"version":"pp.2.00"}}}),
        ] {
            assert!(preflight(&value).is_err());
        }
    }

    #[test]
    fn enabled_requires_registered_root_and_six_schemas() {
        let base = json!({"profile":{"ptsip":{"version":"pp.2.00"},"rbac":{"enabled":true}}});
        assert!(preflight(&base).is_err());
        let escape = json!({"root":"/not-a-root", "profile":{"ptsip":{"version":"pp.2.00"},"rbac":{"enabled":true,"registry_ref":"../escape.json"}}});
        assert!(preflight(&escape).is_err());
    }

    #[test]
    fn enabled_with_six_syntactically_valid_schemas_remains_non_authoritative() {
        use std::fs;
        use std::time::{SystemTime, UNIX_EPOCH};
        let stamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let root = std::env::temp_dir().join(format!(
            "ptsip-rbac-preflight-test-{}-{stamp}",
            std::process::id()
        ));
        let schemas = root.join(SCHEMA_ROOT);
        let registry = root.join(PROFILE_RBAC_ROOT);
        fs::create_dir_all(&schemas).unwrap();
        fs::create_dir_all(&registry).unwrap();
        fs::write(registry.join("index.json"), "{}").unwrap();
        for name in SCHEMAS {
            fs::write(
                schemas.join(name),
                r#"{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}"#,
            )
            .unwrap();
        }
        let request = json!({
            "root": root.to_str().unwrap(),
            "profile": {
                "ptsip": {"version": "pp.2.00"},
                "rbac": {"enabled": true, "registry_ref": ".ptsip/profiles/rbac/index.json"}
            }
        });
        let result = preflight(&request).unwrap();
        assert_eq!(result["verified_schema_count"], 6);
        assert_eq!(result["declaration_valid"], true);
        assert_eq!(result["rbac_usable"], false);
        assert_eq!(result["validation_stage"], "STRUCTURAL_PREFLIGHT_ONLY");
        assert_eq!(result["execution_authorized"], false);
        fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn external_schema_refs_fail_closed() {
        assert!(ensure_local_schema_refs(&json!({"$ref":"https://example.invalid/schema"})).is_err());
        assert!(ensure_local_schema_refs(&json!({"$dynamicRef":"#/$defs/item"})).is_err());
        assert!(ensure_local_schema_refs(&json!({"$ref":"#/$defs/item"})).is_ok());
    }
}
