package machine

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestIWPGrammarTypedSelectors(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	source := "@decorator\nasync def target(value: str = 'class fake:') -> str:\n    def nested():\n        pass\n    return value\n\nclass Worker:\n    @staticmethod\n    async def method():\n        return 1\n\ndef main(args):\n    if args.command == 'first':\n        return 1\n    elif args.command == 'second':\n        return 2\n    else:\n        return 3\n# trailing comment\n"
	_ = r.AtomicWrite("sample.py", []byte(source), nil)
	cases := []struct {
		selector   Object
		start, end int
	}{{Object{"kind": "PYTHON_FUNCTION", "name": "target"}, 2, 5}, {Object{"kind": "PYTHON_METHOD", "class": "Worker", "method": "method"}, 9, 10}, {Object{"kind": "CLI_COMMAND_BRANCH", "command": "first"}, 13, 18}, {Object{"kind": "CLI_COMMAND_BRANCH", "command": "second"}, 15, 18}, {Object{"kind": "PYTHON_MODULE"}, 1, 19}}
	for _, item := range cases {
		resolved, err := ValidateImplementationRef(r, Object{"path": "sample.py", "selector": item.selector})
		if err != nil {
			t.Fatal(item.selector, err)
		}
		location := Map(resolved["resolved_location"])
		if agentInt(location["line_start"]) != item.start || agentInt(location["line_end"]) != item.end {
			t.Fatalf("%v range=%v want=%d:%d", item.selector, location, item.start, item.end)
		}
	}
	for _, selector := range []Object{{"kind": "PYTHON_FUNCTION", "name": "nested"}, {"kind": "PYTHON_METHOD", "class": "Missing", "method": "method"}, {"kind": "CLI_COMMAND_BRANCH", "command": "unknown"}, {"kind": "FAKE", "name": "target"}} {
		if _, err := ValidateImplementationRef(r, Object{"path": "sample.py", "selector": selector}); err == nil {
			t.Fatal("unsupported or nonexact selector accepted", selector)
		}
	}
	_ = r.AtomicWrite("duplicate.py", []byte("def target():\n    pass\ndef target():\n    pass\n"), nil)
	if _, err := ValidateImplementationRef(r, Object{"path": "duplicate.py", "selector": Object{"kind": "PYTHON_FUNCTION", "name": "target"}}); err == nil {
		t.Fatal("duplicate function accepted")
	}
	_ = r.AtomicWrite("invalid.py", []byte("def broken(:\n    pass\n"), nil)
	if _, err := ValidateImplementationRef(r, Object{"path": "invalid.py", "selector": Object{"kind": "PYTHON_MODULE"}}); err == nil {
		t.Fatal("syntax error accepted")
	}
	if _, err := ValidateImplementationRef(r, Object{"path": "../sample.py", "selector": Object{"kind": "PYTHON_MODULE"}}); err == nil {
		t.Fatal("path escape accepted")
	}
}
func TestIWPGrammarTestNodesAndGoSelectors(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	_ = r.AtomicWrite("test_sample.py", []byte("def test_top():\n    pass\nclass TestExample:\n    async def test_method(self):\n        pass\n"), nil)
	for _, node := range []string{"test_sample.py::test_top", "test_sample.py::TestExample::test_method"} {
		if !PythonTestNodeExists(r, node) {
			t.Fatal("registered test not found", node)
		}
	}
	for _, node := range []string{"test_sample.py", "test_sample.py::missing", "test_sample.py::test_top::test_method"} {
		if PythonTestNodeExists(r, node) {
			t.Fatal("nonexact test accepted", node)
		}
	}
	_ = r.AtomicWrite("sample.go", []byte("package example\n\ntype Worker struct{}\nfunc Read() {}\nfunc (w *Worker) Write() {}\n"), nil)
	for _, selector := range []Object{{"kind": "GO_FUNCTION", "name": "Read"}, {"kind": "GO_METHOD", "type": "Worker", "method": "Write"}, {"kind": "GO_TYPE", "name": "Worker"}, {"kind": "GO_MODULE"}} {
		if _, err := ValidateImplementationRef(r, Object{"path": "sample.go", "selector": selector}); err != nil {
			t.Fatal(selector, err)
		}
	}
	if _, err := ValidateImplementationRef(r, Object{"path": "sample.go", "selector": Object{"kind": "GO_FUNCTION", "name": "Write"}}); err == nil {
		t.Fatal("method admitted as top-level function")
	}
}

