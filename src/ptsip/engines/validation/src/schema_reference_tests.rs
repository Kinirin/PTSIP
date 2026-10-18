use crate::schema::{prepare, validate_with_resolution};
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::io::{Read, Write};
use std::net::TcpListener;
use std::path::PathBuf;
use std::time::{SystemTime, UNIX_EPOCH};
struct Directory(PathBuf);
impl Directory {
    fn new() -> Self {
        let p = std::env::temp_dir().join(format!(
            "ptsip-b5-{}-{}",
            std::process::id(),
            SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&p).unwrap();
        Self(p)
    }
}
impl Drop for Directory {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}
fn valid(schema: &Value, instance: &Value, resources: Option<&Value>) -> bool {
    validate_with_resolution(schema, instance, resources, None)
        .unwrap()
        .0
        .is_empty()
}
#[test]
fn pointers_anchors_nested_ids_and_recursive_data() {
    let s = json!({"$defs":{"name":{"$anchor":"name","type":"string","minLength":2}},"type":"object","properties":{"a":{"$ref":"#/$defs/name"},"b":{"$ref":"#name"},"child":{"$ref":"#"}}});
    assert!(valid(
        &s,
        &json!({"a":"ok","b":"yes","child":{"a":"fine"}}),
        None
    ));
    assert!(!valid(&s, &json!({"child":{"a":2}}), None));
    let bundled = json!({"$id":"https://example.invalid/main","$ref":"child","$defs":{"child":{"$id":"child","properties":{"value":{"$ref":"#/$defs/text"}},"$defs":{"text":{"type":"string"}}}}});
    assert!(valid(&bundled, &json!({"value":"ok"}), None));
    assert!(!valid(&bundled, &json!({"value":5}), None));
    let escaped = json!({"$defs":{"a/b~c":{"type":"integer"}},"$ref":"#/$defs/a~1b~0c"});
    assert!(valid(&escaped, &json!(4), None));
}
#[test]
fn dynamic_scope_extends_a_recursive_tree() {
    let resources = json!([{"$id":"https://example.invalid/tree","$dynamicAnchor":"node","type":"object","properties":{"data":true,"children":{"type":"array","items":{"$dynamicRef":"#node"}}}}]);
    let strict = json!({"$id":"https://example.invalid/strict","$dynamicAnchor":"node","$ref":"https://example.invalid/tree","unevaluatedProperties":false});
    assert!(valid(
        &strict,
        &json!({"data":1,"children":[{"data":2}]}),
        Some(&resources)
    ));
    assert!(!valid(
        &strict,
        &json!({"children":[{"unknown":2}]}),
        Some(&resources)
    ));
}
#[test]
fn invalid_missing_duplicate_and_annotation_cases() {
    for r in [
        json!([{"type":"string"}]),
        json!([{"$id":"relative"}]),
        json!([{"$id":"urn:x"},{"$id":"urn:x"}]),
    ] {
        assert!(prepare(&json!({}), Some(&r), None).is_err());
    }
    assert!(prepare(&json!({"$ref":"#/$defs/missing"}), None, None).is_err());
    assert!(prepare(&json!({"$dynamicRef":3}), None, None).is_err());
    let s = json!({"type":"string","examples":[{"$ref":"https://invalid/no","$dynamicRef":"#x"}],"default":{"$ref":"#"}});
    assert!(valid(&s, &json!("ok"), None));
    assert!(valid(&json!(true), &json!(null), None));
    assert!(!valid(&json!(false), &json!(null), None));
    let r = json!([{"$id":"urn:test:string","type":"string"}]);
    assert!(valid(
        &json!({"$ref":"urn:test:string"}),
        &json!("ok"),
        Some(&r)
    ));
    assert!(prepare(&json!({"$ref":"urn:test:string"}), None, None).is_err());
}
#[test]
fn recursive_file_discovery_id_lookup_relative_file_refs_and_hashes() {
    let dir = Directory::new();
    std::fs::create_dir(dir.0.join("nested")).unwrap();
    let bytes = br#"{"$id":"urn:local:text","type":"string"}"#;
    std::fs::write(dir.0.join("nested/text.json"), bytes).unwrap();
    std::fs::write(dir.0.join("number.json"), br#"{"type":"integer"}"#).unwrap();
    let config = json!({"directories":[dir.0],"base_uri":url::Url::from_file_path(dir.0.join("root.json")).unwrap().as_str()});
    let (errors, evidence) = validate_with_resolution(
        &json!({"$ref":"urn:local:text"}),
        &json!("ok"),
        None,
        Some(&config),
    )
    .unwrap();
    assert!(errors.is_empty());
    assert!(evidence
        .iter()
        .any(|e| e["source"] == "FILE_BYTES"
            && e["sha256"] == format!("{:x}", Sha256::digest(bytes))));
    assert!(validate_with_resolution(
        &json!({"$ref":"number.json"}),
        &json!(3),
        None,
        Some(&config)
    )
    .unwrap()
    .0
    .is_empty());
    assert!(!validate_with_resolution(
        &json!({"$ref":"number.json"}),
        &json!("bad"),
        None,
        Some(&config)
    )
    .unwrap()
    .0
    .is_empty());
    assert!(prepare(&json!({"$ref":"urn:local:text"}), None, None).is_err());
}
#[test]
fn file_roots_conflicting_ids_and_resource_limits_fail_closed() {
    let dir = Directory::new();
    let allowed = dir.0.join("allowed");
    std::fs::create_dir(&allowed).unwrap();
    let outside = dir.0.join("outside.json");
    std::fs::write(&outside, "{}").unwrap();
    let config = json!({"directories":[allowed]});
    let ref_uri = url::Url::from_file_path(outside).unwrap().to_string();
    let error = prepare(&json!({"$ref":ref_uri}), None, Some(&config))
        .err()
        .unwrap();
    assert!(error.contains("SCHEMA_FILE_DENIED"), "{error}");
    std::fs::write(
        allowed.join("a.json"),
        r#"{"$id":"urn:duplicate","type":"string"}"#,
    )
    .unwrap();
    std::fs::write(
        allowed.join("b.json"),
        r#"{"$id":"urn:duplicate","type":"integer"}"#,
    )
    .unwrap();
    assert!(prepare(&json!({}), None, Some(&config))
        .err()
        .unwrap()
        .contains("DUPLICATE_SCHEMA_ID"));
    assert!(prepare(&json!({}), None, Some(&json!({"allow_network":true}))).is_err());
    assert!(prepare(&json!({}), None, Some(&json!({"unknown":true}))).is_err());
}
#[test]
fn network_acquisition_is_opt_in_transitive_bounded_and_recorded() {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let origin = format!("http://{}", listener.local_addr().unwrap());
    let uri = format!("{origin}/first");
    assert!(prepare(&json!({"$ref":uri}), None, None).is_err());
    assert!(prepare(
        &json!({"$ref":uri}),
        None,
        Some(&json!({"allow_network":true,"allowed_origins":["https://example.invalid"]}))
    )
    .err()
    .unwrap()
    .contains("SCHEMA_NETWORK_DENIED"));
    let server = std::thread::spawn(move || {
        for (index, stream) in listener.incoming().take(2).enumerate() {
            let mut stream = stream.unwrap();
            stream
                .set_read_timeout(Some(std::time::Duration::from_secs(5)))
                .unwrap();
            let mut request = [0; 4096];
            stream.read(&mut request).unwrap();
            let body = if index == 0 {
                r#"{"$ref":"second"}"#
            } else {
                r##"{"$defs":{"text":{"type":"string"}},"$ref":"#/$defs/text"}"##
            };
            write!(
                stream,
                "HTTP/1.1 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                body.len(),
                body
            )
            .unwrap();
        }
    });
    let config = json!({"allow_network":true,"allowed_origins":[origin]});
    let (errors, evidence) =
        validate_with_resolution(&json!({"$ref":uri}), &json!("ok"), None, Some(&config)).unwrap();
    assert!(errors.is_empty());
    assert_eq!(
        evidence
            .iter()
            .filter(|e| e["source"] == "NETWORK_BYTES")
            .count(),
        2
    );
    server.join().unwrap();
}
#[test]
fn acquired_document_and_total_resource_budgets_are_enforced() {
    let dir = Directory::new();
    std::fs::write(dir.0.join("large.json"), vec![b' '; 1024 * 1024 + 1]).unwrap();
    let config = json!({"directories": [dir.0]});
    assert!(prepare(&json!({}), None, Some(&config))
        .err()
        .unwrap()
        .contains("SCHEMA_RESOURCE_LIMIT"));
    let resources = Value::Array(
        (0..128)
            .map(|i| json!({"$id": format!("urn:budget:{i}")}))
            .collect(),
    );
    assert!(prepare(&json!({}), Some(&resources), None)
        .err()
        .unwrap()
        .contains("SCHEMA_RESOURCE_LIMIT"));
}
#[test]
fn network_redirects_do_not_escape_the_allowed_origin() {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let origin = format!("http://{}", listener.local_addr().unwrap());
    let server = std::thread::spawn(move || {
        let (mut stream, _) = listener.accept().unwrap();
        let mut request = [0; 4096];
        stream.read(&mut request).unwrap();
        write!(stream,"HTTP/1.1 302 Found\r\nLocation: http://127.0.0.1:1/private\r\nContent-Length: 0\r\nConnection: close\r\n\r\n").unwrap();
    });
    let config = json!({"allow_network":true,"allowed_origins":[origin]});
    assert!(prepare(
        &json!({"$ref":format!("{origin}/redirect")}),
        None,
        Some(&config)
    )
    .err()
    .unwrap()
    .contains("SCHEMA_NETWORK_ERROR"));
    server.join().unwrap();
}
#[test]
fn pinned_official_2020_12_reference_cases() {
    let mut passed = 0;
    let mut excluded = 0;
    for text in [
        include_str!("schema_fixtures/ref.json"),
        include_str!("schema_fixtures/dynamicRef.json"),
    ] {
        let cases: Value = serde_json::from_str(text).unwrap();
        for case in cases.as_array().unwrap() {
            if case["schema"].to_string().contains("http://localhost:1234") {
                excluded += case["tests"].as_array().unwrap().len();
                continue;
            }
            let prepared = prepare(&case["schema"], None, None)
                .unwrap_or_else(|e| panic!("{}: {e}", case["description"]));
            for test in case["tests"].as_array().unwrap() {
                assert_eq!(
                    prepared.validator.is_valid(&test["data"]),
                    test["valid"].as_bool().unwrap(),
                    "{} / {}",
                    case["description"],
                    test["description"]
                );
                passed += 1;
            }
        }
    }
    assert!(passed > 100);
    println!("Pinned official reference cases: {passed} passed, {excluded} localhost fixture cases excluded; scoped HTTP covered separately");
}
