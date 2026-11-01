import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("changed", Path(__file__).with_name("changed.py"))
changed = importlib.util.module_from_spec(spec)
spec.loader.exec_module(changed)


class ChangeGuardTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.git("init", "-q")
        self.git("config", "user.name", "Guard test")
        self.git("config", "user.email", "guard@example.invalid")
        self.put("README.md", "base")
        self.base = self.commit()
        self.calls = []

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.root, capture_output=True, check=True).stdout.decode().strip()

    def put(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def commit(self):
        self.git("add", ".")
        self.git("commit", "-qm", "test")
        return self.git("rev-parse", "HEAD")

    def invoke(self, args, **kwargs):
        if args[0] != "go":
            return subprocess.run(args, **kwargs)
        self.calls.append(args)
        target = args[-1]
        result = {"status": "CHECK_COMPLETED", "selection": {
            "target": target, "mutation_authorized": False, "execution_authorized": False},
            "result": {"status": "VALID"}}
        return subprocess.CompletedProcess(args, 0, json.dumps(result).encode(), b"")

    def test_exact_changed_policy_uses_registered_entry(self):
        target = "developer/policy/RECORD/new.yaml"
        self.put(target, "policy")
        head = self.commit()
        result, code = changed.check(self.root, self.base, head, self.invoke)
        self.assertEqual(code, 0, result)
        self.assertEqual(result["source_sha"], head)
        self.assertEqual(result["targets"][0]["path"], target)
        self.assertIn("contract-entry", self.calls[0])

    def test_retired_addition_and_rename_are_rejected(self):
        self.put(".ptsip/contracts/policy/Rules/RECORD_rules.json", "{}")
        self.commit()
        result, code = changed.check(self.root, self.base, "HEAD", self.invoke)
        self.assertEqual(code, 2)
        self.assertEqual(result["problem"], "RETIRED_POLICY_CONTRACT_WRITE")
        self.assertEqual(self.calls, [])

    def test_historical_sources_and_deletions_are_not_authoring(self):
        old = ".ptsip/contracts/policy/Rules/RECORD_rules.json"
        self.put(old, "historical")
        before = self.commit()
        (self.root / old).unlink()
        self.put("developer/policy/legacy/old.yaml", "historical")
        self.put("developer/policy/index.yaml", "lookup")
        self.commit()
        result, code = changed.check(self.root, before, "HEAD", self.invoke)
        self.assertEqual(code, 0, result)
        self.assertEqual(self.calls, [])

    def test_rename_into_retired_tree_cannot_hide_as_rename(self):
        destination = self.root / ".ptsip/contracts/policy/Rules/copied.json"
        destination.parent.mkdir(parents=True)
        (self.root / "README.md").rename(destination)
        self.commit()
        result, code = changed.check(self.root, self.base, "HEAD", self.invoke)
        self.assertEqual((code, result["problem"]), (2, "RETIRED_POLICY_CONTRACT_WRITE"))

    def test_dirty_checkout_and_wrong_head_fail_before_validation(self):
        self.put("new.txt", "uncommitted")
        result, code = changed.check(self.root, self.base, "HEAD", self.invoke)
        self.assertEqual((code, result["problem"]), (2, "CLEAN_CHECKOUT_REQUIRED"))
        self.commit()
        result, code = changed.check(self.root, self.base, self.base, self.invoke)
        self.assertEqual((code, result["problem"]), (2, "CHECKOUT_HEAD_MISMATCH"))
        self.assertEqual(self.calls, [])

    def test_failed_or_unexpected_validation_fails_closed(self):
        self.put("developer/policy/RECORD/new.yaml", "legacy")
        self.commit()
        def fail(args, **kwargs):
            if args[0] == "go":
                return subprocess.CompletedProcess(args, 2, b'{}', b'old structure rejected')
            return subprocess.run(args, **kwargs)
        result, code = changed.check(self.root, self.base, "HEAD", fail)
        self.assertEqual(code, 2)
        self.assertIn("old structure rejected", result["problem"])
        def unexpected(args, **kwargs):
            if args[0] == "go":
                return subprocess.CompletedProcess(args, 0, b'{"status":"VALID"}', b'')
            return subprocess.run(args, **kwargs)
        result, code = changed.check(self.root, self.base, "HEAD", unexpected)
        self.assertEqual(code, 2)
        self.assertIn("REGISTERED_POLICY_CHECK_REQUIRED", result["problem"])

    def test_no_ref_or_path_injection(self):
        for ref in ["--help", "HEAD~1", "main; ignored", ""]:
            result, code = changed.check(self.root, ref, "HEAD", self.invoke)
            self.assertEqual((code, result["problem"]), (2, "EXACT_BASE_SHA_REQUIRED"))
        for path in ["../escape.yaml", "x/../y", "/absolute", "C:/escape", "x\ny"]:
            with self.assertRaises(ValueError):
                changed.classify([path])


if __name__ == "__main__":
    unittest.main()
