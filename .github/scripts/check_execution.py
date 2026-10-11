#!/usr/bin/env python3
"""Independent Check selection, execution and coverage adapter (no policy authority)."""
from __future__ import annotations
import argparse
import datetime
import json
import os
import re
import runpy
import subprocess
import sys
import urllib.request
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[2]
CHECKS = ROOT / ".github/checks.yaml"
MODES = ROOT / ".github/test_modes.yaml"
PROFILE = ROOT / ".ptsip/profiles/main.ptsip.yaml"
EVIDENCE = ROOT / ".artifacts/check-results"

def exact_sha():
    sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    if sha != os.getenv("GITHUB_SHA") or not sha:
        raise ValueError("checkout Source SHA differs from dispatched SHA")
    return sha

def registry():
    check_registry = json.loads(CHECKS.read_text(encoding="utf-8"))
    inventory = json.loads((ROOT / check_registry["legacy_inventory"]).read_text(encoding="utf-8"))
    modes = yaml.safe_load(MODES.read_text(encoding="utf-8"))
    validator = runpy.run_path(str(ROOT / ".github/scripts/validate_test_modes.py"))
    errors = validator["validate_registry"](MODES, PROFILE, ROOT)
    if errors:
        raise ValueError("Mode registry invalid: " + "; ".join(errors))
    if check_registry.get("format") != "ptsip-check-registry/v1" or modes.get("version") != "3.0":
        raise ValueError("unsupported verification registry format")
    mode_map = {x["id"]: x for x in modes["modes"]}
    results, mapped = {}, []
    for check in check_registry["legacy_checks"]:
        name = check["id"]
        if name in results or not re.fullmatch(r"[a-z0-9][a-z0-9-]*", name):
            raise ValueError("invalid/duplicate legacy Check " + name)
        if check["mode"] not in mode_map:
            raise ValueError("legacy Check has no registered Mode: " + name)
        path = (ROOT / check["script"]).resolve()
        if not path.is_file() or ROOT not in path.parents:
            raise ValueError("missing or unsafe legacy Check script: " + name)
        if not check.get("triggers") or not check.get("legacy_steps"):
            raise ValueError("missing legacy applicability/coverage: " + name)
        mapped.extend(check["legacy_steps"])
        results[name] = dict(check, kind="legacy")
    items = inventory["required_execution_steps"]
    if len(mapped) != len(set(mapped)) or set(mapped) != set(items):
        raise ValueError("WEC-09/10/11 legacy obligations missing or duplicated")
    for mode in modes["modes"]:
        for kind in ("pytest", "go"):
            for number, target in enumerate(mode["execution"].get(kind, []), 1):
                check_id = "%s--%s-%d" % (mode["id"], kind, number)
                if check_id in results:
                    raise ValueError("Check identifier collision")
                results[check_id] = dict(id=check_id, mode=mode["id"], kind=kind, target=target)
    return check_registry, mode_map, results

def csv(text):
    if not text:
        return []
    parts = [x.strip() for x in text.split(",")]
    if not all(parts) or len(set(parts)) != len(parts):
        raise ValueError("invalid explicit comma-separated identifiers")
    return parts

def select(checks, modes, registry, event, preset, explicit, requested_modes):
    valid = {"auto", "full", "a3-prerelease", "policy-lifecycle",
             "agent-context-migration", "markdown-cleanup-simulation"}
    if preset not in valid:
        raise ValueError("unknown verification preset " + preset)
    manual, chosen_modes = csv(explicit), csv(requested_modes)
    if (manual or chosen_modes) and preset != "auto":
        raise ValueError("preset and explicit Check/Mode selection cannot mix")
    if any(x not in checks for x in manual) or any(x not in modes for x in chosen_modes):
        raise ValueError("unregistered explicit Check or Mode")
    selected = set(manual)
    def add_mode(mode):
        selected.update(k for k,c in checks.items() if c["mode"] == mode and c["kind"] in ("pytest","go"))
    for mode in chosen_modes:
        add_mode(mode)
    if event == "push":
        selected.update(c["id"] for c in registry["legacy_checks"] if "push" in c["triggers"])
        if os.getenv("HEAD_COMMIT_MESSAGE","").startswith("release:"):
            preset = "full"
    elif preset == "auto" and not selected:
        base = os.getenv("AUTO_BASE","")
        if not base or set(base) == {"0"}:
            base = subprocess.check_output(["git","rev-parse","HEAD^"],cwd=ROOT,text=True).strip()
        raw = subprocess.check_output([sys.executable,".github/scripts/resolve_test_modes.py",
          "automatic","--base",base,"--head","HEAD"],cwd=ROOT,text=True)
        result = json.loads(raw)
        if result.get("resolution") == "NO_TEST_MODE_REQUIRED":
            names = []
        else:
            names = [v["id"] for v in result.get("plan",[])]
            if not names: raise ValueError("auto selection missing required coverage proof")
        for mode in names:
            if mode not in modes: raise ValueError("resolver emitted unknown Mode")
            add_mode(mode)
    if preset == "full":
        for mode in modes:
            add_mode(mode)
    if preset != "auto":
        selected.update(c["id"] for c in registry["legacy_checks"] if preset in c["triggers"])
    return sorted(selected),preset

def append_output(k,v):
    output = os.getenv("GITHUB_OUTPUT")
    if output:
        with open(output,"a",encoding="utf-8") as f: f.write(k+"="+v+"\n")

