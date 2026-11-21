use crate::source::{self, Root};
use serde_json::{json, Value};
use std::collections::BTreeSet;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::process::{Command, Output, Stdio};

pub const ALGORITHM: &str =
    "DISCOVER_EXPLICIT_REPOSITORY_IDENTITY_REMOTE_AND_SORTED_SOURCE_INVENTORY";

pub(crate) fn alias(metadata: &fs::Metadata) -> bool {
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

fn git(root: &Path, args: &[&str], disabled_filters: &[String]) -> io::Result<Output> {
    let mut command = Command::new("git");
    for (key, _) in std::env::vars_os() {
        if key
            .to_string_lossy()
            .to_ascii_uppercase()
            .starts_with("GIT_")
        {
            command.env_remove(key);
        }
    }
    command
        .env("LC_ALL", "C")
        .env("GIT_TERMINAL_PROMPT", "0")
        .env("GIT_OPTIONAL_LOCKS", "0")
        .env("GIT_NO_LAZY_FETCH", "1")
        .arg("--no-optional-locks")
        .args([
            "-c",
            "core.fsmonitor=false",
            "-c",
            "core.untrackedCache=false",
        ])
        .arg("-C")
        .arg(root);
    for setting in disabled_filters {
        command.arg("-c").arg(setting);
    }
    command.args(args).stdin(Stdio::null()).output()
}

// Git status can otherwise invoke a configured clean filter while reading a
// file. Observe the worktree with those commands disabled.
fn filter_overrides(root: &Path) -> Result<Vec<String>, String> {
    let output = git(
        root,
        &["config", "--name-only", "-z", "--get-regexp", "^filter\\."],
        &[],
    )
    .map_err(|_| "unable to inspect Git filter settings")?;
    if output.status.code() == Some(1) {
        return Ok(Vec::new());
    }
    if !output.status.success() {
        return Err("unable to inspect Git filter settings".into());
    }
    let keys =
        std::str::from_utf8(&output.stdout).map_err(|_| "invalid Git filter key encoding")?;
    let mut filters = BTreeSet::new();
    for key in keys.split('\0').filter(|key| !key.is_empty()) {
        let (name, _) = key.rsplit_once('.').ok_or("invalid Git filter key")?;
        filters.insert(name);
    }
    Ok(filters
        .into_iter()
        .flat_map(|name| {
            [
                format!("{name}.clean="),
                format!("{name}.process="),
                format!("{name}.required=false"),
            ]
        })
        .collect())
}

fn trimmed(bytes: &[u8]) -> Option<&str> {
    std::str::from_utf8(bytes)
        .ok()
        .map(str::trim)
        .filter(|text| !text.is_empty())
}

fn files_on_disk(root: &Path, errors: &mut Vec<String>) -> Vec<String> {
    let mut files = Vec::new();
    let mut pending = vec![root.to_owned()];
    while let Some(directory) = pending.pop() {
        let entries = match fs::read_dir(&directory) {
            Ok(entries) => entries,
            Err(error) => {
                errors.push(format!("{}: {error}", directory.display()));
                continue;
            }
        };
        for entry in entries {
            let entry = match entry {
                Ok(entry) => entry,
                Err(error) => {
                    errors.push(format!("{}: {error}", directory.display()));
                    continue;
                }
            };
            if entry.file_name().eq_ignore_ascii_case(".git") {
                continue;
            }
            let path = entry.path();
            match fs::symlink_metadata(&path) {
                Ok(metadata) if metadata.is_dir() && !alias(&metadata) => pending.push(path),
                Ok(metadata) if metadata.is_file() || alias(&metadata) => {
                    match path.strip_prefix(root).ok().and_then(Path::to_str) {
                        Some(relative) => {
                            files.push(relative.replace(std::path::MAIN_SEPARATOR, "/"))
                        }
                        None => errors.push("source path is not representable in UTF-8".into()),
                    }
                }
                Ok(_) => errors.push(format!("{}: unsupported source entry", path.display())),
                Err(error) => errors.push(format!("{}: {error}", path.display())),
            }
        }
    }
    files.sort();
    files
}

fn tracked(root: &Path, errors: &mut Vec<String>) -> Vec<String> {
    let Ok(output) = git(root, &["ls-files", "-z"], &[]) else {
        errors.push("git ls-files failed".into());
        return Vec::new();
    };
    if !output.status.success() {
        errors.push("git ls-files failed".into());
        return Vec::new();
    }
    let mut files = BTreeSet::new();
    for path in output
        .stdout
        .split(|byte| *byte == 0)
        .filter(|path| !path.is_empty())
    {
        match std::str::from_utf8(path) {
            Ok(path) if files.insert(path.to_owned()) => {}
            Ok(path) => errors.push(format!("{path}: multiple Git index entries")),
            Err(_) => errors.push("Git index contains a path outside UTF-8".into()),
        }
    }
    files.into_iter().collect()
}

fn origin(url: &str) -> Value {
    let (host, path) = if let Some((_, rest)) = url.split_once("://") {
        let (authority, tail) = rest.split_once('/').unwrap_or((rest, ""));
        let address = authority.rsplit('@').next().unwrap_or("");
        let host = if let Some(ip) = address.strip_prefix('[') {
            ip.split_once(']').map(|(host, _)| host).unwrap_or("")
        } else {
            address.split(':').next().unwrap_or("")
        };
        (host, tail.split(['?', '#']).next().unwrap_or(""))
    } else if let Some((address, path)) = url.split_once(':') {
        let host = address.rsplit('@').next().unwrap_or("");
        if host.len() == 1 || host.contains(['/', '\\']) || host.chars().any(char::is_whitespace) {
            ("", "")
        } else {
            (host, path)
        }
    } else {
        ("", "")
    };
    let path = path.trim_matches('/');
    let path = if path.to_ascii_lowercase().ends_with(".git") {
        &path[..path.len() - 4]
    } else {
        path
    };
    let github = host.eq_ignore_ascii_case("github.com")
        && path.split('/').count() == 2
        && path.split('/').all(|part| !part.is_empty());
    json!({"name":"origin", "url":url, "host":(!host.is_empty()).then(||host.to_ascii_lowercase()),
           "provider":github.then_some("github"), "repository":github.then_some(path)})
}

fn git_text(root: &Path, args: &[&str]) -> Option<String> {
    let output = git(root, args, &[]).ok()?;
    if !output.status.success() {
        return None;
    }
    trimmed(&output.stdout).map(str::to_owned)
}

fn requested_directory(value: &str) -> Result<PathBuf, String> {
    let mut path = std::path::absolute(value).map_err(|error| error.to_string())?;
    let metadata = fs::symlink_metadata(&path).map_err(|error| error.to_string())?;
    if alias(&metadata) {
        return Err("requested repository path is a source alias".into());
    }
    if metadata.is_file() {
        path.pop();
    } else if !metadata.is_dir() {
        return Err("requested repository path must be a file or directory".into());
    }
    Root::new(path.to_str().ok_or("repository path is not UTF-8")?)?;
    Ok(path)
}

pub fn inspect(input: &Value) -> Result<Value, String> {
    let requested = requested_directory(source::text(input, "path")?)?;
    let probe = git(&requested, &["rev-parse", "--show-toplevel"], &[])
        .map_err(|_| "Git executable unavailable for repository discovery")?;
    let mut errors = Vec::new();
    if !probe.status.success() {
        if !String::from_utf8_lossy(&probe.stderr).contains("not a git repository") {
            return Err("unable to determine Git worktree root".into());
        }
        let files = files_on_disk(&requested, &mut errors);
        errors.sort();
        return Ok(
            json!({"requested_path":requested,"root":requested,"is_git":false,
            "commit":null,"branch":null,"dirty":null,"remote":null,"file_mode":"filesystem",
            "files":files,"observation_errors":errors}),
        );
    }
    let root = requested_directory(trimmed(&probe.stdout).ok_or("invalid Git worktree path")?)?;
    let commit = git_text(&root, &["rev-parse", "--verify", "HEAD"]);
    if commit.is_none() {
        errors.push("git rev-parse HEAD failed".into());
    }
    let branch = git_text(&root, &["branch", "--show-current"]);
    let filters = filter_overrides(&root)?;
    let dirty = git(
        &root,
        &["status", "--porcelain=v1", "-z", "--ignore-submodules=all"],
        &filters,
    )
    .ok()
    .filter(|output| output.status.success())
    .map(|output| !output.stdout.is_empty());
    if dirty.is_none() {
        errors.push("git status failed".into());
    }
    let remote = git_text(&root, &["remote", "get-url", "origin"]).map(|url| origin(&url));
    // File inventory is relative to the caller-selected directory, while the
    // repository identity describes the containing Git worktree.
    let files = tracked(&requested, &mut errors);
    errors.sort();
    Ok(
        json!({"requested_path":requested,"root":root,"is_git":true,"commit":commit,
        "branch":branch,"dirty":dirty,"remote":remote,"file_mode":"git-tracked",
        "files":files,"observation_errors":errors}),
    )
}
