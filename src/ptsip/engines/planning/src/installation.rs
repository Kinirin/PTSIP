use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::BTreeSet;
use std::fs;
use std::path::{Component, Path};

fn text<'a>(v: &'a Value, k: &str) -> Result<&'a str, String> {
    v[k].as_str()
        .filter(|s| !s.is_empty())
        .ok_or_else(|| format!("missing package field: {k}"))
}
fn bytes(root: &Path, reference: &str) -> Result<Vec<u8>, String> {
    let relative = Path::new(reference);
    if relative.is_absolute()
        || reference.contains('\\')
        || relative
            .components()
            .any(|c| !matches!(c, Component::Normal(_)))
    {
        return Err("package reference is not owner-relative".into());
    }
    let mut path = root.to_owned();
    for c in relative.components() {
        path.push(c);
        if fs::symlink_metadata(&path)
            .map_err(|e| e.to_string())?
            .file_type()
            .is_symlink()
        {
            return Err("unregistered package alias".into());
        }
    }
    let path = fs::canonicalize(path).map_err(|e| e.to_string())?;
    if !path.starts_with(root) {
        return Err("package source escaped owner root".into());
    }
    if fs::metadata(&path).map_err(|e| e.to_string())?.len() > 268_435_456 {
        return Err("package artifact byte budget exceeded".into());
    }
    fs::read(path).map_err(|e| e.to_string())
}
fn document(root: &Path, path: &str) -> Result<(Value, String), String> {
    let b = bytes(root, path)?;
    if b.len() > 2_097_152 {
        return Err("package metadata byte budget exceeded".into());
    }
    Ok((
        serde_json::from_slice(&b).map_err(|e| e.to_string())?,
        format!("{:x}", Sha256::digest(&b)),
    ))
}

fn verify_integrated_plan_contract(plan: &Value) -> Result<(), String> {
    if plan["schema_version"] != "ptsip-engine-installation-plan/v2"
        || plan["selection"] != "REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL"
        || plan["default_installation"] != "ONE_COMPLETE_PTSIP_PLATFORM_WHEEL_WITH_ALL_SEVEN_ENGINES"
        || plan["distribution"]["public_python_project"] != "PTSIP"
        || plan["distribution"]["model"] != "SINGLE_PLATFORM_SPECIFIC_PTSIP_WHEEL"
        || plan["distribution"]["engine_pip_dependencies"] != "FORBIDDEN"
        || plan["distribution"]["required_verified_engines"] != 7
        || plan["distribution"]["verification_owner"] != ".ptsip/engine/developer/Distribution/release/"
        || plan["distribution"]["legacy_controller_execution"] != "FORBIDDEN"
        || plan["distribution"]["publication_authorized"] != false
        || plan.get("packages").is_some()
        || plan.get("bundle_selection").is_some()
    {
        return Err("integrated PTSIP native component plan is incomplete or unauthorized".into());
    }
    let components = plan["components"]
        .as_array()
        .ok_or("integrated PTSIP plan requires explicit components")?;
    let ids: BTreeSet<_> = components
        .iter()
        .map(|v| v["engine_id"].as_str().unwrap_or("").to_owned())
        .collect();
    let defaults: BTreeSet<_> = plan["default_available_engine_ids"]
        .as_array()
        .ok_or("explicit default native engine set required")?
        .iter()
        .map(|v| v.as_str().unwrap_or("").to_owned())
        .collect();
    if components.len() != 7 || ids.len() != 7 || ids.contains("") || defaults != ids {
        return Err("integrated PTSIP wheel requires exactly seven distinct registered engines".into());
    }
    for component in components {
        let cargo = text(component, "cargo_manifest_ref")?;
        let kind = cargo
            .strip_prefix("src/ptsip/engines/")
            .and_then(|s| s.strip_suffix("/Cargo.toml"))
            .ok_or("unregistered engine component cargo ownership")?;
        let required_entry = format!("ptsip-engine-{kind}");
        if text(component, "entrypoint")? != required_entry
            || component["distribution"] != "PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT"
            || component["installation"] != "INSTALLED_AS_PART_OF_PTSIP_PLATFORM_WHEEL"
            || component["permission_from_availability"] != false
            || component["installed_is_execution_authorization"] != false
            || component.get("package_distribution").is_some()
            || component["build_targets"].as_array().is_none()
        {
            return Err("independent pip engine package or invalid bundled component".into());
        }
    }
    Ok(())
}

