use crate::{profile, profile_path, repository, source};
use serde_json::{json, Value};
use std::collections::BTreeSet;

pub const TRANSITION: &str = "CLASSIFY_EXPLICIT_PROFILE_TRANSITION_BINDINGS_AND_SNAPSHOT";
pub const CONVERGENCE: &str =
    "VERIFY_EXPLICIT_DIRECT_PROFILE_CONVERGENCE_WITH_REGISTERED_TRANSITIONS";

#[derive(Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
struct DraftVersion(u64, u64, u64);

impl DraftVersion {
    fn parse(declared: &str) -> Option<Self> {
        let mut segments = declared.trim().strip_suffix("-draft")?.split('.');
        let mut next = || {
            let value = segments.next()?;
            if value.is_empty() || !value.bytes().all(|byte| byte.is_ascii_digit()) {
                return None;
            }
            value.parse::<u64>().ok()
        };
        let parsed = Self(next()?, next()?, next()?);
        segments.next().is_none().then_some(parsed)
    }

    fn semantic(self) -> String {
        format!("{}.{}.{}", self.0, self.1, self.2)
    }

    fn value(self) -> Value {
        let semantic = self.semantic();
        json!({"major":self.0,"minor":self.1,"micro":self.2,
            "semantic":semantic,"draft_label":format!("{semantic}-draft")})
    }
}

fn issue(code: &str, message: impl Into<String>, path: Option<&str>) -> Value {
    json!({"code":code,"message":message.into(),"path":path})
}

fn invalid(diagnostics: Vec<Value>) -> Value {
    json!({"valid":false,"state":null,"diagnostics":diagnostics})
}

fn valid(state: Value) -> Value {
    json!({"valid":true,"state":state,"diagnostics":[]})
}

fn canonical_path(binding: &Value, temporary: bool) -> Result<&str, String> {
    let path = source::text(binding, "path")?;
    if profile_path::normalize(Some(path))? != path || binding["temporary"] != temporary {
        return Err("invalid Profile path or binding role".into());
    }
    Ok(path)
}

fn validate_bindings(input: &Value) -> Result<Vec<Value>, String> {
    let mut diagnostics = Vec::new();
    let mut paths = BTreeSet::new();
    if input["source"].is_null() {
        diagnostics.push(issue(
            "NO_CANONICAL_PROFILE",
            "Selected Profile source is missing",
            None,
        ));
    } else {
        paths.insert(canonical_path(&input["source"], false)?);
    }
    let candidates = input["candidates"]
        .as_array()
        .ok_or("candidate bindings required")?;
    for binding in candidates {
        let path = canonical_path(binding, true)?;
        if !paths.insert(path) {
            diagnostics.push(issue(
                "DUPLICATE_PROFILE_PATH",
                "Profile path was bound more than once",
                Some(path),
            ));
        }
    }
    if repository::incomplete(&input["snapshot"]) {
        diagnostics.push(issue(
            "INCOMPLETE_REPOSITORY_SNAPSHOT",
            "Repository observation was incomplete",
            None,
        ));
    }
    Ok(diagnostics)
}

fn draft_binding(binding: &Value, temporary: bool) -> Result<(DraftVersion, Value), Value> {
    let declared = binding["declared_version"].as_str().unwrap_or("");
    let path = binding["path"].as_str();
    let version = DraftVersion::parse(declared).ok_or_else(|| {
        issue(
            "INVALID_PROFILE_VERSION",
            "Legacy transition requires a <major>.<minor>.<micro>-draft identity",
            path,
        )
    })?;
    if temporary {
        let basename = path.and_then(|path| path.rsplit('/').next()).unwrap_or("");
        let token = basename
            .strip_prefix("ptsip_")
            .and_then(|name| name.strip_suffix(".yaml"));
        if token.map(|token| format!("{token}-draft")).as_deref() != Some(declared) {
            return Err(issue(
                "PROFILE_VERSION_FILENAME_MISMATCH",
                "Legacy temporary filename differs from the declared draft identity",
                path,
            ));
        }
    }
    let mut parsed = binding.clone();
    parsed["version"] = version.value();
    Ok((version, parsed))
}