def mark_status(name,sha):
    token, repo = os.getenv("GITHUB_TOKEN"),os.getenv("GITHUB_REPOSITORY")
    if not token or not repo: return
    url="https://api.github.com/repos/"+repo+"/statuses/"+sha
    target=os.getenv("GITHUB_SERVER_URL","https://github.com")+"/"+repo+"/actions/runs/"+os.getenv("GITHUB_RUN_ID","")
    body=json.dumps({"state":"success","context":name,
          "description":"Registered verification passed for exact SHA","target_url":target}).encode()
    req=urllib.request.Request(url,body,method="POST",headers={
       "Authorization":"Bearer "+token,"Accept":"application/vnd.github+json",
       "Content-Type":"application/json","X-GitHub-Api-Version":"2022-11-28"})
    urllib.request.urlopen(req,timeout=30).close()

def run_check(checks,identifier):
    if identifier not in checks: raise ValueError("unregistered Check "+identifier)
    sha=exact_sha()
    check=checks[identifier]
    EVIDENCE.mkdir(parents=True,exist_ok=True)
    output=EVIDENCE/(identifier+".json")
    env=dict(os.environ, PYTHONPATH=".:src",PTSIP_EXECUTION_SHA=sha,
       PTSIP_EXECUTION_BRANCH=os.getenv("GITHUB_REF_NAME",""),VERIFIED_SHA=sha,
       REPOSITORY_ID=os.getenv("GITHUB_REPOSITORY",""),PP_HEAD=sha,
       PUSH_BEFORE=os.getenv("GITHUB_EVENT_BEFORE",""),
       DEFAULT_BRANCH=os.getenv("GITHUB_DEFAULT_BRANCH","main"),
       REF_TYPE=os.getenv("GITHUB_REF_TYPE","branch"))
    cmd=[]
    outcome="UNRESOLVED"
    error=None
    try:
        if check["kind"]=="legacy":
            cmd=["pwsh","-NoProfile","-File",check["script"]]
        elif check["kind"]=="pytest":
            cmd=[sys.executable,"-m","pytest","-q",check["target"]]
        else:
            cmd=["go","-C",check["target"],"test","-count=1","-v","./..."]
        if identifier=="release-distribution":
            subprocess.run([sys.executable,"-m","pip","install","build","twine"],cwd=ROOT,env=env,check=True)
        result=subprocess.run(cmd,cwd=ROOT,env=env)
        outcome="PASS" if result.returncode==0 else "FAIL"
        if outcome=="FAIL": error="verification process exited "+str(result.returncode)
    except (OSError,subprocess.SubprocessError) as exc:
        error="execution infrastructure error: "+str(exc)
    finally:
        record={"format":"ptsip-check-execution-evidence/v1","check_id":identifier,
                "mode_id":check["mode"],"source_sha":sha,
                "execution_identity":{"run_id":os.getenv("GITHUB_RUN_ID"),
                    "run_attempt":os.getenv("GITHUB_RUN_ATTEMPT"),"job":os.getenv("GITHUB_JOB")},
                "outcome":outcome,"normative_assure_judgment":"NOT_TRANSFERRED",
                "evidence_reference":str(output.relative_to(ROOT)),"command":cmd,
                "error":error,"timestamp":datetime.datetime.now(datetime.timezone.utc).isoformat()}
        output.write_text(json.dumps(record,indent=2,ensure_ascii=False)+"\n",encoding="utf-8")
        print(json.dumps(record,ensure_ascii=False))
    if outcome=="PASS" and check.get("status_context"):
        mark_status(check["status_context"],sha)
    return 0 if outcome=="PASS" else 1

def coverage(required,preset):
    expected=json.loads(required)
    if not isinstance(expected,list) or any(not isinstance(x,str) for x in expected):
        raise ValueError("invalid required Check list")
    _,_,checks=registry()
    if any(x not in checks for x in expected):
        raise ValueError("coverage contains unregistered Check")
    source=exact_sha()
    collected={}
    for f in (ROOT/".artifacts/collected").glob("*.json"):
        e=json.loads(f.read_text(encoding="utf-8"))
        id=e.get("check_id")
        if id in collected: raise ValueError("duplicate Check evidence "+str(id))
        collected[id]=e
    failures=[]
    for id in expected:
        e=collected.get(id)
        if not e or e.get("source_sha")!=source or e.get("outcome")!="PASS":
            failures.append(id+":"+(e.get("outcome","INVALID") if e else "MISSING"))
    print(json.dumps({"required":len(expected),"blocking":failures,"source_sha":source}))
    if failures: return 1
    mark_status("ci/tooling-test" if preset in ("full","a3-prerelease") else "ci/change-verification",source)
    return 0

def main():
    p=argparse.ArgumentParser()
    sub=p.add_subparsers(dest="action",required=True)
    sub.add_parser("validate")
    plan=sub.add_parser("plan")
    for flag in ("event","preset","checks","modes"): plan.add_argument("--"+flag,default="")
    run=sub.add_parser("run");run.add_argument("identifier")
    cv=sub.add_parser("coverage");cv.add_argument("--required",required=True);cv.add_argument("--preset",default="auto")
    a=p.parse_args()
    try:
        r,m,c=registry()
        if a.action=="validate":
            print("Registry PASS: %d Modes; %d Checks"%(len(m),len(c)));return 0
        if a.action=="plan":
            exact_sha()
            selected,preset=select(c,m,r,a.event,a.preset,a.checks,a.modes)
            append_output("checks",json.dumps(selected if selected else ["__no_check__"],separators=(",",":")))
            append_output("required",json.dumps(selected,separators=(",",":")))
            append_output("preset",preset)
            print(json.dumps({"required":selected,"preset":preset}));return 0
        if a.action=="run": return run_check(c,a.identifier)
        if a.action=="coverage": return coverage(a.required,a.preset)
    except (ValueError,KeyError,OSError,subprocess.SubprocessError) as exc:
        print("FAIL_CLOSED: "+str(exc),file=sys.stderr)
        return 2
    return 2

if __name__=="__main__":
    raise SystemExit(main())
