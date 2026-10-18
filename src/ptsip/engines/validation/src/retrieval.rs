//! Explicit, bounded, read-only schema acquisition. No ambient repository discovery.
use serde::Deserialize;
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::BTreeSet;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};
use url::Url;

const MAX_BYTES: usize = 1024 * 1024;
#[derive(Default, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Resolution {
    pub base_uri: Option<String>,
    pub directories: Vec<PathBuf>,
    pub allow_network: bool,
    pub allowed_origins: Vec<String>,
}
pub struct Document {
    pub uri: String,
    pub schema: Value,
    pub evidence: Value,
}
pub struct Loader {
    roots: Vec<PathBuf>,
    origins: BTreeSet<String>,
    bytes: usize,
    documents: usize,
    deadline: Instant,
}
impl Loader {
    pub fn new(config: &Resolution) -> Result<Self, String> {
        if config.directories.len() > 8 || config.allowed_origins.len() > 16 {
            return Err("SCHEMA_RESOURCE_LIMIT: at most 8 directories and 16 origins".into());
        }
        let mut roots = Vec::new();
        for root in &config.directories {
            let root = root
                .canonicalize()
                .map_err(|e| format!("SCHEMA_DIRECTORY_INVALID: {e}"))?;
            if !root.is_dir() {
                return Err("SCHEMA_DIRECTORY_INVALID: directory required".into());
            }
            if !roots.contains(&root) {
                roots.push(root);
            }
        }
        let mut origins = BTreeSet::new();
        for origin in &config.allowed_origins {
            let uri = Url::parse(origin).map_err(|e| format!("SCHEMA_ORIGIN_INVALID: {e}"))?;
            if !matches!(uri.scheme(), "http" | "https")
                || uri.host_str().is_none()
                || !uri.username().is_empty()
                || uri.password().is_some()
                || uri.path() != "/"
                || uri.query().is_some()
                || uri.fragment().is_some()
            {
                return Err(
                    "SCHEMA_ORIGIN_INVALID: use an exact HTTP(S) origin, including its port".into(),
                );
            }
            origins.insert(uri.origin().ascii_serialization());
        }
        if config.allow_network && origins.is_empty() {
            return Err("SCHEMA_NETWORK_DENIED: allow_network requires allowed_origins".into());
        }
        if !config.allow_network {
            origins.clear();
        }
        Ok(Self {
            roots,
            origins,
            bytes: 0,
            documents: 0,
            deadline: Instant::now() + Duration::from_secs(20),
        })
    }
    fn document(&mut self, uri: String, bytes: Vec<u8>, source: &str) -> Result<Document, String> {
        self.bytes += bytes.len();
        self.documents += 1;
        if bytes.len() > MAX_BYTES || self.bytes > 8 * MAX_BYTES || self.documents > 128 {
            return Err(
                "SCHEMA_RESOURCE_LIMIT: acquisition byte or document budget exceeded".into(),
            );
        }
        let schema =
            serde_json::from_slice(&bytes).map_err(|e| format!("INVALID_SCHEMA_RESOURCE: {e}"))?;
        let evidence =
            json!({"uri":uri,"sha256":format!("{:x}",Sha256::digest(&bytes)),"source":source});
        Ok(Document {
            uri,
            schema,
            evidence,
        })
    }
    fn file(&mut self, path: &Path) -> Result<Document, String> {
        let path = path
            .canonicalize()
            .map_err(|e| format!("SCHEMA_FILE_INVALID: {e}"))?;
        if !self.roots.iter().any(|root| path.starts_with(root)) || !path.is_file() {
            return Err("SCHEMA_FILE_DENIED: file is outside configured directories".into());
        }
        let uri = Url::from_file_path(&path)
            .map_err(|_| "SCHEMA_FILE_INVALID: absolute file URI required")?
            .to_string();
        let mut bytes = Vec::new();
        std::fs::File::open(path)
            .map_err(|e| format!("SCHEMA_FILE_INVALID: {e}"))?
            .take((MAX_BYTES + 1) as u64)
            .read_to_end(&mut bytes)
            .map_err(|e| e.to_string())?;
        self.document(uri, bytes, "FILE_BYTES")
    }
    pub fn discover(&mut self) -> Result<Vec<Document>, String> {
        let mut directories = self.roots.clone();
        let mut visited = BTreeSet::new();
        let mut files = BTreeSet::new();
        let mut entries = 0;
        while let Some(directory) = directories.pop() {
            let directory = directory.canonicalize().map_err(|e| e.to_string())?;
            if !self.roots.iter().any(|r| directory.starts_with(r)) {
                return Err("SCHEMA_FILE_DENIED: directory escapes configured roots".into());
            }
            if !visited.insert(directory.clone()) {
                continue;
            }
            for entry in std::fs::read_dir(directory).map_err(|e| e.to_string())? {
                entries += 1;
                if entries > 4096 {
                    return Err("SCHEMA_RESOURCE_LIMIT: directory scan exceeds 4096 entries".into());
                }
                let entry = entry.map_err(|e| e.to_string())?;
                let kind = entry.file_type().map_err(|e| e.to_string())?;
                if kind.is_symlink() {
                    continue;
                }
                if kind.is_dir() {
                    directories.push(entry.path());
                } else if kind.is_file()
                    && entry
                        .path()
                        .extension()
                        .is_some_and(|x| x.eq_ignore_ascii_case("json"))
                {
                    files.insert(entry.path().canonicalize().map_err(|e| e.to_string())?);
                    if files.len() > 128 {
                        return Err("SCHEMA_RESOURCE_LIMIT: too many discovered JSON files".into());
                    }
                }
            }
        }
        files.iter().map(|path| self.file(path)).collect()
    }
    pub fn retrieve(&mut self, uri: &str) -> Result<Document, String> {
        let url = Url::parse(uri).map_err(|e| format!("SCHEMA_RESOURCE_NOT_FOUND: {e}"))?;
        if url.scheme() == "file" {
            let path = url
                .to_file_path()
                .map_err(|_| "SCHEMA_FILE_DENIED: invalid local file URI")?;
            return self.file(&path);
        }
        if !matches!(url.scheme(), "http" | "https") {
            return Err(format!("SCHEMA_RESOURCE_NOT_FOUND: {uri}"));
        }
        if self.origins.is_empty() {
            return Err("SCHEMA_RESOURCE_NOT_FOUND: network retrieval is disabled".into());
        }
        if !self.origins.contains(&url.origin().ascii_serialization())
            || !url.username().is_empty()
            || url.password().is_some()
        {
            return Err(
                "SCHEMA_NETWORK_DENIED: referenced origin is not explicitly allowed".into(),
            );
        }
        let remaining = self
            .deadline
            .checked_duration_since(Instant::now())
            .ok_or("SCHEMA_RESOURCE_LIMIT: acquisition time budget exceeded")?;
        let agent = ureq::Agent::config_builder()
            .proxy(None)
            .max_redirects(0)
            .timeout_global(Some(remaining.min(Duration::from_secs(5))))
            .build()
            .new_agent();
        let mut response = agent
            .get(url.as_str())
            .header("Accept", "application/schema+json, application/json")
            .call()
            .map_err(|e| format!("SCHEMA_NETWORK_ERROR: {e}"))?;
        if response.status().as_u16() != 200 {
            return Err("SCHEMA_NETWORK_ERROR: expected HTTP 200; redirects are disabled".into());
        }
        let bytes = response
            .body_mut()
            .with_config()
            .limit((MAX_BYTES + 1) as u64)
            .read_to_vec()
            .map_err(|e| format!("SCHEMA_NETWORK_ERROR: {e}"))?;
        self.document(uri.to_owned(), bytes, "NETWORK_BYTES")
    }
}
