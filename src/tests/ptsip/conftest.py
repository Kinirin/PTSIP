from __future__ import annotations

import sys
from copy import deepcopy
from pathlib import Path

import pytest


TEST_ROOT = Path(__file__).resolve().parent
REPO_ROOT = TEST_ROOT.parents[2]
SOURCE_ROOT = REPO_ROOT / "src"

# Focused local pytest runs must verify the checked-out source tree, not an
# older PTSIP distribution that may already be installed in the interpreter.
for candidate in (TEST_ROOT, SOURCE_ROOT):
    value = str(candidate)
    if value in sys.path:
        sys.path.remove(value)
    sys.path.insert(0, value)


@pytest.fixture
def historical_pp101_runtime(monkeypatch):
    """Pin tests of the registered 0.3.7/pp.1.01 migration scenario.

    Current distribution tests use the unmodified pp.1.02 registry. This fixture
    never changes shipped compatibility bridges or rewrites historical assets.
    """
    from ptsip.profiles import contracts

    distribution_version = contracts.current_runtime_project_profile_contract().version
    historical = deepcopy(contracts._embedded_registry_payload())
    historical["current"] = "pp.1.01"
    for contract in historical["contracts"]:
        if contract["version"] == "pp.1.01":
            contract["lifecycle"] = "CURRENT"
        elif contract["version"] == "pp.1.02":
            contract["lifecycle"] = "SUPERSEDED"
    monkeypatch.setattr(contracts, "_embedded_registry_payload", lambda: deepcopy(historical))
    initial_modules = set(sys.modules)
    for name, module in tuple(sys.modules.items()):
        if name.startswith("ptsip.") and hasattr(module, "CURRENT_PROJECT_PROFILE_VERSION"):
            monkeypatch.setattr(module, "CURRENT_PROJECT_PROFILE_VERSION", "pp.1.01")
    yield
    for name in set(sys.modules) - initial_modules:
        module = sys.modules[name]
        if name.startswith("ptsip.") and getattr(module, "CURRENT_PROJECT_PROFILE_VERSION", None) == "pp.1.01":
            setattr(module, "CURRENT_PROJECT_PROFILE_VERSION", distribution_version)
