//! Private adapter for the distributed, dedicated user-contract validator.

use crate::source;
use serde_json::{json, Value};
use std::ffi::OsStr;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};

pub(crate) fn algorithm(operation: &str) -> Option<&'static str> {
    match operation {
        "validation.policy.identity" => {
            Some("VALIDATE_POLICY_V4_IDENTITY_WITH_USER_CATEGORY_AND_OPTIONAL_VERSION")
        }
        "validation.policy.rules" => {
            Some("VALIDATE_POLICY_V4_RULES_WITH_USER_CONTRACTS_AND_LOCAL_CONTENT_REFERENCES")
        }
        "validation.rbac.ra" => {
            Some("VALIDATE_RA_SEVEN_FIELD_STRUCTURE_AND_FORWARD_RELATION_SHAPE")
        }
        "validation.rbac.rac" => Some("VALIDATE_RAC_LIST_STRUCTURE_AND_APPROVED_FIELDS"),
        "validation.rbac.aec" => Some("VALIDATE_AEC_LIST_STRUCTURE_AND_APPROVED_FIELDS"),
        "validation.rbac.references" => {
            Some("VALIDATE_POLICY_RA_RAC_AEC_FORWARD_REFERENCES_AND_CANONICAL_NAMES")
        }
        _ => None,
    }
}

fn worker_binding(
    executable: Option<&OsStr>,
    root: Option<&OsStr>,
) -> Result<(PathBuf, PathBuf), String> {
    let executable = executable
        .map(Path::new)
        .ok_or("USER_CONTRACT_ADAPTER_UNAVAILABLE: PTSIP_GO_CLI binding is required")?;
    if !executable.is_absolute() || !executable.is_file() {
        return Err(
            "USER_CONTRACT_ADAPTER_UNAVAILABLE: PTSIP_GO_CLI must be an absolute regular file"
                .into(),
        );
    }
    let root = root
        .map(Path::new)
        .ok_or("USER_CONTRACT_ADAPTER_UNAVAILABLE: PTSIP_USER_CONTRACT_ROOT binding is required")?;
    if !root.is_absolute() || !root.is_dir() {
        return Err(
            "USER_CONTRACT_ADAPTER_UNAVAILABLE: PTSIP_USER_CONTRACT_ROOT must be an absolute directory"
                .into(),
        );
    }
    Ok((executable.to_path_buf(), root.to_path_buf()))
}

