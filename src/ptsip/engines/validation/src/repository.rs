use serde_json::{json, Value};

pub const ALGORITHM: &str =
    "COMPARE_EXPLICIT_REPOSITORY_SNAPSHOTS_AND_REJECT_INCOMPLETE_OBSERVATION";

pub fn compare(input: &Value) -> Result<Value, String> {
    let before = &input["before"];
    let after = &input["after"];
    let mut reasons = Vec::new();
    for (field, message) in [
        ("is_git", "repository git identity changed"),
        ("head", "HEAD changed during observation"),
        (
            "status_fingerprint",
            "working-tree/index/untracked-or-ignored state changed",
        ),
        (
            "tracked_content_fingerprint",
            "tracked file content changed",
        ),
        ("tracked_files", "observed file count changed"),
    ] {
        if before[field] != after[field] {
            reasons.push(message);
        }
    }
    if incomplete(before) || incomplete(after) {
        reasons.push("snapshot observation was incomplete");
    }
    let stable = reasons.is_empty();
    Ok(
        json!({"status":if stable {"STABLE"} else {"INVALIDATED"},"stable":stable,"reasons":reasons}),
    )
}

pub fn incomplete(snapshot: &Value) -> bool {
    snapshot["observation_errors"]
        .as_array()
        .is_none_or(|errors| !errors.is_empty())
        || (snapshot["is_git"] == true && snapshot["head"].is_null())
        || (snapshot["is_git"] == false && !snapshot["head"].is_null())
}
