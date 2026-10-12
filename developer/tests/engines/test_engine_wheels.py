"""Offline unit checks for independent native wheel assembly; no release admission."""
from __future__ import annotations

import csv
import importlib.util
import io
import json
import tempfile
import unittest
import zipfile
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[3] / "developer" / "automation" / "engine_wheels.py"
SPEC = importlib.util.spec_from_file_location("ptsip_engine_wheels", SCRIPT)
module = importlib.util.module_from_spec(SPEC)
assert SPEC and SPEC.loader
SPEC.loader.exec_module(module)

KINDS = ("context", "storage", "validation", "discovery", "planning", "execution", "projection")
VERSIONS = {"context": "0.2.0", "projection": "0.2.0"}
ENGINES = {"kind": "UNKNOWN"}


def write(root: Path, name: str, value: dict | str) -> None:
    f = root / name
    f.parent.mkdir(parents=True, exist_ok=True)
    f.write_text(value if isinstance(value, str) else json.dumps(value), encoding="utf-8")


class IndependentEngineWheelTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        entries = []
        packages = []
        for kind in KINDS:
            engine_id = "urn:uuid:" + kind
            version = VERSIONS.get(kind, "0.3.0")
            entry = {
                "kind": kind.upper(),
                "engine_id": engine_id,
                "version": version,
                "manifest_ref": f"src/ptsip/engines/{kind}/manifest.json",
                "execution_binding_ref": f"src/ptsip/engines/{kind}/execution-bindings.json",
            }
            entries.append(entry)
            packages.append({
                "engine_id": engine_id,
                "version": version,
                "distribution": "INDEPENDENT_VERSIONED_NATIVE_WHEEL",
                "package_distribution": "ptsip-engine-" + kind,
                "installation": "AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY",
                "build_targets": list(module.TARGETS),
                "permission_from_availability": False,
                "installed_is_execution_authorization": False,
            })
            write(self.root, entry["manifest_ref"], {
                "engine_id": engine_id, "implementation_version": version
            })
            write(self.root, entry["execution_binding_ref"], {"bindings": [{
                "engine_id": engine_id,
                "engine_version": version,
                "entrypoint": "ptsip-engine-" + kind,
            }]})
        write(self.root, "src/ptsip/engines/index.json", {"official_engine_entries": entries})
        write(self.root, "src/ptsip/engines/contracts/installation-plan.json", {
            "packages": packages
        })
        write(self.root, "License-Authority/LICENSE.md", "Reference license fixture")
        self.binary = self.root / "ptsip-engine-context"
        self.binary.write_bytes(b"\x7fELF" + b"X" * 300)
        self.output = self.root / "wheelhouse"

    def run_builder(self, **kwargs: object) -> tuple[Path, Path]:
        args = {
            "root": self.root,
            "kind": "context",
            "target": "x86_64-unknown-linux-gnu",
            "binary": self.binary,
            "output": self.output,
            "source_sha": "a" * 40,
        }
        args.update(kwargs)
        return module.build(**args)

    def test_one_engine_wheel_has_only_its_registered_binary(self) -> None:
        wheel, receipt = self.run_builder()
        self.assertEqual(wheel.name, "ptsip_engine_context-0.2.0-py3-none-linux_x86_64.whl")
        with zipfile.ZipFile(wheel) as bundle:
            names = set(bundle.namelist())
            scripts = [n for n in names if "/scripts/" in n]
            self.assertEqual(scripts, [
                "ptsip_engine_context-0.2.0.data/scripts/ptsip-engine-context"
            ])
            self.assertNotIn("ptsip-engine-storage", "\n".join(names))
            self.assertEqual(bundle.read(scripts[0]), self.binary.read_bytes())
            record = "ptsip_engine_context-0.2.0.dist-info/RECORD"
            rows = list(csv.reader(io.StringIO(bundle.read(record).decode())))
            self.assertEqual({r[0] for r in rows}, names)
            self.assertEqual(rows[-1], [record, "", ""])
            self.assertIn(b"Tag: py3-none-linux_x86_64",
                          bundle.read("ptsip_engine_context-0.2.0.dist-info/WHEEL"))
        receipt_data = json.loads(receipt.read_text())
        self.assertEqual(receipt_data["release_verification"], "NOT_PERFORMED_BY_ASSEMBLER")
        self.assertIs(receipt_data["runtime_authorized"], False)

    def test_contract_drift_rejected(self) -> None:
        f = self.root / "src/ptsip/engines/contracts/installation-plan.json"
        data = json.loads(f.read_text())
        data["packages"][0]["installation"] = "EXPLICIT_LOCAL_OPERATOR_INSTALL_ONLY_NO_AUTOMATIC_EXTERNAL_INSTALL"
        write(self.root, str(f.relative_to(self.root)), data)
        with self.assertRaisesRegex(ValueError, "contract mismatch"):
            self.run_builder()

    def test_invalid_binary_and_source_are_rejected(self) -> None:
        self.binary.write_bytes(b"not an ELF" + b"x" * 300)
        with self.assertRaisesRegex(ValueError, "platform"):
            self.run_builder()
        self.binary.write_bytes(b"\x7fELF" + b"x" * 300)
        with self.assertRaisesRegex(ValueError, "source SHA"):
            self.run_builder(source_sha="unknown")

    def test_unknown_host_and_unregistered_kind_fail(self) -> None:
        with self.assertRaisesRegex(ValueError, "unsupported"):
            self.run_builder(target="aarch64-unknown-linux-gnu")
        with self.assertRaisesRegex(ValueError, "not registered"):
            self.run_builder(kind="unregistered")

    def test_duplicate_registration_fails_closed(self) -> None:
        f = self.root / "src/ptsip/engines/index.json"
        data = json.loads(f.read_text())
        data["official_engine_entries"][1]["engine_id"] = data["official_engine_entries"][0]["engine_id"]
        write(self.root, str(f.relative_to(self.root)), data)
        with self.assertRaisesRegex(ValueError, "duplicate"):
            self.run_builder()


if __name__ == "__main__":
    unittest.main()
