use crate::source::{self, Root};
use serde_json::Value;
use std::time::{SystemTime, UNIX_EPOCH};

pub struct Consumed {
    profile_path: String,
    profile_hash: String,
    authority_path: String,
    authority_hash: String,
    expires: u64,
}

fn now() -> Result<u64, String> {
    Ok(SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|e| e.to_string())?
        .as_millis() as u64)
}

impl Consumed {
    pub fn check(
        root: &mut Root,
        input: &Value,
        manifest: &Value,
        contract: &Value,
        operation: &str,
        invocation_id: &str,
        request_hash: &Value,
        artifact_hash: &Value,
    ) -> Result<Self, String> {
        let profile = root.file(&input["profile_ref"])?;
        source::schema("owner_admission", &profile)?;
        let authority = root.bound(&input["authority_ref"])?;
        source::schema("authority_consumption", &authority)?;
        if authority["profile_ref"] != input["profile_ref"]
            || authority["repository_id"] != profile["repository_id"]
            || authority["owner_id"] != profile["owner_id"]
            || authority["actor_id"] != input["actor_id"]
            || authority["engine_id"] != manifest["engine_id"]
            || authority["capability_id"] != contract["contract_id"]
            || authority["contract_version"] != contract["version"]
            || authority["operation_id"] != operation
            || authority["invocation_id"] != invocation_id
            || &authority["request_sha256"] != request_hash
            || &authority["artifact_sha256"] != artifact_hash
        {
            return Err("authority is not bound to this exact current native invocation".into());
        }
        if !profile["actors"]
            .as_array()
            .ok_or("admitted actors required")?
            .contains(&input["actor_id"])
        {
            return Err("actor has no admitted local authority".into());
        }
        let subjects = profile["operation_subjects"]
            .as_array()
            .ok_or("registered operation subjects required")?;
        if subjects
            .iter()
            .filter(|s| {
                s["engine_id"] == manifest["engine_id"]
                    && s["capability_id"] == contract["contract_id"]
                    && s["contract_version"] == contract["version"]
                    && s["operation_id"] == operation
            })
            .count()
            != 1
        {
            return Err("operation subject mandate missing or ambiguous".into());
        }
        let starts = authority["not_before_unix_ms"]
            .as_u64()
            .ok_or("authority start required")?;
        let expires = authority["expires_unix_ms"]
            .as_u64()
            .ok_or("authority expiration required")?;
        let timestamp = now()?;
        if starts > timestamp || timestamp >= expires || starts >= expires {
            return Err("authority is not currently valid".into());
        }
        Ok(Self {
            profile_path: source::text(&input["profile_ref"], "path")?.into(),
            profile_hash: source::text(&input["profile_ref"], "sha256")?.into(),
            authority_path: source::text(&input["authority_ref"], "path")?.into(),
            authority_hash: source::text(&input["authority_ref"], "sha256")?.into(),
            expires,
        })
    }
    pub fn current(&self, root: &Root) -> bool {
        let check = || -> Result<bool, String> {
            Ok(now()? < self.expires
                && source::hash(&root.bytes(&self.profile_path, 2_097_152)?) == self.profile_hash
                && source::hash(&root.bytes(&self.authority_path, 2_097_152)?)
                    == self.authority_hash)
        };
        check().unwrap_or(false)
    }
}
