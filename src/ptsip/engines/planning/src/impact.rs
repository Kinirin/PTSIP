use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet, VecDeque};
use std::fs;
use std::path::{Component, Path};

fn read(root: &Path, reference: &str) -> Result<(Value, String), String> {
    let relative = Path::new(reference);
    if relative.is_absolute()
        || reference.contains('\\')
        || relative
            .components()
            .any(|c| !matches!(c, Component::Normal(_)))
    {
        return Err("invalid graph source reference".into());
    }
    let mut path = root.to_owned();
    for c in relative.components() {
        path.push(c);
        if fs::symlink_metadata(&path)
            .map_err(|e| e.to_string())?
            .file_type()
            .is_symlink()
        {
            return Err("unregistered graph source alias".into());
        }
    }
    let path = fs::canonicalize(path).map_err(|e| e.to_string())?;
    if !path.starts_with(root) {
        return Err("graph source escaped owner root".into());
    }
    let bytes = fs::read(path).map_err(|e| e.to_string())?;
    if bytes.len() > 2_097_152 {
        return Err("graph byte limit exceeded".into());
    }
    Ok((
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?,
        format!("{:x}", Sha256::digest(&bytes)),
    ))
}
fn text<'a>(v: &'a Value, k: &str) -> Result<&'a str, String> {
    v[k].as_str()
        .filter(|s| !s.is_empty())
        .ok_or_else(|| format!("missing graph field {k}"))
}

