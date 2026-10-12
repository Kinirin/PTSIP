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

pub fn select(input: &Value) -> Result<Value, String> {
    let root = fs::canonicalize(text(input, "root")?).map_err(|e| e.to_string())?;
    let (plan, revision) = document(&root, text(input, "plan_ref")?)?;
    if input["expected_plan_sha256"] != revision
        || plan["schema_version"] != "ptsip-engine-installation-plan/v1"
    {
        return Err("current installation plan proof missing".into());
    }
    if plan["selection"] != "REGISTERED_INDEPENDENT_AUTO_INSTALLED_PACKAGES_CURRENT_HOST_ONLY"
        || plan["default_installation"] != "SEVEN_OFFICIAL_PACKAGES_AS_PYTHON_DISTRIBUTION_DEPENDENCIES"
        || plan.get("bundle_selection").is_some()
    {
        return Err("manual or bundled official engine installation contract is obsolete".into());
    }
    let target = text(input, "host_target")?;
    let package_ids: BTreeSet<_> = plan["packages"]
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
        return Err("seven default packages must have exact independent declarations".into());
    }
    let reports = input["evidence_refs"]
        .as_array()
        .ok_or("explicit package evidence references required")?;
    let mut selected = Vec::new();
    let mut unverified = Vec::new();
    let mut seen = BTreeSet::new();
    for package in plan["packages"].as_array().ok_or("package set required")? {
        let id = text(package, "engine_id")?;
        if !seen.insert(id) {
            return Err("ambiguous default package identity".into());
        }
        let proof = (|| -> Result<Value, String> {
            let package_name = text(package, "package_distribution")?;
            let cargo_manifest = text(package, "cargo_manifest_ref")?;
            let kind = cargo_manifest
                .strip_prefix("src/ptsip/engines/")
                .and_then(|s| s.strip_suffix("/Cargo.toml"))
                .ok_or("unregistered engine package cargo ownership")?;
            if package_name != format!("ptsip-engine-{kind}")
                || package["distribution"] != "INDEPENDENT_VERSIONED_NATIVE_WHEEL"
                || package["installation"] != "AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY"
                || package["installed_is_execution_authorization"] != false
            {
                return Err("official engine requires independent auto-installed wheel".into());
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
                json!({"engine_id":id,"version":package["version"],"distribution":package["distribution"],"package_distribution":package_name,"host_target":target,"artifact_ref":artifact,"installation_authorized":false,"execution_authorized":false}),
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
        json!({"plan_revision":revision,"ready":unverified.is_empty(),"selected_packages":selected,"unverified_packages":unverified,"installation_authorized":false,"external_installation_performed":false,"verification_report_authenticity":"SELECTED_OWNER_SOURCE_NOT_SELF_ISSUED_BY_ENGINE"}),
    )
}