pub fn transition_state(input: &Value) -> Result<Value, String> {
    // The identity-only branch supports legacy draft planning without
    // treating a label parse as an observation of a repository.
    if input["root"].is_null() {
        let declared = source::text(&input["source"], "declared_version")?;
        return Ok(match DraftVersion::parse(declared) {
            Some(version) => valid(version.value()),
            None => invalid(vec![issue(
                "INVALID_PROFILE_VERSION",
                "Expected a legacy draft identity",
                None,
            )]),
        });
    }
    let root = profile_path::directory(source::text(input, "root")?)?;
    let diagnostics = validate_bindings(input)?;
    if !diagnostics.is_empty() {
        return Ok(invalid(diagnostics));
    }
    let (source_version, canonical) = match draft_binding(&input["source"], false) {
        Ok(binding) => binding,
        Err(diagnostic) => return Ok(invalid(vec![diagnostic])),
    };
    let mut versions = BTreeSet::new();
    let mut temporaries = Vec::new();
    let mut diagnostics = Vec::new();
    for binding in input["candidates"]
        .as_array()
        .ok_or("candidate bindings required")?
    {
        let (version, binding) = match draft_binding(binding, true) {
            Ok(binding) => binding,
            Err(diagnostic) => {
                diagnostics.push(diagnostic);
                continue;
            }
        };
        let path = binding["path"].as_str();
        if !versions.insert(version) {
            diagnostics.push(issue(
                "DUPLICATE_TARGET_IDENTITY",
                "Multiple temporary Profiles declare the same draft identity",
                path,
            ));
        }
        if version <= source_version {
            diagnostics.push(issue(
                "NON_MONOTONIC_TARGET",
                "Temporary draft must follow the canonical draft version",
                path,
            ));
        }
        temporaries.push((version, binding));
    }
    if !diagnostics.is_empty() {
        return Ok(invalid(diagnostics));
    }
    temporaries.sort_by_key(|(version, _)| *version);
    let temporaries: Vec<_> = temporaries
        .into_iter()
        .map(|(_, binding)| binding)
        .collect();
    let mode = match temporaries.len() {
        0 => "IDLE",
        1 => "SIMPLE",
        _ => "SEQUENTIAL",
    };
    let final_point = temporaries.last();
    let mut sources = Vec::new();
    if let Some((_, earlier)) = temporaries.split_last() {
        sources.extend(earlier.iter().rev());
        sources.push(&canonical);
    }
    let observed: Vec<_> = std::iter::once(&canonical).chain(&temporaries).collect();
    Ok(valid(json!({"mode":mode,"canonical_source":canonical,
        "temporary_profiles":temporaries,"final_point":final_point,"ordered_sources":sources,
        "snapshot":{"repository_root":root,"repository":input["snapshot"],"profiles":observed}})))
}

