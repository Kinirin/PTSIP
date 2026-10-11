use crate::source::{self, Root};
use serde_json::{json, Value};
use std::collections::{BTreeMap, BTreeSet};

pub fn prepare(input: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "root")?)?;
    let plan = root.bound(&input["plan_ref"])?;
    source::schema("native_plan", &plan)?;
    let mut steps = BTreeMap::new();
    let mut dependencies = BTreeMap::new();
    let mut requests = Vec::new();
    for step in plan["steps"]
        .as_array()
        .ok_or("native plan steps required")?
    {
        let id = source::text(step, "id")?.to_owned();
        let deps: BTreeSet<String> = step["dependencies"]
            .as_array()
            .ok_or("step dependencies required")?
            .iter()
            .map(|v| v.as_str().unwrap_or("").to_owned())
            .collect();
        if dependencies.insert(id.clone(), deps).is_some()
            || steps.insert(id, step.clone()).is_some()
        {
            return Err("duplicate native plan step".into());
        }
        let (invocation, _, _) = source::invocation(
            &mut root,
            &input["registry_ref"],
            source::text(step, "invocation_id")?,
        )?;
        requests.push(invocation["request_ref"].clone());
    }
    if dependencies
        .values()
        .any(|ds| ds.iter().any(|id| !steps.contains_key(id)))
    {
        return Err("unknown native plan dependency".into());
    }
    let mut ordered = Vec::new();
    while !dependencies.is_empty() {
        let id = dependencies
            .iter()
            .find(|(_, ds)| ds.is_empty())
            .map(|(id, _)| id.clone())
            .ok_or("native plan cycle")?;
        dependencies.remove(&id);
        for ds in dependencies.values_mut() {
            ds.remove(&id);
        }
        ordered.push(steps[&id].clone());
    }
    root.finish()?;
    Ok(
        json!({"plan_revision":input["plan_ref"]["sha256"],"registry_revision":input["registry_ref"]["sha256"],"ordered_steps":ordered,"request_refs":requests,"execution_authorized":false}),
    )
}
