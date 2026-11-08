"""Process adapter for Go-backed policy source operations."""

import atexit
from functools import lru_cache
import json
import os
from pathlib import Path
import subprocess
import tempfile


UNIFIED_VERSION = "ptsip-unified-policy/v3-candidate"
PROTOCOL = "ptsip-common-contract/v1"
COMMAND = "ptsip-common-contract" + (".exe" if os.name == "nt" else "")


class PolicySourceError(ValueError):
    pass


@lru_cache(maxsize=1)
def _command() -> Path:
    module = Path(__file__).resolve().with_name("common_contract")
    installed = module / COMMAND
    if installed.is_file():
        return installed
    if not (module / "go.mod").is_file():
        raise PolicySourceError("PTSIP_COMMON_CONTRACT_EXECUTABLE_MISSING")
    temporary = tempfile.TemporaryDirectory(prefix="ptsip-policy-source-")
    binary = Path(temporary.name) / COMMAND
    try:
        subprocess.run(
            ["go", "build", "-mod=readonly", "-trimpath", "-o", str(binary), "./cmd/ptsip-common-contract"],
            cwd=module, env=dict(os.environ, GOWORK="off", CGO_ENABLED="0"), check=True,
        )
    except (OSError, subprocess.SubprocessError) as error:
        temporary.cleanup()
        raise PolicySourceError(f"PTSIP_COMMON_CONTRACT_BUILD_FAILED: {error}") from error
    atexit.register(temporary.cleanup)
    return binary


def request(operation: str, **payload: object) -> dict:
    raw = json.dumps({"format": PROTOCOL, "operation": operation, **payload}, allow_nan=False)
    try:
        response = subprocess.run(
            [str(_command())], input=raw.encode("ascii"), capture_output=True, check=True, timeout=90,
        )
        report = json.loads(response.stdout)
    except (OSError, subprocess.SubprocessError, ValueError) as error:
        raise PolicySourceError(f"PTSIP_COMMON_CONTRACT_REQUEST_FAILED: {error}") from error
    if not isinstance(report, dict) or report.pop("format", None) != PROTOCOL:
        raise PolicySourceError("PTSIP_COMMON_CONTRACT_RESPONSE_INVALID")
    return report


def source_result(operation: str, key: str, **payload: object):
    report = request(operation, **payload)
    if report.get("status") != "VALID" or key not in report:
        raise PolicySourceError(str(report.get("problem", "PTSIP_COMMON_CONTRACT_RESPONSE_INVALID")))
    return report[key]
