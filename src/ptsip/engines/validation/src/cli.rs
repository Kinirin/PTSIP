use crate::{dispatch, source, CONTRACT, MANIFEST};
use serde_json::{json, Value};
use std::io::{self, Read};

pub(crate) fn run() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args == ["--help"] || args == ["-h"] {
        println!("PTSIP Validation Engine 0.3.4 / capability contract 4.1.0\nInput: one versioned JSON request on stdin.\nSupports internal $ref (JSON Pointer, anchors, recursive schemas), external $id references and $dynamicRef/$dynamicAnchor.\nschema_resources: explicit local resources.\nschema_resolution.directories: recursively discover JSON schemas inside explicitly permitted roots.\nschema_resolution.allow_network + allowed_origins: retrieve missing HTTP(S) references from exact permitted origins; redirects and ambient proxies disabled.\nOptional base_uri resolves relative references for anonymous input schemas.\nResults include schema_resources_used (URI, source, SHA256).\nLimits: 128 schema documents, 1 MiB per acquired document, 8 MiB total, 16384 schema nodes, 5s per HTTP request and 20s acquisition budget per schema evaluation.\nGuide: https://github.com/Kinirin/PTSIP/blob/main/docs/validation-schema-references.md");
        return;
    }
    if args == ["--manifest"] {
        println!("{MANIFEST}");
        return;
    }
    let mut bytes = Vec::new();
    let result = if !args.is_empty() {
        Err("unsupported invocation arguments".into())
    } else {
        io::stdin()
            .take(1048577)
            .read_to_end(&mut bytes)
            .map_err(|e| e.to_string())
            .and_then(|_| {
                if bytes.len() > 1048576 {
                    return Err("request exceeds registered byte limit".into());
                }
                let request = source::parse(&bytes)?;
                dispatch(&request)
            })
    };
    let manifest: Value = serde_json::from_str(MANIFEST).expect("compiled manifest must be valid");
    let contract: Value = serde_json::from_str(CONTRACT).expect("compiled contract must be valid");
    let (status, output, error) = match result {
        Ok(value) => ("PASS", value, Value::Null),
        Err(error) => ("FAIL", Value::Null, json!(error)),
    };
    println!(
        "{}",
        json!({"schema_version":"ptsip-engine-operation-result/v1","engine_id":manifest["engine_id"],"engine_version":manifest["implementation_version"],"contract_id":contract["contract_id"],"contract_version":contract["version"],"status":status,"output":output,"error":error,"normative_authority":false,"execution_authorized":false,"evidence_authentication":"NOT_PROVIDED_BY_RESULT"})
    );
    if status == "FAIL" {
        std::process::exit(2);
    }
}