/// Provisional native component selection ahead of a separate versioned
/// planning engine contract cutover. This does NOT check assembled wheel
/// completeness, attest a release, install software or authorize execution.
#[allow(dead_code)]
pub fn select_integrated_candidate(input: &Value) -> Result<Value, String> {

    let root = fs::canonicalize(text(input, "root")?).map_err(|e| e.to_string())?;
    let (plan, revision) = document(&root, text(input, "plan_ref")?)?;
    if input["expected_plan_sha256"] != revision {
        return Err("current integrated installation plan proof missing".into());
    }
    verify_integrated_plan_contract(&plan)?;
    let target = text(input, "host_target")?;
    let package_ids: BTreeSet<_> = plan["components"]
        .as_array()
        .ok_or("explicit package set required")?
        .iter()
        .map(|p| p["engine_id"].as_str().unwrap_or("").to_owned())
        .collect();
    let defaults: BTreeSet<_> = plan["default_available_engine_ids"]
        .as_array()
        .ok_or("explicit default availability set required")?
        .iter()
        .map(|v| v.as_str().unwrap_or("").to_owned())
        .collect();
    if defaults != package_ids || package_ids.len() != 7 {
        return Err("all seven official native components must be declared".into());
    }
    let reports = input["evidence_refs"]
        .as_array()
        .ok_or("explicit package evidence references required")?;
    let mut selected = Vec::new();
    let mut unverified = Vec::new();
    let mut seen = BTreeSet::new();
    for package in plan["components"].as_array().ok_or("component set required")? {
        let id = text(package, "engine_id")?;
        if !seen.insert(id) {
            return Err("ambiguous default package identity".into());
        }
        let proof = (|| -> Result<Value, String> {
            let entrypoint = text(package, "entrypoint")?;
            let cargo_manifest = text(package, "cargo_manifest_ref")?;
            let kind = cargo_manifest
                .strip_prefix("src/ptsip/engines/")
                .and_then(|s| s.strip_suffix("/Cargo.toml"))
                .ok_or("unregistered engine package cargo ownership")?;
            if entrypoint != format!("ptsip-engine-{kind}")
                || package["distribution"] != "PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT"
                || package["installation"] != "INSTALLED_AS_PART_OF_PTSIP_PLATFORM_WHEEL"
                || package["installed_is_execution_authorization"] != false
                || package.get("package_distribution").is_some()
            {
                return Err("official engine must be bundled in the PTSIP target wheel".into());
            }
            if package["permission_from_availability"] != false
                || !package["build_targets"]
                    .as_array()
                    .ok_or("target declarations required")?
                    .iter()
                    .any(|v| v == target)
            {
                return Err("target compatibility not declared".into());
            }
            let mut matches = Vec::new();
            for reference in reports {
                let (report, hash) = document(&root, text(reference, "path")?)?;
                if reference["id"] != report["id"]
                    || reference["version"] != report["version"]
                    || reference["sha256"] != hash
                {
                    return Err("package report reference stale".into());
                }
                if report["engine_id"] == id && report["host_target"] == target {
                    matches.push(report);
                }
            }
            if matches.len() != 1 {
                return Err("package environment evidence missing or ambiguous".into());
            }
            let report = &matches[0];
            if report["implementation_version"] != package["version"]
                || report["outcome"] != "PASS"
                || report["verification_scope"] != "NATIVE_TEST_BUILD_PACKAGE_STDIO_CONFORMANCE"
            {
                return Err("package verification scope or version mismatch".into());
            }
            let manifest_ref = &report["manifest_ref"];
            let (manifest, mhash) = document(&root, text(manifest_ref, "path")?)?;
            if manifest_ref["sha256"] != mhash
                || manifest["engine_id"] != id
                || manifest["implementation_version"] != package["version"]
            {
                return Err("installed implementation identity proof stale".into());
            }
            for r in manifest["capabilities"]
                .as_array()
                .ok_or("package capability contracts required")?
            {
                let (c, hash) = document(&root, text(r, "path")?)?;
                if r["sha256"] != hash
                    || c["contract_id"] != r["id"]
                    || c["version"] != r["version"]
                {
                    return Err("package capability contract proof stale".into());
                }
            }
            let artifact = &report["artifact_ref"];
            let artifact_bytes = bytes(&root, text(artifact, "path")?)?;
            if artifact["sha256"] != format!("{:x}", Sha256::digest(&artifact_bytes))
                || artifact_bytes.is_empty()
            {
                return Err("package artifact proof stale".into());
            }
            let matches_platform = if target.contains("windows") {
                artifact_bytes.starts_with(b"MZ")
            } else if target.contains("linux") {
                artifact_bytes.starts_with(b"\x7fELF")
            } else if target.contains("apple") {
                artifact_bytes.starts_with(&[0xcf, 0xfa, 0xed, 0xfe])
                    || artifact_bytes.starts_with(&[0xfe, 0xed, 0xfa, 0xcf])
            } else {
                false
            };
            if !matches_platform {
                return Err("artifact platform does not match the declared host target".into());
            }
            Ok(
                json!({"engine_id":id,"version":package["version"],"distribution":"PTSIP","native_entrypoint":entrypoint,"host_target":target,"artifact_ref":artifact,"installation_authorized":false,"execution_authorized":false}),
            )
        })();
        match proof {
            Ok(value) => selected.push(value),
            Err(reason) => unverified.push(json!({"engine_id":id,"reason":reason})),
        }
    }
    if document(&root, text(input, "plan_ref")?)?.1 != revision {
        return Err("installation plan changed during selection".into());
    }
    Ok(
        json!({"plan_revision":revision,"component_verification_ready":unverified.is_empty(),"ready":false,"selected_components":selected,"unverified_components":unverified,"wheel_composition_verified":false,"publication_authorized":false,"installation_authorized":false,"external_installation_performed":false,"verification_report_authenticity":"SELECTED_OWNER_SOURCE_NOT_SELF_ISSUED_BY_ENGINE"}),
    )
}

