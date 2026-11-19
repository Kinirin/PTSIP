use crate::{dispatch, source, CONTRACT, MANIFEST};
use serde_json::{json, Value};
use std::io::{self, BufRead, Read, Write};

const MAX_REQUEST_BYTES: u64 = 1_048_576;

fn response(result: Result<Value, String>, authorized: bool) -> Value {
    let manifest: Value = serde_json::from_str(MANIFEST).expect("compiled manifest must be valid");
    let contract: Value = serde_json::from_str(CONTRACT).expect("compiled contract must be valid");
    let (status, output, error) = match result {
        Ok(value) => ("PASS", value, Value::Null),
        Err(error) => ("FAIL", Value::Null, json!(error)),
    };
    json!({"schema_version":"ptsip-engine-operation-result/v1","engine_id":manifest["engine_id"],"engine_version":manifest["implementation_version"],"contract_id":contract["contract_id"],"contract_version":contract["version"],"status":status,"output":output,"error":error,"normative_authority":false,"execution_authorized":authorized && status == "PASS","evidence_authentication":"NOT_PROVIDED_BY_RESULT"})
}

fn evaluate_bytes(bytes: &[u8]) -> Value {
    let request = source::parse(bytes);
    let authorized = request
        .as_ref()
        .ok()
        .and_then(|r| r["operation_id"].as_str())
        .is_some_and(|operation| crate::user_contracts::algorithm(operation).is_some());
    response(request.and_then(|request| dispatch(&request)), authorized)
}

fn emit(output: &mut impl Write, value: &Value) -> io::Result<()> {
    serde_json::to_writer(&mut *output, value)?;
    output.write_all(b"\n")?;
    output.flush()
}

fn serve() -> io::Result<()> {
    let input = io::stdin();
    let mut input = input.lock();
    let output = io::stdout();
    let mut output = output.lock();
    loop {
        let mut bytes = Vec::new();
        let count = (&mut input)
            .take(MAX_REQUEST_BYTES + 2)
            .read_until(b'\n', &mut bytes)?;
        if count == 0 {
            return Ok(());
        }
        if bytes.last() == Some(&b'\n') {
            bytes.pop();
        }
        if bytes.last() == Some(&b'\r') {
            bytes.pop();
        }
        if bytes.len() as u64 > MAX_REQUEST_BYTES {
            emit(
                &mut output,
                &response(Err("request exceeds registered byte limit".into()), false),
            )?;
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                "request exceeds registered byte limit",
            ));
        }
        // Each complete line is an independent envelope. A diagnostic does not
        // terminate subsequent requests; EOF closes the caller-owned session.
        emit(&mut output, &evaluate_bytes(&bytes))?;
    }
}

pub(crate) fn run() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args == ["--manifest"] {
        println!("{MANIFEST}");
        return;
    }
    if args == ["--serve"] {
        if let Err(error) = serve() {
            eprintln!("{error}");
            std::process::exit(2);
        }
        return;
    }
    let mut bytes = Vec::new();
    let value = if !args.is_empty() {
        response(Err("unsupported invocation arguments".into()), false)
    } else {
        match io::stdin()
            .take(MAX_REQUEST_BYTES + 1)
            .read_to_end(&mut bytes)
        {
            Err(error) => response(Err(error.to_string()), false),
            Ok(_) if bytes.len() as u64 > MAX_REQUEST_BYTES => {
                response(Err("request exceeds registered byte limit".into()), false)
            }
            Ok(_) => evaluate_bytes(&bytes),
        }
    };
    if emit(&mut io::stdout().lock(), &value).is_err() || value["status"] == "FAIL" {
        std::process::exit(2);
    }
}
