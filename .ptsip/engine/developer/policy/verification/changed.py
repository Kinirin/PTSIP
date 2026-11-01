"""Check an exact Git change range through the registered Go contract entry.

This transport does not select schemas, approve policy semantics or migrate old
documents. Historical policy sources remain available for explicit review.
"""

import argparse
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys


def classify(paths):
    policies, retired = [], []
    for name in sorted(set(paths)):
        path = PurePosixPath(name)
        if (str(path) != name or path.is_absolute() or ".." in path.parts
                or any(c in name for c in "\\:\x00\r\n")):
            raise ValueError("INVALID_CHANGED_PATH")
        if name.lower().startswith(".ptsip/contracts/policy/rules/"):
            retired.append(name)
        elif (name.startswith("developer/policy/")
              and not name.startswith("developer/policy/legacy/")
              and path.suffix in {".yaml", ".yml"}):
            # The catalog is a lookup projection, not a policy document.
            if name != "developer/policy/index.yaml":
                policies.append(name)
    return policies, retired


def check(root, base, head, invoke=subprocess.run):
    report = {"status": "FAIL", "policy_conformance": "NOT_ASSERTED",
              "mutation_authorized": False, "execution_authorized": False,
              "targets": []}

    def run(args):
        result = invoke(args, cwd=root, capture_output=True)
        if result.returncode:
            raise ValueError("COMMAND_FAILED: " + result.stderr.decode("utf-8", errors="replace"))
        return result.stdout

    try:
        # Refs are resolved before use and cannot be options or shell commands.
        if not re.fullmatch(r"[0-9a-fA-F]{40}", base):
            raise ValueError("EXACT_BASE_SHA_REQUIRED")
        if head != "HEAD" and not re.fullmatch(r"[0-9a-fA-F]{40}", head):
            raise ValueError("EXACT_HEAD_SHA_REQUIRED")
        actual = run(["git", "rev-parse", "HEAD"]).decode().strip()
        selected = run(["git", "rev-parse", "--verify", head + "^{commit}"]).decode().strip()
        if actual != selected:
            raise ValueError("CHECKOUT_HEAD_MISMATCH")
        run(["git", "rev-parse", "--verify", base + "^{commit}"])
        # Read only the exact committed checkout; local uncommitted edits cannot
        # be validated and then attributed to an unrelated commit.
        if run(["git", "status", "--porcelain", "--untracked-files=all"]):
            raise ValueError("CLEAN_CHECKOUT_REQUIRED")
        raw = run(["git", "diff", "--name-only", "-z", "--no-renames",
                   "--diff-filter=ACMRT", base, selected, "--"])
        policies, retired = classify([p.decode("utf-8") for p in raw.split(b"\0") if p])
        report.update(base_sha=base.lower(), source_sha=actual, retired_paths=retired)
        if retired:
            raise ValueError("RETIRED_POLICY_CONTRACT_WRITE")
        for target in policies:
            command = ["go", "-C", str(root / ".ptsip/engine/developer"), "run", ".",
                       "--validator", "contract-entry", "--root", str(root), "--target", target]
            result = json.loads(run(command))
            selection = result.get("selection", {})
            validation = result.get("result", {})
            if (result.get("status") != "CHECK_COMPLETED"
                    or validation.get("status") != "VALID"
                    or selection.get("target") != target
                    or selection.get("mutation_authorized") is not False
                    or selection.get("execution_authorized") is not False):
                raise ValueError("REGISTERED_POLICY_CHECK_REQUIRED: " + target)
            report["targets"].append({"path": target, "selection": selection,
                                      "validation": validation})
        report["status"] = "VALID"
        return report, 0
    except (ValueError, OSError, UnicodeError) as error:
        report["problem"] = str(error)
        return report, 2


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".")
    parser.add_argument("--base", required=True)
    parser.add_argument("--head", default="HEAD")
    arguments = parser.parse_args()
    # Reuse this interpreter for the Go entry's existing YAML/runtime bridge.
    os.environ["PTSIP_PYTHON"] = sys.executable
    result, code = check(Path(arguments.root).resolve(), arguments.base, arguments.head)
    print(json.dumps(result, ensure_ascii=True, indent=2))
    sys.exit(code)