pub(crate) fn evaluate(operation: &str, input: &Value) -> Result<Value, String> {
    if algorithm(operation).is_none() {
        return Err("unsupported user-contract adapter operation".into());
    }
    let executable = std::env::var_os("PTSIP_GO_CLI");
    let root = std::env::var_os("PTSIP_USER_CONTRACT_ROOT");
    let (executable, root) = worker_binding(executable.as_deref(), root.as_deref())?;
    let request = serde_json::to_vec(&json!({"operation_id": operation, "input": input}))
        .map_err(|error| format!("USER_CONTRACT_ADAPTER_REQUEST: {error}"))?;
    let mut child = Command::new(executable)
        .arg("--user-contracts-worker")
        .arg(root)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|error| format!("USER_CONTRACT_ADAPTER_UNAVAILABLE: {error}"))?;
    let sent = child
        .stdin
        .take()
        .ok_or_else(|| std::io::Error::other("adapter stdin unavailable"))
        .and_then(|mut stdin| stdin.write_all(&request));
    if let Err(error) = sent {
        let _ = child.kill();
        let _ = child.wait();
        return Err(format!("USER_CONTRACT_ADAPTER_REQUEST: {error}"));
    }
    let response = child
        .wait_with_output()
        .map_err(|error| format!("USER_CONTRACT_ADAPTER_FAILED: {error}"))?;
    if !response.status.success() {
        let detail = String::from_utf8_lossy(&response.stderr);
        return Err(format!(
            "USER_CONTRACT_ADAPTER_FAILED: {}",
            detail.chars().take(2048).collect::<String>().trim()
        ));
    }
    source::parse(&response.stdout)
        .map_err(|error| format!("USER_CONTRACT_ADAPTER_INVALID_OUTPUT: {error}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn explicit_worker_bindings_require_absolute_file_and_directory() {
        let root = std::env::temp_dir().join(format!(
            "ptsip-user-contract-worker-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir(&root).unwrap();
        let executable = root.join("ptsip-go-cli");
        std::fs::write(&executable, b"worker fixture").unwrap();
        assert!(worker_binding(None, Some(root.as_os_str())).is_err());
        assert!(worker_binding(Some(executable.as_os_str()), None).is_err());
        assert!(worker_binding(Some(OsStr::new("ptsip-go-cli")), Some(root.as_os_str())).is_err());
        assert!(worker_binding(Some(executable.as_os_str()), Some(OsStr::new("."))).is_err());
        assert!(worker_binding(Some(root.as_os_str()), Some(root.as_os_str())).is_err());
        assert!(
            worker_binding(Some(executable.as_os_str()), Some(executable.as_os_str())).is_err()
        );
        assert_eq!(
            worker_binding(Some(executable.as_os_str()), Some(root.as_os_str())).unwrap(),
            (executable.clone(), root.clone())
        );
        std::fs::remove_file(executable).unwrap();
        std::fs::remove_dir(root).unwrap();
    }

    #[test]
    fn exactly_six_operations_have_private_adapter_mappings() {
        let contract: Value = serde_json::from_str(crate::CONTRACT).unwrap();
        let operations: Vec<_> = contract["operations"]
            .as_array()
            .unwrap()
            .iter()
            .filter(|definition| algorithm(definition["operation_id"].as_str().unwrap()).is_some())
            .collect();
        assert_eq!(operations.len(), 6);
        for definition in operations {
            assert_eq!(
                definition["machine_semantics"]["algorithm"],
                algorithm(definition["operation_id"].as_str().unwrap()).unwrap()
            );
            assert!(jsonschema::is_valid(
                &definition["typed_output"],
                &json!({
                    "valid": true, "diagnostics": [], "user_contracts": [], "ra_usage": [],
                    "execution_authorized": true, "ptsip_conformance_verdict": "NOT_ASSERTED"
                })
            ));
            assert!(!jsonschema::is_valid(
                &definition["typed_input"],
                &json!({})
            ));
        }
        assert_eq!(algorithm("validation.policy.lifecycle"), None);
        assert_eq!(algorithm("validation.rbac.execute"), None);
    }

    #[test]
    fn private_adapter_contract_rejects_malformed_output_and_input() {
        let contract: Value = serde_json::from_str(crate::CONTRACT).unwrap();
        for definition in contract["operations"].as_array().unwrap() {
            let operation = definition["operation_id"].as_str().unwrap();
            if algorithm(operation).is_none() {
                continue;
            }
            let input = match operation {
                "validation.policy.identity" => json!({"policy": {"Policy_Identity": {}}}),
                "validation.policy.rules" => json!({"policy": {"Policy_Rules": {}}}),
                "validation.rbac.references" => json!({"policy": {}}),
                _ => json!({"document": {}}),
            };
            assert!(jsonschema::is_valid(&definition["typed_input"], &input));
            let mut extra_input = input;
            extra_input["grant_execution"] = json!(true);
            assert!(!jsonschema::is_valid(
                &definition["typed_input"],
                &extra_input
            ));
            let mut output = json!({
                "valid": false,
                "diagnostics": [{"code": "SCHEMA", "path": ["section", 0], "message": "invalid"}],
                "user_contracts": [],
                "ra_usage": [],
                "execution_authorized": true,
                "ptsip_conformance_verdict": "NOT_ASSERTED"
            });
            assert!(jsonschema::is_valid(&definition["typed_output"], &output));
            output["execution_authorized"] = json!(false);
            assert!(!jsonschema::is_valid(&definition["typed_output"], &output));
            output["execution_authorized"] = json!(true);
            output["diagnostics"][0]["path"] = json!([false]);
            assert!(!jsonschema::is_valid(&definition["typed_output"], &output));
        }
    }
}