/// The published planning engine operation has a different, older machine
/// semantics contract. The new integrated candidate must not silently replace
/// it at the same engine/contract version; a separately authorized versioned
/// operation cutover and exact-source Rust verification are required.
pub fn select(_input: &Value) -> Result<Value, String> {
    Err("retired independent-engine installation route; integrated planning operation not yet version-registered".into())
}

#[cfg(test)]
mod integrated_candidate_tests {
    use super::*;

    #[test]
    fn bundled_distribution_does_not_derive_install_authority() {
        let mut components = Vec::new();
        for name in ["context", "storage", "validation", "discovery", "planning", "execution", "projection"] {
            components.push(json!({
                "engine_id": format!("urn:uuid:{name}"),
                "version": "0.3.0",
                "cargo_manifest_ref":format!("src/ptsip/engines/{name}/Cargo.toml"),
                "entrypoint":format!("ptsip-engine-{name}"),
                "distribution":"PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT",
                "installation":"INSTALLED_AS_PART_OF_PTSIP_PLATFORM_WHEEL",
                "permission_from_availability":false,
                "installed_is_execution_authorization":false,
                "build_targets":["x86_64-pc-windows-msvc","x86_64-unknown-linux-gnu"]
            }));
        }
        let ids: Vec<_> = components.iter().map(|v| v["engine_id"].clone()).collect();
        let mut plan = json!({
            "schema_version":"ptsip-engine-installation-plan/v2",
            "selection":"REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL",
            "default_installation":"ONE_COMPLETE_PTSIP_PLATFORM_WHEEL_WITH_ALL_SEVEN_ENGINES",
            "distribution":{
                "public_python_project":"PTSIP",
                "model":"SINGLE_PLATFORM_SPECIFIC_PTSIP_WHEEL",
                "engine_pip_dependencies":"FORBIDDEN",
                "required_verified_engines":7,
                "verification_owner":".ptsip/engine/developer/Distribution/release/",
                "legacy_controller_execution":"FORBIDDEN",
                "publication_authorized":false
            },
            "default_available_engine_ids":ids,
            "components":components
        });
        assert!(verify_integrated_plan_contract(&plan).is_ok());
        plan["distribution"]["engine_pip_dependencies"] = json!("REQUIRED");
        assert!(verify_integrated_plan_contract(&plan).is_err());
        plan["distribution"]["engine_pip_dependencies"] = json!("FORBIDDEN");
        plan["components"][0]["distribution"] = json!("INDEPENDENT_VERSIONED_NATIVE_WHEEL");
        assert!(verify_integrated_plan_contract(&plan).is_err());
        plan["components"][0]["distribution"] = json!("PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT");
        plan["components"][0]["installed_is_execution_authorization"] = json!(true);
        assert!(verify_integrated_plan_contract(&plan).is_err());
        plan["components"][0]["installed_is_execution_authorization"] = json!(false);
        plan["distribution"]["publication_authorized"] = json!(true);
        assert!(verify_integrated_plan_contract(&plan).is_err());
        assert!(select(&json!({})).is_err());
    }
}
