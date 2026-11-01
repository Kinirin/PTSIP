import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

import workflow


SHA = "a" * 40
TARGET = "developer/policy/RECORD/MPD-RECORD-0001.yaml"


class WorkflowTransportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "checkout"
        self.root.mkdir()
        (self.root / "unchanged.txt").write_text("preserve", encoding="utf-8")
        self.output = Path(self.temp.name) / "result"
        self.env = {"GITHUB_WORKSPACE": str(self.root),
                    "REVIEW_OUTPUT_DIRECTORY": str(self.output),
                    "REVIEW_TARGETS": json.dumps([TARGET]), "REVIEW_SOURCES": "[]",
                    "REVIEW_EXPECTED_SHA": SHA,
                    "GITHUB_OUTPUT": str(Path(self.temp.name) / "outputs"),
                    "GITHUB_STEP_SUMMARY": str(Path(self.temp.name) / "summary")}
        self.packet = {"format": "ptsip-non-normative-policy-review/v1",
                       "status": "NON_NORMATIVE_REVIEW", "mutation_authorized": False,
                       "schema_validation": "NOT_PERFORMED",
                       "targets": [{"path": TARGET, "field_findings": []}],
                       "sources": [], "file_sha256": {TARGET: "0" * 64}}
        self.commands = []
        self.helper_code = 0
        self.dirty = b""

    def invoke(self, command, **kwargs):
        self.commands.append((command, kwargs))
        if command[:3] == ["git", "rev-parse", "HEAD"]:
            return subprocess.CompletedProcess(command, 0, (SHA + "\n").encode(), b"")
        if command[:2] == ["git", "status"]:
            return subprocess.CompletedProcess(command, 0, self.dirty, b"")
        self.assertEqual(command[0], "go")
        self.assertNotIn("shell", kwargs)
        self.assertEqual(kwargs["env"]["PTSIP_PYTHON"], workflow.sys.executable)
        raw = json.dumps(self.packet, ensure_ascii=False).encode("utf-8")
        return subprocess.CompletedProcess(command, self.helper_code, raw, b"helper diagnostic\n")

    def execute(self):
        with contextlib.redirect_stdout(io.StringIO()):
            return workflow.run(self.env, self.invoke)

    def metadata(self):
        return json.loads((self.output / "run.json").read_text(encoding="utf-8"))

    def test_literal_arguments_unicode_and_checkout_are_preserved(self):
        self.env["GITHUB_ACTIONS"] = "true"
        self.env["GITHUB_SHA"] = SHA
        source = "developer/policy/legacy/a;echo-owned.yaml#/rules/value"
        self.env["REVIEW_SOURCES"] = json.dumps([source])
        self.packet["targets"][0]["title"] = chr(0xD55C) + chr(0xAE00)
        self.assertEqual(self.execute(), 0)
        command = self.commands[-1][0]
        self.assertEqual(command[-4:], ["--target", TARGET, "--source", source])
        raw = (self.output / "review.json").read_bytes()
        self.assertEqual(raw, json.dumps(self.packet, ensure_ascii=False).encode("utf-8"))
        self.assertNotIn(chr(0xFFFD), raw.decode("utf-8"))
        self.assertEqual(self.metadata()["status"], "NON_NORMATIVE_REVIEW")
        self.assertFalse(self.metadata()["mutation_authorized"])
        self.assertEqual(list(self.root.iterdir()), [self.root / "unchanged.txt"])
        outputs = Path(self.env["GITHUB_OUTPUT"]).read_text(encoding="utf-8")
        self.assertIn("source_sha=" + SHA, outputs)
        self.assertIn("artifact_name=policy-review-", outputs)

    def test_review_findings_are_not_reported_as_approval(self):
        self.packet["targets"][0].update(selection_problem="UNRESOLVED", field_findings=[{"path": "/x"}])
        self.assertEqual(self.execute(), 0)
        meta = self.metadata()
        self.assertEqual(meta["status"], "REVIEW_REQUIRED")
        self.assertEqual(meta["counts"]["unresolved_selections"], 1)
        self.assertEqual(meta["counts"]["field_findings"], 1)
        self.assertEqual(meta["schema_validation"], "NOT_PERFORMED")

    def test_sha_mismatch_prevents_helper_execution(self):
        for key in ["REVIEW_EXPECTED_SHA", "GITHUB_SHA"]:
            with self.subTest(key=key):
                self.env[key] = "b" * 40
                self.env["REVIEW_OUTPUT_DIRECTORY"] = str(self.output / key)
                self.commands.clear()
                self.assertEqual(self.execute(), 2)
                self.assertTrue(all(command[0] == "git" for command, _ in self.commands))
                self.assertEqual(json.loads((Path(self.env["REVIEW_OUTPUT_DIRECTORY"]) / "run.json").read_text())["status"], "FAIL")
                self.env.pop(key)

    def test_invalid_inputs_do_not_execute_commands(self):
        cases = [
            ("REVIEW_TARGETS", "not json"), ("REVIEW_TARGETS", "[]"),
            ("REVIEW_TARGETS", '[true]'), ("REVIEW_TARGETS", json.dumps([TARGET, TARGET])),
            ("REVIEW_TARGETS", '["../outside.yaml"]'),
            ("REVIEW_TARGETS", '["src/policy/RECORD/SFP-RECORD-0001.yaml"]'),
            ("REVIEW_SOURCES", '["legacy.yaml"]'),
            ("REVIEW_SOURCES", '["../outside.yaml#/rules"]'),
            ("REVIEW_EXPECTED_SHA", "main"),
        ]
        original = dict(self.env)
        for i, (key, value) in enumerate(cases):
            with self.subTest(key=key, value=value):
                self.env = dict(original, REVIEW_OUTPUT_DIRECTORY=str(self.output / str(i)))
                self.env[key] = value
                self.commands.clear()
                self.assertEqual(self.execute(), 2)
                self.assertFalse(self.commands)
                self.assertTrue((Path(self.env["REVIEW_OUTPUT_DIRECTORY"]) / "review.json").is_file())

    def test_helper_failure_preserves_diagnostics_and_fails_job(self):
        self.helper_code = 2
        self.packet = {"status": "FAIL", "problem": "POINTER_NOT_FOUND"}
        self.assertEqual(self.execute(), 2)
        self.assertEqual(self.metadata()["helper_exit_code"], 2)
        self.assertEqual(self.metadata()["status"], "FAIL")
        self.assertEqual((self.output / "stderr.log").read_bytes(), b"helper diagnostic\n")
        self.assertEqual(json.loads((self.output / "review.json").read_text()), self.packet)

    def test_dirty_actions_checkout_is_rejected_before_helper_execution(self):
        self.env["GITHUB_ACTIONS"] = "true"
        self.dirty = b" M developer/policy/changed.yaml\n"
        self.assertEqual(self.execute(), 2)
        self.assertEqual(self.metadata()["problem"], "ACTIONS_CHECKOUT_MUST_BE_CLEAN")
        self.assertTrue(all(command[0] == "git" for command, _ in self.commands))

    def test_unexpected_authority_claim_fails_transport(self):
        self.packet["mutation_authorized"] = True
        self.assertEqual(self.execute(), 2)
        self.assertEqual(self.metadata()["problem"], "REVIEW_AUTHORITY_BOUNDARY_CHANGED")
        self.assertFalse(self.metadata()["mutation_authorized"])

    def test_output_cannot_overwrite_checkout_or_previous_artifact(self):
        self.env["REVIEW_OUTPUT_DIRECTORY"] = str(self.root / "artifact")
        with self.assertRaisesRegex(ValueError, "OUTSIDE_CHECKOUT"):
            self.execute()
        self.env["REVIEW_OUTPUT_DIRECTORY"] = str(self.output)
        self.output.mkdir()
        marker = self.output / "keep.txt"
        marker.write_text("keep", encoding="utf-8")
        with self.assertRaises(FileExistsError):
            self.execute()
        self.assertEqual(marker.read_text(), "keep")


if __name__ == "__main__":
    unittest.main()
