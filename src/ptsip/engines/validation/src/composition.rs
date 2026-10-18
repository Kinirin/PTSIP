use serde_json::{json, Value};
use std::collections::{BTreeMap, BTreeSet};

pub fn validate(input: &Value) -> Result<Value, String> {
    let checks = input["checks"]
        .as_array()
        .ok_or("explicit check composition required")?;
    let mut pending = BTreeMap::new();
    let mut definitions = BTreeMap::new();
    for check in checks {
        let id = check["check_id"]
            .as_str()
            .ok_or("typed check identity required")?
            .to_owned();
        if definitions.insert(id.clone(), check).is_some() {
            return Err("duplicate composed check identity".into());
        }
        let dependencies: BTreeSet<_> = check["depends_on"]
            .as_array()
            .ok_or("explicit dependency list required")?
            .iter()
            .map(|v| v.as_str().unwrap_or("").to_owned())
            .collect();
        pending.insert(id, dependencies);
    }
    if pending
        .values()
        .any(|deps| deps.iter().any(|d| !pending.contains_key(d)))
    {
        return Err("unregistered check dependency".into());
    }
    let mut results = Vec::new();
    let mut valid_all = true;
    while !pending.is_empty() {
        let id = pending
            .iter()
            .find(|(_, d)| d.is_empty())
            .map(|(id, _)| id.clone())
            .ok_or("composed validation dependency cycle")?;
        let check = definitions[&id];
        if !matches!(
            check["operation_id"].as_str(),
            Some(
                "validation.contract"
                    | "validation.schema"
                    | "validation.typed-io"
                    | "validation.precondition"
                    | "validation.case"
            )
        ) {
            return Err("unsupported independent validation capability".into());
        }
        let (diagnostics, evidence) = crate::schema::validate_with_resolution(
            &check["schema"],
            &check["instance"],
            input.get("schema_resources"),
            input.get("schema_resolution"),
        )?;
        valid_all &= diagnostics.is_empty();
        results
            .push(json!({"check_id":id,"valid":diagnostics.is_empty(),"diagnostics":diagnostics,"schema_resources_used":evidence}));
        pending.remove(&id);
        for deps in pending.values_mut() {
            deps.remove(&id);
        }
    }
    Ok(json!({"checks":results,"valid_all":valid_all,"ptsip_conformance_verdict":"NOT_ASSERTED"}))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn checks_remain_independent_and_composition_never_claims_conformance() {
        let input = json!({"checks":[{"check_id":"b","operation_id":"validation.typed-io","schema":{"type":"integer"},"instance":"bad","depends_on":["a"]},{"check_id":"a","operation_id":"validation.schema","schema":{"type":"string"},"instance":"ok","depends_on":[]}]});
        let result = validate(&input).unwrap();
        assert_eq!(result["checks"][0]["check_id"], "a");
        assert_eq!(result["checks"][0]["valid"], true);
        assert_eq!(result["checks"][1]["valid"], false);
        assert_eq!(result["ptsip_conformance_verdict"], "NOT_ASSERTED");
    }
    #[test]
    fn unknown_composition_cycles_and_external_refs_fail_closed() {
        for input in [
            json!({"checks":[{"check_id":"a","operation_id":"validation.case","schema":{},"instance":{},"depends_on":["a"]}]}),
            json!({"checks":[{"check_id":"a","operation_id":"vpms.run","schema":{},"instance":{},"depends_on":[]}]}),
            json!({"checks":[{"check_id":"a","operation_id":"validation.schema","schema":{"$ref":"https://invalid.example/schema"},"instance":{},"depends_on":[]}]}),
        ] {
            assert!(validate(&input).is_err());
        }
    }
}
