use crate::source::{self, Root};
use serde_json::{json, Value};
use std::io::{Read, Write};
use std::process::{Command, Stdio};
use std::sync::mpsc;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

fn time_ms() -> Result<u64, String> {
    Ok(SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|e| e.to_string())?
        .as_millis() as u64)
}

pub fn invoke(input: &Value, manifest: &Value, contract: &Value) -> Result<Value, String> {
    crate::provider::mediate(input, manifest, contract)?;
    let mut root = Root::new(source::text(input, "root")?)?;
    let (registry, digest) = root.read(source::text(input, "provider_registry_ref")?)?;
    if input["expected_registry_sha256"] != digest {
        return Err("provider registry changed before invocation".into());
    }
    let registrations: Vec<_> = registry["providers"]
        .as_array()
        .ok_or("provider registrations required")?
        .iter()
        .filter(|p| {
            p["operation_id"] == input["operation_id"]
                && p["storage_capability_id"] == input["capability_id"]
                && p["storage_contract_version"] == input["contract_version"]
        })
        .collect();
    if registrations.len() != 1 {
        return Err("provider registration missing or ambiguous".into());
    }
    let registration = registrations[0];
    let provider = root.bound(&registration["source_ref"])?;
    let compatibility = root.bound(&registration["compatibility_ref"])?;
    for reference in compatibility["source_refs"]
        .as_array()
        .ok_or("current provider/consumer sources required")?
    {
        root.bound(reference)?;
    }
    let binding = &provider["native_binding"];
    source::schema("provider_native_binding", binding)?;
    if binding["transport"] != "PROVIDER_JSON_STDIO_V1"
        || binding["arguments"] != json!([])
        || binding["environment"] != json!({})
    {
        return Err("unsupported or unbounded native provider binding".into());
    }
    let artifact = root.artifact(&binding["artifact_ref"])?;
    let stored = root.bound(&input["request_ref"])?;
    if stored["operation_id"] != registration["provider_operation_id"] {
        return Err("provider request operation mismatch".into());
    }
    let operations: Vec<_> = provider["operations"]
        .as_array()
        .ok_or("provider operations required")?
        .iter()
        .filter(|o| o["operation_id"] == stored["operation_id"])
        .collect();
    if operations.len() != 1 {
        return Err("provider operation missing or ambiguous".into());
    }
    let operation = operations[0];
    source::inline(&operation["input_schema"])?;
    source::inline(&operation["output_schema"])?;
    jsonschema::validate(&operation["input_schema"], &stored["input"])
        .map_err(|e| e.to_string())?;
    let profile = root.file(&input["profile_ref"])?;
    source::schema("owner_admission", &profile)?;
    let authority = root.bound(&input["authority_ref"])?;
    source::schema("authority_consumption", &authority)?;
    if authority["profile_ref"] != input["profile_ref"]
        || authority["owner_id"] != profile["owner_id"]
        || authority["repository_id"] != profile["repository_id"]
        || authority["actor_id"] != input["actor_id"]
        || authority["engine_id"] != manifest["engine_id"]
        || authority["capability_id"] != contract["contract_id"]
        || authority["contract_version"] != contract["version"]
        || authority["operation_id"] != "storage.invoke"
        || authority["invocation_id"] != registration["provider_id"]
        || authority["request_sha256"] != input["request_ref"]["sha256"]
        || authority["artifact_sha256"] != binding["artifact_ref"]["sha256"]
    {
        return Err("provider invocation has no exact admitted current authority".into());
    }
    if !profile["actors"]
        .as_array()
        .ok_or("admitted actors required")?
        .contains(&input["actor_id"])
        || profile["operation_subjects"]
            .as_array()
            .ok_or("operation subjects required")?
            .iter()
            .filter(|s| {
                s["engine_id"] == manifest["engine_id"]
                    && s["capability_id"] == contract["contract_id"]
                    && s["contract_version"] == contract["version"]
                    && s["operation_id"] == "storage.invoke"
            })
            .count()
            != 1
    {
        return Err("provider operation subject mandate missing or ambiguous".into());
    }
    let starts = authority["not_before_unix_ms"]
        .as_u64()
        .ok_or("authority start required")?;
    let expires = authority["expires_unix_ms"]
        .as_u64()
        .ok_or("authority expiration required")?;
    let is_current = || -> bool {
        let check = || -> Result<bool, String> {
            let now = time_ms()?;
            Ok(starts <= now
                && now < expires
                && source::hash(
                    &root.bytes(source::text(&input["profile_ref"], "path")?, 2_097_152)?,
                ) == input["profile_ref"]["sha256"]
                && source::hash(
                    &root.bytes(source::text(&input["authority_ref"], "path")?, 2_097_152)?,
                ) == input["authority_ref"]["sha256"])
        };
        check().unwrap_or(false)
    };
    root.finish()?;
    if !is_current() {
        return Err("provider authority expired or changed".into());
    }
    let provider_request = json!({"operation_id":stored["operation_id"],"input":stored["input"]});
    source::schema("provider_request", &provider_request)?;
    let payload = serde_json::to_vec(&provider_request).map_err(|e| e.to_string())?;
    if payload.len() > 1_048_576 {
        return Err("provider request byte budget exceeded".into());
    }
    let mut process = Command::new(artifact)
        .current_dir(root.directory())
        .env_clear()
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;
    let mut stdin = process.stdin.take().ok_or("provider stdin unavailable")?;
    let stdout = process.stdout.take().ok_or("provider stdout unavailable")?;
    let stderr = process.stderr.take().ok_or("provider stderr unavailable")?;
    std::thread::spawn(move || {
        let _ = stdin.write_all(&payload);
        drop(stdin);
    });
    let (sender, receiver) = mpsc::channel();
    std::thread::spawn(move || {
        let mut bytes = Vec::new();
        let result = stdout.take(1_048_577).read_to_end(&mut bytes);
        let _ = sender.send(result.map(|_| bytes));
    });
    std::thread::spawn(move || {
        let mut bytes = Vec::new();
        let _ = stderr.take(65_537).read_to_end(&mut bytes);
    });
    let started = Instant::now();
    let budget = Duration::from_millis(
        input["budget_ms"]
            .as_u64()
            .ok_or("provider budget required")?,
    );
    let mut cause = "COMPLETED";
    let mut captured = None;
    let mut ack = Value::Null;
    let mut stop = Value::Null;
    let exit = loop {
        if let Ok(data) = receiver.try_recv() {
            match data {
                Ok(bytes) if bytes.len() <= 1_048_576 => captured = Some(bytes),
                _ => cause = "OUTPUT_LIMIT",
            }
        }
        if !is_current() {
            cause = "AUTHORITY_REVOKED";
        } else if started.elapsed() >= budget {
            cause = "TIMEOUT";
        }
        if cause != "COMPLETED" {
            ack = json!({"correlation_id":input["attempt_id"],"status":"ACCEPTED","cause":cause});
            let signaled = process.kill().is_ok();
            let status = process.wait().map_err(|e| e.to_string())?;
            stop = json!({"correlation_id":input["attempt_id"],"attempt_id":input["attempt_id"],"status":if signaled {"STOP_CONFIRMED"} else {"UNKNOWN"},"scope":"DIRECT_CHILD_ONLY","effect_evidence_refs":[]});
            break status;
        }
        if let Some(status) = process.try_wait().map_err(|e| e.to_string())? {
            break status;
        }
        std::thread::sleep(Duration::from_millis(5));
    };
    if captured.is_none() {
        if let Ok(Ok(bytes)) = receiver.recv_timeout(Duration::from_millis(250)) {
            if bytes.len() <= 1_048_576 {
                captured = Some(bytes);
            } else if cause == "COMPLETED" {
                cause = "OUTPUT_LIMIT";
            }
        }
    }
    let mut response = Value::Null;
    if cause == "COMPLETED" {
        match captured
            .as_deref()
            .ok_or_else(|| "provider response unavailable".to_owned())
            .and_then(source::parse)
        {
            Ok(value) => {
                if !exit.success() || value["status"] != "PASS" {
                    cause = "PROVIDER_FAILED";
                } else if source::schema("provider_response", &value).is_err()
                    || !value["error"].is_null()
                    || jsonschema::validate(&operation["output_schema"], &value["output"]).is_err()
                {
                    cause = "INVALID_PROVIDER_RESPONSE";
                }
                response = value;
            }
            Err(_) => cause = "INVALID_PROVIDER_RESPONSE",
        }
    }
    if root.finish().is_err() && cause == "COMPLETED" {
        cause = "AUTHORITY_REVOKED";
    }
    Ok(
        json!({"attempt_id":input["attempt_id"],"invocation_id":registration["provider_id"],"invoked":true,"attempt_status":if cause=="COMPLETED"{"SUCCEEDED"}else{"FAILED"},"cause":cause,"exit_code":exit.code(),"response":response,"effect_status":"UNKNOWN","authority_consumed":true,"authority_sha256":input["authority_ref"]["sha256"],"request_sha256":input["request_ref"]["sha256"],"artifact_sha256":binding["artifact_ref"]["sha256"],"stop_ack":ack,"stop_result":stop,"descendant_stop":"NOT_ASSERTED","normative_permission_created":false,"os_sandbox":"NOT_PROVIDED"}),
    )
}
