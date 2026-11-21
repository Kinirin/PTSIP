use crate::source;
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::BTreeSet;
use std::fs::{self, File, Metadata};
use std::io::{self, Read};
use std::path::{Component, Path, PathBuf};
use std::process::{Command, Output, Stdio};
use std::time::{SystemTime, UNIX_EPOCH};

pub const ALGORITHM: &str =
    "OBSERVE_EXPLICIT_REPOSITORY_GIT_STATUS_AND_TRACKED_CONTENT_WITHOUT_MUTATION";

fn alias(metadata: &Metadata) -> bool {
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

fn directory(value: &str) -> Result<PathBuf, String> {
    let path = std::path::absolute(value).map_err(|error| error.to_string())?;
    let mut current = PathBuf::new();
    for component in path.components() {
        current.push(component.as_os_str());
        if matches!(component, Component::Prefix(_)) {
            continue;
        }
        let metadata = fs::symlink_metadata(&current).map_err(|error| error.to_string())?;
        if !metadata.is_dir() || alias(&metadata) {
            return Err("repository root must be a directory without source aliases".into());
        }
    }
    Ok(path)
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

fn hash_entry(root: &Path, relative: &str, digest: &mut Sha256) -> Result<(), String> {
    let mut path = root.to_owned();
    let mut parts = Path::new(relative).components().peekable();
    while let Some(part) = parts.next() {
        let Component::Normal(name) = part else {
            return Err("invalid observation path".into());
        };
        path.push(name);
        let metadata = fs::symlink_metadata(&path).map_err(|error| error.to_string())?;
        if parts.peek().is_some() {
            if !metadata.is_dir() || alias(&metadata) {
                return Err("source ancestor is not an ordinary directory".into());
            }
            continue;
        }
        if metadata.file_type().is_symlink() {
            let target = fs::read_link(path).map_err(|error| error.to_string())?;
            let target = target.to_str().ok_or("link target is not UTF-8")?;
            digest.update(b"symlink\0");
            digest.update(target.as_bytes());
            return Ok(());
        }
        if !metadata.is_file() || alias(&metadata) {
            return Err("source is not an ordinary file".into());
        }
        let mut file = File::open(&path).map_err(|error| error.to_string())?;
        let mut buffer = [0; 64 * 1024];
        loop {
            let count = file.read(&mut buffer).map_err(|error| error.to_string())?;
            if count == 0 {
                break;
            }
            digest.update(&buffer[..count]);
        }
        let after = fs::symlink_metadata(&path).map_err(|error| error.to_string())?;
        if alias(&after)
            || after.len() != metadata.len()
            || after.modified().ok() != metadata.modified().ok()
        {
            return Err("source changed while observed".into());
        }
        return Ok(());
    }
    Err("empty observation path".into())
}

fn captured_at() -> Result<String, String> {
    let seconds = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|error| error.to_string())?
        .as_secs();
    let leap = |year: u64| year % 4 == 0 && (year % 100 != 0 || year % 400 == 0);
    let mut days = seconds / 86_400;
    let mut year = 1970;
    loop {
        let length = if leap(year) { 366 } else { 365 };
        if days < length {
            break;
        }
        days -= length;
        year += 1;
    }
    let months = [
        31,
        if leap(year) { 29 } else { 28 },
        31,
        30,
        31,
        30,
        31,
        31,
        30,
        31,
        30,
        31,
    ];
    let mut month = 1;
    for length in months {
        if days < length {
            break;
        }
        days -= length;
        month += 1;
    }
    let (day, hour, minute, second) = (
        days + 1,
        seconds / 3600 % 24,
        seconds / 60 % 60,
        seconds % 60,
    );
    Ok(format!(
        "{year:04}-{month:02}-{day:02}T{hour:02}:{minute:02}:{second:02}Z"
    ))
}

pub fn snapshot(input: &Value) -> Result<Value, String> {
    let root = directory(source::text(input, "root")?)?;
    let probe = git(&root, &["rev-parse", "--is-inside-work-tree"], &[])
        .map_err(|_| "Git executable unavailable for repository observation")?;
    let is_git = probe.status.success() && trimmed(&probe.stdout) == Some("true");
    let mut errors = Vec::new();
    if !probe.status.success()
        && !String::from_utf8_lossy(&probe.stderr).contains("not a git repository")
    {
        return Err("unable to determine Git worktree identity".into());
    }
    let (head, status, files) = if is_git {
        let head = git(&root, &["rev-parse", "--verify", "HEAD"], &[]).ok();
        let head = head
            .as_ref()
            .filter(|o| o.status.success())
            .and_then(|o| trimmed(&o.stdout))
            .map(str::to_owned);
        if head.is_none() {
            errors.push("git rev-parse HEAD failed".into());
        }
        let filters = filter_overrides(&root)?;
        let status = git(
            &root,
            &[
                "status",
                "--porcelain=v1",
                "-z",
                "--untracked-files=all",
                "--ignored=matching",
                "--ignore-submodules=all",
            ],
            &filters,
        )
        .ok();
        let status = match status.filter(|output| output.status.success()) {
            Some(output) => output.stdout,
            None => {
                errors.push("git status failed".into());
                b"<git-status-failed>".to_vec()
            }
        };
        (head, status, tracked(&root, &mut errors))
    } else {
        (
            None,
            b"<non-git>".to_vec(),
            files_on_disk(&root, &mut errors),
        )
    };
    let mut content = Sha256::new();
    for path in &files {
        content.update(path.as_bytes());
        content.update(b"\0");
        if let Err(error) = hash_entry(&root, path, &mut content) {
            errors.push(format!("{path}: {error}"));
            content.update(b"<unreadable>");
        }
        content.update(b"\0");
    }
    errors.sort();
    Ok(json!({
        "captured_at": captured_at()?, "is_git": is_git, "head": head,
        "status_fingerprint": source::hash(&status),
        "tracked_content_fingerprint": format!("{:x}",content.finalize()),
        "tracked_files": files.len(), "observation_errors": errors,
    }))
}