func TestIWPNativeIdentifierAudit(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	cases := []struct {
		source string
		found  bool
	}{{"text = 'retired'\n# retired\n", false}, {"value = object.retired\n", false}, {"class retired:\n    pass\n", false}, {"def current(retired: str = 'retired'):\n    pass\n", false}, {"from package import retired\n", false}, {"async def retired():\n    pass\n", true}, {"value = retired()\n", true}, {"retired = 1\n", true}}
	for i, item := range cases {
		_ = r.AtomicWrite("source.py", []byte(item.source), nil)
		found, err := PythonIdentifierPresent(r, "source.py", "retired")
		if err != nil || found != item.found {
			t.Fatalf("case %d found=%v err=%v", i, found, err)
		}
	}
}
func TestIWPFailureEscalationAndNormalization(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	policy := Object{"recheck_context_at": 2, "reresolve_scope_at": 3}
	for i, action := range []string{"FIX_WITHIN_CURRENT_MUTATION_PLAN", "RECHECK_PACKET_ACCEPTANCE_AND_CONTEXT", "RE_RESOLVE_MUTATION_SCOPE"} {
		result, err := RouteWorkPacketFailure(r, "packet", "focused", 1, "Failed in 1.2s at 0x12", ".git/failure.json", policy)
		if err != nil || result["next_action"] != action || agentInt(result["repeat_count"]) != i+1 {
			t.Fatal(result, err)
		}
	}
	if WorkPacketFailureSignature("focused", 1, "Failed in 1.2s at 0x12") != WorkPacketFailureSignature("focused", 1, "Failed in 2.9s at 0x34") {
		t.Fatal("duration/address destabilize failure routing")
	}
	result, err := RouteWorkPacketFailure(r, "packet", "focused", 1, "ERROR collecting sample.py", ".git/failure.json", policy)
	if err != nil || result["classification"] != "TEST_CONTRACT_FAILURE" {
		t.Fatal(result, err)
	}
	if err := ClearWorkPacketFailure(r, ".git/failure.json", "packet", "focused"); err != nil {
		t.Fatal(err)
	}
	state, err := iwpFailureState(r, ".git/failure.json")
	if err != nil || len(Map(state["entries"])) != 0 {
		t.Fatal(state, err)
	}
}