pub fn convergence(input: &Value) -> Result<Value, String> {
    let diagnostics = validate_bindings(input)?;
    if !diagnostics.is_empty() {
        return Ok(invalid(diagnostics));
    }
    let source = &input["source"];
    let declared = source::text(source, "declared_version")?;
    let path = source::text(source, "path")?;
    let version = source::text(&input["selected_contract"], "version")?;
    let selected = profile::identity(&json!({"version":version,"operation":"ANALYZE",
        "selected_contract":input["selected_contract"]}))?;
    if selected["status"] != "SUPPORTED" {
        return Ok(invalid(vec![issue(
            "PP_CONVERGENCE_UNSUPPORTED_TARGET",
            "Selected contract does not support analysis",
            Some(path),
        )]));
    }
    let candidates = input["candidates"]
        .as_array()
        .ok_or("candidate bindings required")?;
    if declared == version {
        if !candidates.is_empty() {
            return Ok(invalid(
                candidates
                    .iter()
                    .map(|binding| {
                        issue(
                            "PP_CONVERGENCE_UNEXPECTED_TEMPORARY",
                            "Current Profile already matches the selected contract",
                            binding["path"].as_str(),
                        )
                    })
                    .collect(),
            ));
        }
        return Ok(valid(json!({"mode":"CURRENT","source":source,
            "source_compatibility_contract":version,"target_contract":version,"transition_kind":null,
            "target_path":path,"target":null,"target_is_legacy_alias":false,"requires_temporary_target":false,
            "intermediate_profiles":[],"snapshot":input["snapshot"]})));
    }

    let bridges = input["historical_bridges"]
        .as_array()
        .ok_or("historical bridges required")?;
    // The older direct-edge validator consumes the edge fields. Compatibility
    // identity is additional evidence of the historical convergence state.
    let edges: Vec<_> = bridges
        .iter()
        .map(|bridge| {
            let mut edge = bridge.clone();
            edge.as_object_mut()
                .expect("typed historical bridge")
                .remove("compatibility_contract");
            edge
        })
        .collect();
    let transition = profile::transition(&json!({"source_contract_version":declared,
        "source_revision":source["specification_revision"],"target_contract_version":version,
        "transitions":input["transitions"],"historical_bridges":edges}))?;
    if transition["allowed"] != true {
        return Ok(invalid(vec![issue(
            "PP_CONVERGENCE_UNSUPPORTED_SOURCE_TARGET",
            "No exact direct transition connects the source to the selected contract",
            Some(path),
        )]));
    }
    let historical = bridges.iter().find(|bridge| {
        bridge["declared_version"] == declared
            && bridge["specification_revision"] == source["specification_revision"]
            && bridge["target_contract"] == version
    });
    let compatible = if let Some(bridge) = historical {
        match bridge["compatibility_contract"].as_str() {
            Some(version) => version,
            None => {
                return Ok(invalid(vec![issue(
                    "PP_CONVERGENCE_COMPATIBILITY_IDENTITY_REQUIRED",
                    "Historical bridge must identify the source compatibility contract",
                    Some(path),
                )]))
            }
        }
    } else {
        declared
    };
    if candidates.len() > 1 {
        return Ok(invalid(vec![issue(
            "DUPLICATE_EQUIVALENT_TARGET",
            "Direct convergence requires at most one explicitly observed target",
            None,
        )]));
    }
    let kind = &transition["transition_kind"];
    let mut legacy_alias = false;
    let (target_path, target, mode) = match candidates.first() {
        Some(candidate) => {
            let target_path = source::text(candidate, "path")?;
            let exact_alias = bridges.iter().any(|bridge| {
                bridge["target_contract"] == version
                    && bridge["legacy_target_filename"] == target_path
                    && bridge["declared_version"] == candidate["declared_version"]
                    && bridge["specification_revision"] == candidate["specification_revision"]
            });
            if candidate["declared_version"] != version && !exact_alias {
                return Ok(invalid(vec![issue("PP_CONVERGENCE_SYNTHETIC_INTERMEDIATE",
                    "Observed target does not bind the selected contract or its exact historical alias", Some(target_path))]));
            }
            legacy_alias = exact_alias;
            (
                target_path.to_string(),
                Some(candidate),
                "DIRECT_SEMANTIC_MIGRATION",
            )
        }
        None if kind == "IDENTITY_ONLY" => (path.to_string(), None, "IDENTITY_ONLY"),
        None if historical.is_some() && version == "pp.1.01" => {
            // This is the existing target filename of the admitted pp.1.01
            // bridge. A new official version gets no inferred target path.
            (
                "ptsip_pp1.01.yaml".to_string(),
                None,
                "DIRECT_SEMANTIC_MIGRATION",
            )
        }
        None => {
            return Ok(invalid(vec![issue(
                "PP_CONVERGENCE_TARGET_REQUIRED",
                "A semantic transition requires an explicitly bound target Profile",
                None,
            )]))
        }
    };
    Ok(valid(
        json!({"mode":mode,"source":source,"source_compatibility_contract":compatible,
        "target_contract":version,"transition_kind":kind,"target_path":target_path,"target":target,
        "target_is_legacy_alias":legacy_alias,"requires_temporary_target":target.is_none() && mode == "DIRECT_SEMANTIC_MIGRATION",
        "intermediate_profiles":[],"snapshot":input["snapshot"]}),
    ))
}
