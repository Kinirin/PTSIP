use crate::source::{self, Root};
use serde_json::{json, Value};
use std::collections::{BTreeMap, BTreeSet};

pub fn prepare(operation: &str, input: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "root")?)?;
    let catalog = root.bound(&input["catalog_ref"])?;
    source::schema("catalog", &catalog)?;
    let mut cases = BTreeMap::new();
    for reference in catalog["cases"]
        .as_array()
        .ok_or("case references required")?
    {
        let case = root.bound(reference)?;
        source::schema("case", &case)?;
        source::inline(&case["input_schema"])?;
        source::inline(&case["output_schema"])?;
        jsonschema::validate(&case["input_schema"], &case["input"])
            .map_err(|e| format!("case input: {e}"))?;
        let id = source::text(&case, "id")?.to_owned();
        if cases.insert(id, case).is_some() {
            return Err("ambiguous registered case identity".into());
        }
    }
    let mut selectors = BTreeMap::new();
    for reference in catalog["selectors"]
        .as_array()
        .ok_or("selector references required")?
    {
        let selector = root.bound(reference)?;
        source::schema("selector", &selector)?;
        for id in selector["case_ids"]
            .as_array()
            .ok_or("selector IDs required")?
        {
            if !cases.contains_key(id.as_str().ok_or("selector case ID required")?) {
                return Err("registered selector references unknown case".into());
            }
        }
        if selectors
            .insert(source::text(&selector, "id")?.to_owned(), selector)
            .is_some()
        {
            return Err("ambiguous registered selector identity".into());
        }
    }
    if operation == "validation.prepare-cases" {
        root.finish()?;
        return Ok(
            json!({"catalog_revision":input["catalog_ref"]["sha256"],"cases":cases.values().collect::<Vec<_>>(),"selectors":selectors.values().collect::<Vec<_>>(),"ptsip_conformance_verdict":"NOT_ASSERTED","execution_authorized":false}),
        );
    }
    let selection = &input["selection"];
    let explicit = selection["case_ids"]
        .as_array()
        .ok_or("explicit case IDs required")?;
    let selected = if let Some(selector_id) = selection["selector_id"].as_str() {
        if !explicit.is_empty() {
            return Err("conflicting explicit IDs and registered selector".into());
        }
        selectors
            .get(selector_id)
            .ok_or("unknown registered selector")?["case_ids"]
            .as_array()
            .ok_or("selector IDs required")?
            .clone()
    } else {
        explicit.clone()
    };
    let mut seen = BTreeSet::new();
    let mut selected_cases = Vec::new();
    for id in &selected {
        let id = id.as_str().ok_or("selected case identity required")?;
        if !seen.insert(id) {
            return Err("duplicate selected case".into());
        }
        selected_cases.push(cases.get(id).ok_or("unknown selected case")?);
    }
    if operation == "validation.select-cases" {
        root.finish()?;
        return Ok(
            json!({"catalog_revision":input["catalog_ref"]["sha256"],"selected_ids":selected,"cases":selected_cases,"execution_authorized":false,"ptsip_conformance_verdict":"NOT_ASSERTED"}),
        );
    }
    let mut adapters = Vec::new();
    for case in selected_cases {
        let adapter = root.bound(&case["adapter_ref"])?;
        source::schema("adapter", &adapter)?;
        source::inline(&adapter["input_schema"])?;
        source::inline(&adapter["output_schema"])?;
        jsonschema::validate(&adapter["input_schema"], &case["input"])
            .map_err(|e| format!("adapter preflight input: {e}"))?;
        if adapter["output_schema"] != case["output_schema"] {
            return Err("adapter/case output contract requires explicit compatible binding".into());
        }
        adapters.push(json!({"case_id":case["id"],"invocation_id":adapter["invocation_id"],"adapter_revision":case["adapter_ref"]["sha256"]}));
    }
    root.finish()?;
    Ok(
        json!({"catalog_revision":input["catalog_ref"]["sha256"],"selected_ids":selected,"prepared_adapters":adapters,"all_preflight_passed":true,"execution_authorized":false,"ptsip_conformance_verdict":"NOT_ASSERTED"}),
    )
}