pub fn analyze(input: &Value) -> Result<Value, String> {
    let root = fs::canonicalize(text(input, "root")?).map_err(|e| e.to_string())?;
    let (graph, graph_hash) = read(&root, text(input, "graph_ref")?)?;
    if input["expected_graph_sha256"] != graph_hash {
        return Err("graph revision changed; current analysis required".into());
    }
    let nodes = graph["nodes"].as_array().ok_or("graph nodes required")?;
    let edges = graph["edges"].as_array().ok_or("graph edges required")?;
    let mut records = BTreeMap::new();
    let mut current = BTreeMap::new();
    let mut invalid = BTreeSet::new();
    for n in nodes {
        let id = text(n, "id")?.to_owned();
        if records.insert(id.clone(), n).is_some() {
            return Err("ambiguous graph node".into());
        }
        match read(&root, text(n, "path")?) {
            Ok((doc, hash)) => {
                let identity = doc
                    .get("engine_id")
                    .or_else(|| doc.get("contract_id"))
                    .or_else(|| doc.get("id"));
                let version = doc
                    .get("implementation_version")
                    .or_else(|| doc.get("version"));
                if identity != n.get("id") || version != n.get("version") || n["sha256"] != hash {
                    invalid.insert(id.clone());
                }
                current.insert(id, (doc, hash));
            }
            Err(_) => {
                invalid.insert(id);
            }
        }
    }
    let changed = text(input, "changed_id")?;
    if !records.contains_key(changed) {
        return Err("changed dependency identity unregistered".into());
    }
    let mut adjacency: BTreeMap<String, BTreeSet<String>> = records
        .keys()
        .map(|id| (id.clone(), BTreeSet::new()))
        .collect();
    let mut edge_keys = BTreeSet::new();
    for edge in edges {
        let p = text(edge, "provider")?;
        let c = text(edge, "consumer")?;
        if !records.contains_key(p) || !records.contains_key(c) {
            return Err("unknown dependency boundary".into());
        }
        if !matches!(
            text(edge, "relation")?,
            "DEPENDENCY" | "COMPATIBILITY" | "COMPOSITION"
        ) {
            return Err("unregistered dependency relation".into());
        }
        if !edge_keys.insert((
            p.to_owned(),
            c.to_owned(),
            text(edge, "relation")?.to_owned(),
        )) {
            return Err("ambiguous dependency edge".into());
        }
        adjacency
            .get_mut(p)
            .ok_or("unknown provider")?
            .insert(c.to_owned());
    }
    // Validate a closed DAG, independent of lexical ordering of declarations.
    let mut indegree: BTreeMap<String, usize> = records.keys().map(|id| (id.clone(), 0)).collect();
    for consumers in adjacency.values() {
        for c in consumers {
            *indegree.get_mut(c).ok_or("unknown consumer")? += 1;
        }
    }
    let mut ready: VecDeque<_> = indegree
        .iter()
        .filter(|(_, v)| **v == 0)
        .map(|(id, _)| id.clone())
        .collect();
    let mut visited = 0;
    while let Some(id) = ready.pop_front() {
        visited += 1;
        for c in &adjacency[&id] {
            let n = indegree.get_mut(c).ok_or("unknown consumer")?;
            *n -= 1;
            if *n == 0 {
                ready.push_back(c.clone());
            }
        }
    }
    if visited != records.len() {
        return Err("dependency cycle".into());
    }
    let mut affected = BTreeSet::new();
    let mut uncertain = false;
    let mut queue = VecDeque::from([changed.to_owned()]);
    let mut evidence_hashes = BTreeMap::new();
    while let Some(p) = queue.pop_front() {
        if !affected.insert(p.clone()) {
            continue;
        }
        let source_invalid = invalid.contains(&p);
        uncertain |= source_invalid;
        for edge in edges.iter().filter(|e| e["provider"] == p) {
            let c = text(edge, "consumer")?;
            let mut compatible = !source_invalid && !invalid.contains(c);
            for (field, producer) in [("provider_evidence", p.as_str()), ("consumer_evidence", c)] {
                let reference = &edge[field];
                let checked = (|| -> Result<bool, String> {
                    let path = text(reference, "path")?;
                    let (e, hash) = read(&root, path)?;
                    if reference["sha256"] != hash
                        || reference["id"] != e["id"]
                        || reference["version"] != e["version"]
                        || e["producer_id"] != producer
                        || e["provider"] != p
                        || e["consumer"] != c
                    {
                        return Err("compatibility producer identity mismatch".into());
                    }
                    if e["provider_sha256"]
                        != current.get(&p).ok_or("provider source unavailable")?.1
                        || e["consumer_sha256"]
                            != current.get(c).ok_or("consumer source unavailable")?.1
                    {
                        return Err("compatibility source revisions stale".into());
                    }
                    if !matches!(e["compatibility"].as_str(), Some("COMPATIBLE" | "AFFECTED"))
                        || !matches!(e["outcome"].as_str(), Some("PASS" | "FAIL"))
                    {
                        return Err("unknown compatibility verdict".into());
                    }
                    evidence_hashes.insert(path.to_owned(), hash);
                    Ok(e["outcome"] == "PASS" && e["compatibility"] == "COMPATIBLE")
                })();
                match checked {
                    Ok(pass) => compatible &= pass,
                    Err(_) => {
                        uncertain = true;
                        compatible = false;
                    }
                }
            }
            if !compatible {
                queue.push_back(c.to_owned());
            }
        }
    }
    let status = if uncertain {
        "UNCERTAIN_UNVERIFIED"
    } else if affected.len() > 1 {
        "VERIFIED_AFFECTED"
    } else {
        "VERIFIED_COMPATIBLE_UNAFFECTED"
    };
    // Detect source or evidence changes during analysis instead of issuing stale eligibility.
    if read(&root, text(input, "graph_ref")?)?.1 != graph_hash {
        return Err("graph changed during analysis".into());
    }
    for id in &affected {
        if let Some((_, hash)) = current.get(id) {
            if read(&root, text(records[id], "path")?)?.1 != *hash {
                return Err("dependency source changed during analysis".into());
            }
        }
    }
    Ok(
        json!({"changed_id":changed,"affected_ids":affected,"status":status,"verification_eligible":!uncertain,"execution_authorized":false}),
    )
}
