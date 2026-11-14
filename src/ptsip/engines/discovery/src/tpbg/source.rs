use super::{Document, Error, Snapshot, Source};
use saphyr::{Scalar, Yaml, YamlLoader};
use saphyr_parser::{Event, Parser, SpannedEventReceiver};
use serde_json::{Map, Number, Value};
use std::fs::{self, File, Metadata, OpenOptions};
use std::io::Read;
use std::path::{Path, PathBuf};

const CONFIG: &str = ".ptsip/ptsip_config.json";
const PROFILES: &str = ".ptsip/profiles";
const MAX_SOURCE_BYTES: u64 = 2_097_152;
const MAX_YAML_DEPTH: usize = 128;

pub(super) fn read(root: &Path) -> Result<Snapshot, Error> {
    let metadata =
        fs::symlink_metadata(root).map_err(|error| failure("discovery", ".", error.to_string()))?;
    if !metadata.is_dir() || is_alias(&metadata) {
        return Err(failure(
            "discovery",
            ".",
            "expected a repository directory".into(),
        ));
    }
    let root =
        fs::canonicalize(root).map_err(|error| failure("discovery", ".", error.to_string()))?;
    let (config, bytes) =
        read_bytes(&root, CONFIG).map_err(|detail| failure("config", CONFIG, detail))?;
    let config_value =
        crate::source::parse(&bytes).map_err(|detail| failure("config", CONFIG, detail))?;
    if !config_value.is_object() {
        return Err(failure("config", CONFIG, "expected a JSON object".into()));
    }

    let mut profiles = Vec::new();
    for relative in candidates(&root)? {
        let (source, bytes) = read_bytes(&root, &relative)
            .map_err(|detail| failure("discovery", &relative, detail))?;
        let value = if is_json(Path::new(&relative)) {
            crate::source::parse(&bytes)
        } else {
            parse_yaml(&bytes)
        }
        .map_err(|detail| failure("normalization", &relative, detail))?;
        if !value.is_object() {
            return Err(failure(
                "normalization",
                &relative,
                "expected an object".into(),
            ));
        }
        profiles.push(Document { source, value });
    }
    Ok(Snapshot {
        config,
        config_value,
        profiles,
    })
}

fn candidates(root: &Path) -> Result<Vec<String>, Error> {
    let directory = within_root(root, PROFILES, true)
        .map_err(|detail| failure("discovery", PROFILES, detail))?;
    let entries = fs::read_dir(directory)
        .and_then(|entries| entries.collect::<Result<Vec<_>, _>>())
        .map_err(|error| failure("discovery", PROFILES, error.to_string()))?;
    let mut candidates = Vec::new();
    for entry in entries {
        let name = entry.file_name();
        let path = Path::new(&name);
        let supported = path
            .extension()
            .and_then(|extension| extension.to_str())
            .is_some_and(|extension| {
                ["json", "yaml", "yml"]
                    .iter()
                    .any(|kind| extension.eq_ignore_ascii_case(kind))
            });
        if !supported
            || name
                .to_str()
                .is_some_and(|name| name.eq_ignore_ascii_case("index.json"))
        {
            continue;
        }
        let name = name
            .to_str()
            .filter(|name| !name.contains('\\'))
            .ok_or_else(|| failure("discovery", PROFILES, "unsupported profile filename".into()))?;
        candidates.push(format!("{PROFILES}/{name}"));
    }
    candidates.sort();
    let mut files = Vec::new();
    for relative in candidates {
        let metadata = fs::symlink_metadata(root.join(&relative))
            .map_err(|error| failure("discovery", &relative, error.to_string()))?;
        if is_alias(&metadata) {
            return Err(failure(
                "discovery",
                &relative,
                "profile path is a link or reparse point".into(),
            ));
        }
        if metadata.is_file() {
            files.push(relative);
        }
    }
    Ok(files)
}

fn is_json(path: &Path) -> bool {
    path.extension()
        .and_then(|extension| extension.to_str())
        .is_some_and(|extension| extension.eq_ignore_ascii_case("json"))
}

fn failure(stage: &'static str, path: &str, detail: String) -> Error {
    Error {
        stage,
        path: path.into(),
        detail,
    }
}

fn within_root(root: &Path, relative: &str, directory: bool) -> Result<PathBuf, String> {
    let path = checked_path(&root.join(relative), directory)?;
    if !path.starts_with(root) {
        return Err("source escapes selected repository root".into());
    }
    Ok(path)
}

fn checked_path(path: &Path, directory: bool) -> Result<PathBuf, String> {
    let ancestors: Vec<_> = path.ancestors().collect();
    for ancestor in ancestors.into_iter().rev() {
        let metadata = fs::symlink_metadata(ancestor).map_err(|error| error.to_string())?;
        if is_alias(&metadata) {
            return Err("source path crosses a link or reparse point".into());
        }
        let expected_type = if directory || ancestor != path {
            metadata.is_dir()
        } else {
            metadata.is_file()
        };
        if !expected_type {
            return Err("source path has an unexpected file type".into());
        }
    }
    fs::canonicalize(path).map_err(|error| error.to_string())
}