func iwpFixture(t *testing.T) (*Repository, Object) {
	t.Helper()
	r := &Repository{Root: t.TempDir()}
	files := map[string]string{"src/sample.py": "def edit():\n    return 1\n\ndef read():\n    return 2\n", "src/tests/test_sample.py": "def test_baseline():\n    pass\ndef test_new():\n    pass\n", "developer/planning/task.yaml": "state: ACTIVE\n", "developer/rules.yaml": "rules: []\n", iwpRegistryRef: "tasks: []\n", iwpSchemaRef: "{}\n"}
	for ref, text := range files {
		if err := r.AtomicWrite(ref, []byte(text), nil); err != nil {
			t.Fatal(err)
		}
	}
	profile := Object{"components": []any{Object{"id": "core", "roles": []any{"VERIFICATION"}, "include": []any{"src/tests/**"}}}}
	_ = r.WriteYAML("developer/profile.yaml", profile, nil)
	commands := [][]string{{"init", "--quiet"}, {"config", "user.email", "fixture@example.invalid"}, {"config", "user.name", "Go Fixture"}, {"add", "."}, {"commit", "--quiet", "-m", "fixture"}}
	for _, args := range commands {
		command := exec.Command("git", append([]string{"-C", r.Root}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatal(string(output), err)
		}
	}
	branch, err := iwpGit(r, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	edit, err := ValidateImplementationRef(r, Object{"path": "src/sample.py", "selector": Object{"kind": "PYTHON_FUNCTION", "name": "edit"}})
	if err != nil {
		t.Fatal(err)
	}
	read, err := ValidateImplementationRef(r, Object{"path": "src/sample.py", "selector": Object{"kind": "PYTHON_FUNCTION", "name": "read"}})
	if err != nil {
		t.Fatal(err)
	}
	baseline, newNode := "src/tests/test_sample.py::test_baseline", "src/tests/test_sample.py::test_new"
	recipe := Object{"branch": branch, "scope": "src/sample.py", "operation": "MODIFY", "mutation": Object{"targets": []any{Object{"path": "src/sample.py", "selector": edit["selector"], "rationale": "Fix edit"}}, "allowed_test_paths": []any{"src/tests/test_sample.py"}, "scope_expansion": "RE_RESOLVE_REQUIRED"}, "acceptance_vectors": []any{Object{"id": "A1", "test_nodes": []any{newNode}, "invariants": []any{"Read remains stable"}}}, "verification": Object{"baseline_pytest_nodes": []any{baseline}, "task_regression_pytest_targets": []any{"src/tests/test_sample.py"}, "full_command": []any{"go", "version"}, "required_new_tests": []any{Object{"node": newNode, "acceptance_ids": []any{"A1"}}}, "core_regression": Object{"component_ref": "core", "source": "developer/profile.yaml", "selection": "CANONICAL"}}}
	registry := Object{"tasks": []any{recipe}, "failure_routing": Object{"recheck_context_at": 2, "reresolve_scope_at": 3}}
	resolved := Object{"scope": "src/sample.py", "policies": []any{}, "task_context": Object{"branch_context": Object{"actual": branch}, "implementation_refs": []any{edit, read}, "test_refs": []any{"src/tests/test_sample.py"}, "normative_rule_refs": []any{}, "normative_rules": []any{}, "constraints": Object{}, "planning_entry": "developer/planning/task.yaml", "normative_rule_source": "developer/rules.yaml"}}
	packet, err := BuildWorkPacketFromResolution(r, registry, resolved, "MODIFY")
	if err != nil {
		t.Fatal(err)
	}
	return r, packet
}
func TestIWPBuildCheckBriefAndSourceContext(t *testing.T) {
	r, packet := iwpFixture(t)
	checked, err := CheckWorkPacket(r, packet)
	if err != nil || checked["status"] != "PASS" {
		t.Fatal(checked, err)
	}
	brief, err := BuildWorkPacketBrief(packet)
	if err != nil {
		t.Fatal(err)
	}
	read := List(brief["read_context"])
	if len(read) != 1 {
		t.Fatal("read context leaked or missing")
	}
	id := Text(Map(read[0])["context_id"])
	context, err := BuildWorkPacketSourceContext(r, packet, "read", id)
	if err != nil {
		t.Fatal(err)
	}
	if len(List(context["items"])) != 1 || !strings.Contains(Text(Map(List(context["items"])[0])["source"]), "def read") {
		t.Fatal(context)
	}
	if _, err := BuildWorkPacketSourceContext(r, packet, "read", "unknown"); err == nil {
		t.Fatal("unknown context selector admitted")
	}
	result, err := VerifyWorkPacket(r, packet, "full", "", "")
	if err != nil || result["status"] != "PASS" {
		t.Fatal(result, err)
	}
	_ = r.AtomicWrite("src/sample.py", []byte("def edit():\n    return 3\n\ndef read():\n    return 2\n"), nil)
	checked, err = CheckWorkPacket(r, packet)
	if err != nil || checked["status"] != "PASS" {
		t.Fatal("allowed code edit blocked", checked, err)
	}
	_ = r.AtomicWrite("src/sample.py", []byte("def edit():\n    return 3\n\ndef read():\n    return 4\n"), nil)
	checked, err = CheckWorkPacket(r, packet)
	if err != nil || !Has(Strings(checked["problems"]), "READ_CONTEXT_CHANGED") || !Has(Strings(checked["problems"]), "CODE_SCOPE_VIOLATION") {
		t.Fatal("out-of-selector edit admitted", checked, err)
	}
	_ = r.AtomicWrite("unlisted.txt", []byte("data"), nil)
	checked, err = CheckWorkPacket(r, packet)
	if err != nil || !Has(Strings(checked["problems"]), "UNEXPECTED_CHANGED_PATH") {
		t.Fatal("unlisted path admitted", checked, err)
	}
}
func TestIWPGuardHunksAndWireIDs(t *testing.T) {
	ranges := [][2]int{{10, 20}}
	for _, span := range [][2]int{{10, 2}, {9, 0}, {20, 0}} {
		if !iwpHunkAllowed(span[0], span[1], ranges) {
			t.Fatal("allowed hunk rejected")
		}
	}
	for _, span := range [][2]int{{9, 2}, {21, 1}, {21, 0}} {
		if iwpHunkAllowed(span[0], span[1], ranges) {
			t.Fatal("outside hunk admitted")
		}
	}
	data, err := iwpCanonicalJSON(Object{"name": "정책"})
	if err != nil || string(data) != "{\"name\":\"\\uc815\\ucc45\"}" {
		t.Fatal(string(data), err)
	}
	if !reflect.DeepEqual(iwpSortedUnique([]string{"b", "a", "b"}), []string{"a", "b"}) {
		t.Fatal("deterministic ordering failed")
	}
}
