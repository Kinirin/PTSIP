use crate::source::{self, Root};
use serde_json::{json, Value};

pub fn verify(input: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "root")?)?;
    let id = source::text(input, "invocation_id")?;
    let (invocation, _, _) = source::invocation(&mut root, &input["registry_ref"], id)?;
    let registry = root.bound(&input["registry_ref"])?;
    crate::registry::verify(
        &json!({"root":input["root"],"index_ref":registry["index_ref"],"expected_revision":null,"engine_id":invocation["engine_id"],"capability_id":invocation["capability_id"],"version":invocation["engine_version"],"contract_id":invocation["capability_id"]}),
        true,
    )?;
    root.finish()?;
    Ok(
        json!({"invocation_id":id,"engine_id":invocation["engine_id"],"operation_id":invocation["operation_id"],"request_sha256":invocation["request_ref"]["sha256"],"artifact_sha256":invocation["artifact_ref"]["sha256"],"registry_revision":input["registry_ref"]["sha256"],"source_refs":root.refs(),"execution_authorized":false}),
    )
}
