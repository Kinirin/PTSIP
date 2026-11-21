use crate::source;
use serde_json::{json, Value};
use std::fs::{self, Metadata};
use std::io::ErrorKind;
use std::path::{Component, PathBuf};

pub const ALGORITHM: &str = "NORMALIZE_OWNER_RELATIVE_PROFILE_PATH_AND_REJECT_ROOT_ESCAPE";
pub const DEFAULT_PROFILE_PATH: &str = "ptsip.yaml";

fn is_alias(metadata: &Metadata) -> bool {
    if metadata.file_type().is_symlink() {
        return true;
    }
    #[cfg(windows)]
    {
        use std::os::windows::fs::MetadataExt;
        if metadata.file_attributes() & 0x400 != 0 {
            return true;
        }
    }
    false
}

pub fn directory(value: &str) -> Result<PathBuf, String> {
    let path = std::path::absolute(value).map_err(|error| error.to_string())?;
    let mut current = PathBuf::new();
    for component in path.components() {
        current.push(component.as_os_str());
        if matches!(component, Component::Prefix(_)) {
            continue;
        }
        let metadata = fs::symlink_metadata(&current).map_err(|error| error.to_string())?;
        if !metadata.is_dir() || is_alias(&metadata) {
            return Err("repository root must be a directory without source aliases".into());
        }
    }
    Ok(path)
}

pub fn normalize(value: Option<&str>) -> Result<String, String> {
    let value = value
        .unwrap_or(DEFAULT_PROFILE_PATH)
        .trim()
        .replace('\\', "/");
    if value.starts_with('/') || value.contains([':', '\0']) {
        return Err("Profile path must be repository-relative".into());
    }
    let mut parts = Vec::new();
    for part in value.split('/') {
        match part {
            "" | "." => continue,
            ".." => return Err("Profile path cannot escape the selected repository".into()),
            _ => parts.push(part),
        }
    }
    if parts.is_empty() {
        return Err("Profile path must name a file in the repository".into());
    }
    Ok(parts.join("/"))
}

pub fn resolve(input: &Value) -> Result<Value, String> {
    let root = directory(source::text(input, "root")?)?;
    let relative = normalize(input["path"].as_str())?;
    let mut path = root;
    let mut parts = relative.split('/').peekable();
    while let Some(part) = parts.next() {
        path.push(part);
        let metadata = match fs::symlink_metadata(&path) {
            Ok(metadata) => metadata,
            // Missing descendants can be named by a plan, but no existing
            // component may redirect the selection through an alias.
            Err(error) if error.kind() == ErrorKind::NotFound => continue,
            Err(error) => return Err(error.to_string()),
        };
        if is_alias(&metadata) {
            return Err("Profile path contains a source alias".into());
        }
        if parts.peek().is_some() && !metadata.is_dir() {
            return Err("Profile path ancestor is not a directory".into());
        }
        if parts.peek().is_none() && !metadata.is_file() {
            return Err("Profile path must name a regular file".into());
        }
    }
    Ok(json!({"normalized_path":relative,"path_on_disk":path}))
}
