from __future__ import annotations

import sys
from pathlib import Path


TEST_ROOT = Path(__file__).resolve().parent
REPO_ROOT = TEST_ROOT.parents[1]
SOURCE_ROOT = REPO_ROOT / "src"

# Focused local pytest runs must verify the checked-out source tree, not an
# older PTSIP distribution that may already be installed in the interpreter.
for candidate in (TEST_ROOT, SOURCE_ROOT):
    value = str(candidate)
    if value in sys.path:
        sys.path.remove(value)
    sys.path.insert(0, value)