#[cfg(windows)]
fn is_alias(metadata: &Metadata) -> bool {
    use std::os::windows::fs::MetadataExt;
    const FILE_ATTRIBUTE_REPARSE_POINT: u32 = 0x400;
    metadata.file_attributes() & FILE_ATTRIBUTE_REPARSE_POINT != 0
}

#[cfg(not(windows))]
fn is_alias(metadata: &Metadata) -> bool {
    metadata.file_type().is_symlink()
}

fn open_file(path: &Path) -> std::io::Result<File> {
    let mut options = OpenOptions::new();
    options.read(true);
    #[cfg(windows)]
    {
        use std::os::windows::fs::OpenOptionsExt;
        const FILE_FLAG_OPEN_REPARSE_POINT: u32 = 0x0020_0000;
        options.custom_flags(FILE_FLAG_OPEN_REPARSE_POINT);
    }
    options.open(path)
}

fn read_bytes(root: &Path, relative: &str) -> Result<(Source, Vec<u8>), String> {
    let path = within_root(root, relative, false)?;
    let mut file = open_file(&path).map_err(|error| error.to_string())?;
    let before = file.metadata().map_err(|error| error.to_string())?;
    if !before.is_file() || is_alias(&before) || before.len() > MAX_SOURCE_BYTES {
        return Err("invalid source file or source byte limit exceeded".into());
    }
    if within_root(root, relative, false)? != path {
        return Err("source path changed while opening".into());
    }
    let mut bytes = Vec::new();
    (&mut file)
        .take(MAX_SOURCE_BYTES + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 > MAX_SOURCE_BYTES {
        return Err("source byte limit exceeded".into());
    }
    let after = file.metadata().map_err(|error| error.to_string())?;
    if before.len() != bytes.len() as u64
        || after.len() != before.len()
        || after.modified().ok() != before.modified().ok()
        || within_root(root, relative, false)? != path
    {
        return Err("source changed while reading".into());
    }
    let source = Source {
        path: relative.into(),
        sha256: crate::source::hash(&bytes),
    };
    Ok((source, bytes))
}

fn parse_yaml(bytes: &[u8]) -> Result<Value, String> {
    let text = std::str::from_utf8(bytes).map_err(|error| error.to_string())?;
    let mut loader = YamlLoader::<Yaml>::default();
    let mut depth = 0;
    let mut document_started = false;
    for event in Parser::new_from_str(text) {
        let (mut event, span) = event.map_err(|error| error.to_string())?;
        match &mut event {
            Event::DocumentStart(_) => {
                if document_started {
                    return Err("expected one YAML document".into());
                }
                document_started = true;
            }
            Event::Alias(_) => return Err("YAML aliases are unsupported".into()),
            Event::Scalar(_, _, anchor, _) => *anchor = 0,
            Event::MappingStart(anchor, _) | Event::SequenceStart(anchor, _) => {
                if depth == MAX_YAML_DEPTH {
                    return Err("YAML nesting limit exceeded".into());
                }
                depth += 1;
                *anchor = 0;
            }
            Event::MappingEnd | Event::SequenceEnd => depth -= 1,
            _ => {}
        }
        loader.on_event(event, span);
        if let Some(error) = loader.error() {
            return Err(error.to_string());
        }
    }
    let document = loader
        .into_documents()
        .pop()
        .ok_or("expected one YAML document")?;
    yaml_value(document)
}

fn yaml_value(node: Yaml<'_>) -> Result<Value, String> {
    match node {
        Yaml::Value(Scalar::Null) => Ok(Value::Null),
        Yaml::Value(Scalar::Boolean(value)) => Ok(Value::Bool(value)),
        Yaml::Value(Scalar::Integer(value)) => Ok(Value::Number(value.into())),
        Yaml::Value(Scalar::FloatingPoint(value)) => Number::from_f64(value.into_inner())
            .map(Value::Number)
            .ok_or_else(|| "non-finite YAML number".into()),
        Yaml::Value(Scalar::String(value)) => Ok(Value::String(value.into_owned())),
        Yaml::Sequence(items) => items
            .into_iter()
            .map(yaml_value)
            .collect::<Result<Vec<_>, _>>()
            .map(Value::Array),
        Yaml::Mapping(entries) => {
            let mut object = Map::new();
            for (key, value) in entries {
                let Yaml::Value(Scalar::String(key)) = key else {
                    return Err("YAML mapping keys must be strings".into());
                };
                object.insert(key.into_owned(), yaml_value(value)?);
            }
            Ok(Value::Object(object))
        }
        _ => Err("YAML node cannot be represented as JSON".into()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};
    use std::time::{SystemTime, UNIX_EPOCH};

    struct Repository(PathBuf);

    impl Repository {
        fn new() -> Self {
            static NEXT_ID: AtomicU64 = AtomicU64::new(0);
            let time = SystemTime::now().duration_since(UNIX_EPOCH).unwrap();
            let path = std::env::temp_dir().join(format!(
                "ptsip-tpbg-{}-{}-{}",
                std::process::id(),
                time.as_nanos(),
                NEXT_ID.fetch_add(1, Ordering::Relaxed)
            ));
            fs::create_dir(&path).unwrap();
            let repository = Self(path);
            fs::create_dir_all(repository.0.join(PROFILES)).unwrap();
            repository.write(CONFIG, "{}\n");
            repository
        }

        fn write(&self, relative: &str, data: &str) {
            fs::write(self.0.join(relative), data).unwrap();
        }
    }

    impl Drop for Repository {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }

    #[test]
    fn reads_sorted_direct_candidates_and_ignores_legacy_indexes() {
        let repository = Repository::new();
        repository.write(".ptsip/index.json", "invalid JSON");
        repository.write(".ptsip/profiles/INDEX.JSON", "invalid JSON");
        repository.write(".ptsip/profiles/z.YAML", "ptsip:\n  id: profile\n");
        let json = "{\"other\":true}\n";
        repository.write(".ptsip/profiles/a.json", json);
        repository.write(".ptsip/profiles/ignore.txt", "invalid text");
        fs::create_dir(repository.0.join(".ptsip/profiles/folder.yaml")).unwrap();
        repository.write(".ptsip/profiles/folder.yaml/ignored.yml", "invalid: [");
        let snapshot =
            read(&repository.0).unwrap_or_else(|error| panic!("{}: {}", error.path, error.detail));
        assert_eq!(snapshot.config.path, CONFIG);
        assert_eq!(snapshot.config.sha256, crate::source::hash(b"{}\n"));
        assert_eq!(snapshot.profiles.len(), 2);
        let first = &snapshot.profiles[0];
        assert_eq!(first.source.path, ".ptsip/profiles/a.json");
        assert_eq!(first.source.sha256, crate::source::hash(json.as_bytes()));
        assert_eq!(first.value["other"], true);
        assert_eq!(snapshot.profiles[1].source.path, ".ptsip/profiles/z.YAML");
        assert_eq!(snapshot.profiles[1].value["ptsip"]["id"], "profile");
    }

    #[test]
    fn rejects_config_that_is_not_an_unambiguous_json_object() {
        let repository = Repository::new();
        for invalid in ["[]", "{\"key\":1,\"key\":2}"] {
            repository.write(CONFIG, invalid);
            let error = read(&repository.0).err().expect("invalid config accepted");
            assert_eq!(error.stage, "config");
            assert_eq!(error.path, CONFIG);
        }
    }

    #[test]
    fn rejects_malformed_candidate_without_returning_other_profiles() {
        let repository = Repository::new();
        repository.write(".ptsip/profiles/valid.json", "{}");
        let path = ".ptsip/profiles/wrong.yml";
        for invalid in [
            "outer: {key: one, 'key': two}\n",
            "outer: {1: value}\n",
            "outer: {true: value}\n",
            "---\nfirst: value\n---\nsecond: value\n",
            "first: &anchor value\nsecond: *anchor\n",
            "outer: !custom value\n",
            "[1, 2]\n",
        ] {
            repository.write(path, invalid);
            let error = read(&repository.0).err().expect("invalid YAML accepted");
            assert_eq!(error.stage, "normalization");
            assert_eq!(error.path, path);
        }
        let nested = "[".repeat(MAX_YAML_DEPTH + 1) + &"]".repeat(MAX_YAML_DEPTH + 1);
        repository.write(path, &nested);
        let error = read(&repository.0)
            .err()
            .expect("excessive YAML nesting accepted");
        assert_eq!(error.stage, "normalization");
        assert_eq!(error.detail, "YAML nesting limit exceeded");
        repository.write(path, "outer: {'1': value, 'true': value}\n");
        assert!(read(&repository.0).is_ok());
        repository.write(".ptsip/profiles/valid.json", "{\"key\":1,\"key\":2}");
        let error = read(&repository.0)
            .err()
            .expect("duplicate JSON key accepted");
        assert_eq!(error.stage, "normalization");
        assert_eq!(error.path, ".ptsip/profiles/valid.json");
    }

    #[cfg(unix)]
    #[test]
    fn rejects_symlink_candidates_and_repository_roots() {
        use std::os::unix::fs::symlink;
        let repository = Repository::new();
        let other = Repository::new();
        let candidate = ".ptsip/profiles/linked.json";
        symlink(other.0.join(CONFIG), repository.0.join(candidate)).unwrap();
        let error = read(&repository.0)
            .err()
            .expect("linked candidate accepted");
        assert_eq!(error.stage, "discovery");
        assert_eq!(error.path, candidate);
        let alias = other.0.join("alias");
        symlink(&repository.0, &alias).unwrap();
        let error = read(&alias).err().expect("linked repository root accepted");
        assert_eq!(error.stage, "discovery");
        assert_eq!(error.path, ".");
    }
}
