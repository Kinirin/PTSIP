use crate::authority::Consumed;
use crate::source::{self, Root};
use serde_json::{json, Value};
use std::io::{Read, Write};
use std::process::{Command, Stdio};
use std::sync::mpsc;
use std::time::{Duration, Instant};

pub fn invoke(input: &Value, manifest: &Value, contract: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "root")?)?;
    let id = source::text(input, "invocation_id")?;
    let (invocation, request, artifact) =
        source::invocation(&mut root, &input["registry_ref"], id)?;
    if matches!(
        invocation["operation_id"].as_str(),
        Some("execution.invoke" | "storage.invoke")
    ) {
        return Err(
            "nested process invocation requires its own separately admitted orchestration contract"
                .into(),
        );
    }
    let consumed = Consumed::check(
        &mut root,
        input,
        manifest,
        contract,
        "execution.invoke",
        id,
        &invocation["request_ref"]["sha256"],
        &invocation["artifact_ref"]["sha256"],
    )?;
    root.finish()?;
    if !consumed.current(&root) {
        return Err("authority changed before native invocation".into());
    }
    let encoded = serde_json::to_vec(&request).map_err(|e| e.to_string())?;
    if encoded.len() > 1_048_576 {
        return Err("native request byte budget exceeded".into());
    }
    let mut child = Command::new(artifact)
        .current_dir(root.directory())
        .env_clear()
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;
    let mut stdin = child.stdin.take().ok_or("native stdin unavailable")?;
    let stdout = child.stdout.take().ok_or("native stdout unavailable")?;
    let stderr = child.stderr.take().ok_or("native stderr unavailable")?;
    let (tx, rx) = mpsc::channel();
    std::thread::spawn(move || {
        let result = stdin.write_all(&encoded);
        drop(stdin);
        let _ = result;
    });
    std::thread::spawn(move || {
        let mut bytes = Vec::new();
        let result = stdout.take(1_048_577).read_to_end(&mut bytes);
        let _ = tx.send(result.map(|_| bytes));
    });
    std::thread::spawn(move || {
        let mut discarded = Vec::new();
        let _ = stderr.take(65_537).read_to_end(&mut discarded);
    });
    let started = Instant::now();
    let budget = Duration::from_millis(
        input["budget_ms"]
            .as_u64()
            .ok_or("native budget required")?,
    );
    let mut cause = "COMPLETED";
    let mut captured = None;
    let mut stop_ack = Value::Null;
    let mut stop_result = Value::Null;
    let exit = loop {
        if let Ok(output) = rx.try_recv() {
            match output {
                Ok(bytes) if bytes.len() <= 1_048_576 => captured = Some(bytes),
                _ => cause = "OUTPUT_LIMIT",
            }
        }
        if !consumed.current(&root) {
            cause = "AUTHORITY_REVOKED";
        } else if started.elapsed() >= budget {
            cause = "TIMEOUT";
        }
        if cause != "COMPLETED" {
            stop_ack =
                json!({"correlation_id":input["attempt_id"],"status":"ACCEPTED","cause":cause});
            let killed = child.kill().is_ok();
            let status = child.wait().map_err(|e| e.to_string())?;
            stop_result = json!({"correlation_id":input["attempt_id"],"attempt_id":input["attempt_id"],"status":if killed {"STOP_CONFIRMED"} else {"UNKNOWN"},"scope":"DIRECT_CHILD_ONLY","effect_evidence_refs":[]});
            break status;
        }
        if let Some(status) = child.try_wait().map_err(|e| e.to_string())? {
            break status;
        }
        std::thread::sleep(Duration::from_millis(5));
    };
    if captured.is_none() {
        if let Ok(Ok(bytes)) = rx.recv_timeout(Duration::from_millis(250)) {
            if bytes.len() <= 1_048_576 {
                captured = Some(bytes);
            } else if cause == "COMPLETED" {
                cause = "OUTPUT_LIMIT";
            }
        }
    }
    let mut response = Value::Null;
    if cause == "COMPLETED" {
        let parsed = captured
            .as_deref()
            .ok_or_else(|| "native response unavailable".to_owned())
            .and_then(source::parse);
        match parsed {
            Ok(value) => {
                let bindings = root.file(&invocation["binding_ref"])?;
                let binding = bindings["bindings"]
                    .as_array()
                    .ok_or("target bindings required")?
                    .iter()
                    .find(|b| {
                        b["capability_id"] == invocation["capability_id"]
                            && b["contract_version"] == invocation["contract_version"]
                    })
                    .ok_or("target binding missing")?;
                let transport = root.bound(&binding["transport_ref"])?;
                let target_contract = root.bound(&invocation["contract_ref"])?;
                let definition = target_contract["operations"]
                    .as_array()
                    .ok_or("target operations required")?
                    .iter()
                    .find(|o| o["operation_id"] == invocation["operation_id"])
                    .ok_or("target operation missing")?;
                let identity = value["engine_id"] == invocation["engine_id"]
                    && value["engine_version"] == invocation["engine_version"]
                    && value["contract_id"] == invocation["capability_id"]
                    && value["contract_version"] == invocation["contract_version"]
                    && value["normative_authority"] == false
                    && value["execution_authorized"] == false;
                if !exit.success() || value["status"] != "PASS" {
                    cause = "PROVIDER_FAILED";
                } else if !identity
                    || !value["error"].is_null()
                    || jsonschema::validate(&transport["response_schema"], &value).is_err()
                    || jsonschema::validate(&definition["typed_output"], &value["output"]).is_err()
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
        json!({"attempt_id":input["attempt_id"],"invocation_id":id,"invoked":true,"attempt_status":if cause=="COMPLETED" {"SUCCEEDED"} else {"FAILED"},"cause":cause,"exit_code":exit.code(),"response":response,"effect_status":"UNKNOWN","authority_consumed":true,"authority_sha256":input["authority_ref"]["sha256"],"request_sha256":invocation["request_ref"]["sha256"],"artifact_sha256":invocation["artifact_ref"]["sha256"],"stop_ack":stop_ack,"stop_result":stop_result,"descendant_stop":"NOT_ASSERTED","normative_permission_created":false,"os_sandbox":"NOT_PROVIDED"}),
    )
}
