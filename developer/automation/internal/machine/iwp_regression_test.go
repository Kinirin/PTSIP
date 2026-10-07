package machine

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func registeredIWPFixture(t *testing.T) (*Repository, Object, Object) {
	t.Helper()
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := iwpRegistry(r)
	if err != nil {
		t.Fatal(err)
	}
	recipe := Map(List(registry["tasks"])[0])
	refs := []any{}
	for _, raw := range List(Map(recipe["mutation"])["targets"]) {
		resolved, err := ValidateImplementationRef(r, Map(raw))
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, resolved)
	}
	for _, ref := range []Object{
		{"path": "src/ptsip/app/github_authority.py", "selector": Object{"kind": "PYTHON_FUNCTION", "name": "_global_decision_id"}},
		{"path": "src/ptsip/app/github_authority.py", "selector": Object{"kind": "PYTHON_METHOD", "class": "GithubControlPlaneClient", "method": "gate"}},
		{"path": "developer/automation/internal/machine/policy_domain_bridge.go", "selector": Object{"kind": "GO_METHOD", "type": "Repository", "method": "CurrentBranch"}},
		{"path": "developer/automation/internal/machine/iwp_packet.go", "selector": Object{"kind": "GO_FUNCTION", "name": "iwpMergeTargets"}},
	} {
		resolved, err := ValidateImplementationRef(r, ref)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, resolved)
	}
	resolver, err := NewResolver(r)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := resolver.Rule("PTSIP-AUT-007")
	if err != nil {
		t.Fatal(err)
	}
	context := Object{"scope": recipe["scope"], "policies": []any{}, "task_context": Object{"branch_context": Object{"actual": recipe["branch"]}, "implementation_refs": refs, "test_refs": Map(recipe["mutation"])["allowed_test_paths"], "planning_entry": iwpRegistryRef, "normative_rule_source": rule["canonical_source"], "normative_rule_refs": []any{rule["rule_id"]}, "normative_rules": []any{rule}, "constraints": Object{"fixture_only": true, "live_task_registration": false}}}
	packet, err := BuildWorkPacketFromResolution(r, registry, context, "MODIFY")
	if err != nil {
		t.Fatal(err)
	}
	return r, packet, context
}

func TestIWPUnregisteredLiveTaskStillFailsClosed(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewResolver(r)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolver.Resolve("src/ptsip/app/github_authority.py", "MODIFY")
	if err != nil {
		t.Fatal(err)
	}
	if resolved["task_context"] != nil {
		t.Fatal("live task registration inferred")
	}
	if _, err := BuildWorkPacket(r, "src/ptsip/app/github_authority.py", "MODIFY"); err == nil || !strings.Contains(err.Error(), "task_context must be a mapping") {
		t.Fatal(err)
	}
}

func TestIWPRegisteredWorkflowIdentityAndFailureRouting(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := iwpRegistry(r)
	if err != nil {
		t.Fatal(err)
	}
	if payload["schema_version"] != "ptsip-implementation-workflows/v2" || payload["projection_authority"] != false || agentInt(Map(payload["failure_routing"])["recheck_context_at"]) != 2 || agentInt(Map(payload["failure_routing"])["reresolve_scope_at"]) != 3 || len(List(payload["tasks"])) == 0 {
		t.Fatal(payload)
	}
}

func TestIWPRegisteredPacketPreservesMutationAcceptanceAndRegressionPlan(t *testing.T) {
	_, packet, _ := registeredIWPFixture(t)
	mutation := Map(packet["mutation_plan"])
	if packet["schema_version"] != "ptsip-implementation-work-packet/v2" || mutation["scope_expansion"] != "RE_RESOLVE_REQUIRED" || !iwpRegressionStringsEqual(mutation["allowed_test_paths"], []string{"src/tests/ptsip/control_plane/test_github_authority.py"}) {
		t.Fatal(packet)
	}
	if len(List(mutation["targets"])) != 2 {
		t.Fatal(mutation)
	}
	for _, raw := range List(mutation["targets"]) {
		if Text(Map(raw)["rationale"]) == "" {
			t.Fatal("missing mutation rationale")
		}
	}
	if len(List(packet["read_context"])) != 4 {
		t.Fatal(packet["read_context"])
	}
	verification := Map(packet["verification"])
	required := Strings(verification["required_new_pytest_nodes"])
	if len(List(verification["required_new_tests"])) != 3 || len(required) != 3 {
		t.Fatal(verification)
	}
	for _, node := range Strings(verification["missing_required_new_tests"]) {
		if !Has(required, node) {
			t.Fatal("unexpected missing test", node)
		}
	}
	if !Has([]string{"READY", "REQUIRES_NEW_TESTS"}, Text(verification["status"])) {
		t.Fatal(verification)
	}
	core := Strings(Map(verification["core_regression"])["pytest_targets"])
	combined := Strings(verification["combined_regression_pytest_targets"])
	for _, target := range []string{"src/tests/ptsip/control_plane", "src/tests/ptsip/test_proposed_component.py"} {
		if !Has(core, target) || !Has(combined, target) {
			t.Fatal("required core regression omitted", target)
		}
	}
	for _, redundant := range []string{"src/tests/ptsip/control_plane/test_github_authority.py", "src/tests/ptsip/control_plane/test_github_authority_reconciliation.py"} {
		if Has(combined, redundant) {
			t.Fatal("redundant child target", redundant)
		}
	}
	if got := Strings(Map(verification["commands"])["regression"]); len(got) < 3 || !iwpRegressionStringsEqual(got[:3], []string{"python", "-m", "pytest"}) {
		t.Fatal("consumer regression command changed", got)
	}
	coverage := map[string]Object{}
	for _, raw := range List(packet["acceptance_coverage"]) {
		item := Map(raw)
		coverage[Text(item["id"])] = item
	}
	if len(coverage) != 4 {
		t.Fatal(coverage)
	}
	for _, id := range []string{"GITHUB_PROPOSAL_RESOLUTION", "GITHUB_PROPOSAL_REPEAT", "NORMAL_GITHUB_DECISION_UNCHANGED", "LOCAL_RECEIPT_DOES_NOT_CHANGE_WINNER"} {
		if coverage[id] == nil {
			t.Fatal("acceptance vector missing", id)
		}
	}
	if !iwpRegressionStringsEqual(coverage["GITHUB_PROPOSAL_RESOLUTION"]["test_nodes"], []string{"src/tests/ptsip/control_plane/test_github_authority.py::test_github_proposal_resolution_returns_terminal_local_receipt"}) {
		t.Fatal(coverage)
	}
	mode := Map(packet["test_mode"])
	if mode["component_ref"] != "ptsip-core-verification" || mode["status"] != "REGISTERED" || mode["mode_id"] != "ptsip-core" || mode["fallback"] != nil {
		t.Fatal(mode)
	}
	freshness := Map(packet["freshness"])
	if freshness["strategy"] != "FILE_AND_SELECTOR_RECHECK_BEFORE_EVERY_VERIFICATION" || Has(Strings(freshness["context_files"]), "src/ptsip/app/github_authority.py") {
		t.Fatal(freshness)
	}
	count := 0
	for _, raw := range List(freshness["read_context_fingerprints"]) {
		row := Map(raw)
		if Text(row["path"]) == "src/ptsip/app/github_authority.py" {
			count++
			if len(Text(row["fingerprint"])) != 64 {
				t.Fatal(row)
			}
		}
	}
	if count != 2 {
		t.Fatal("same-file read context lost", count)
	}
}

func TestIWPCoreRegressionComesFromCanonicalProfile(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	targets, err := iwpComponentTargets(r, "ptsip-core-verification", "developer/profiles/ptsip-repository.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"src/tests/ptsip/control_plane", "src/tests/ptsip/identity", "src/tests/ptsip/test_proposed_component.py"} {
		if !Has(targets, expected) {
			t.Fatal(expected, targets)
		}
	}
	for _, target := range targets {
		if !strings.HasPrefix(target, "src/tests/") {
			t.Fatal(target)
		}
	}
}

func TestIWPWorkflowProfilesHaveNoRetiredRootBridgeDependency(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	if iwpPathExists(r, "ptsip.yaml") {
		t.Fatal("retired root bridge reintroduced")
	}
	if !iwpPathExists(r, "developer/profiles/ptsip-repository.yaml") {
		t.Fatal("self-profile missing")
	}
	registry, err := iwpRegistry(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(registry["tasks"]) {
		if Map(Map(Map(raw)["verification"])["core_regression"])["source"] != "developer/profiles/ptsip-repository.yaml" {
			t.Fatal(raw)
		}
	}
}

func TestIWPMutationSelectorsResolveCurrentSourceTargets(t *testing.T) {
	r, packet, _ := registeredIWPFixture(t)
	if failures := iwpSelectorIntegrity(r, Map(packet["mutation_plan"])["targets"]); len(failures) != 0 {
		t.Fatal(failures)
	}
}

func TestIWPHunkScopeGuardPreservesExactRanges(t *testing.T) {
	ranges := [][2]int{{100, 120}, {200, 230}}
	for _, test := range []struct {
		start, length int
		valid         bool
	}{{105, 3, true}, {120, 0, true}, {150, 2, false}, {99, 3, false}} {
		if iwpHunkAllowed(test.start, test.length, ranges) != test.valid {
			t.Fatal(test)
		}
	}
}

func TestIWPCheckBlocksUnlistedChangesWithoutExpandingScope(t *testing.T) {
	r, packet := iwpFixture(t)
	if err := r.AtomicWrite("README.md", []byte("unlisted fixture change\n"), nil); err != nil {
		t.Fatal(err)
	}
	result, err := CheckWorkPacket(r, packet)
	if err != nil || result["status"] != "BLOCKED" || !Has(Strings(result["problems"]), "UNEXPECTED_CHANGED_PATH") || !iwpRegressionStringsEqual(result["unexpected_changed_paths"], []string{"README.md"}) || result["reprepare_required"] != false || result["safe_to_continue_iteration"] != false || len(Text(result["iteration_fingerprint"])) != 64 {
		t.Fatal(result, err)
	}
}

func TestIWPRepeatedFailuresEscalateWithoutScopeExpansion(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	policy := Object{"recheck_context_at": 2, "reresolve_scope_at": 3}
	for i, expected := range []string{"FIX_WITHIN_CURRENT_MUTATION_PLAN", "RECHECK_PACKET_ACCEPTANCE_AND_CONTEXT", "RE_RESOLVE_MUTATION_SCOPE"} {
		result, err := RouteWorkPacketFailure(r, "iwp-example", "focused", 1, "FAILED test_example.py::test_case - AssertionError: expected terminal state\n", "failure-state.json", policy)
		if err != nil || agentInt(result["repeat_count"]) != i+1 || result["next_action"] != expected || result["scope_expansion_allowed"] != false {
			t.Fatal(result, err)
		}
	}
}

func TestIWPCollectionFailureRoutesToTestContractRepair(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	result, err := RouteWorkPacketFailure(r, "iwp-example", "focused", 4, "ERROR collecting tests/ptsip/control_plane/test_github_authority.py\n", "failure-state.json", Object{"recheck_context_at": 2, "reresolve_scope_at": 3})
	if err != nil || result["classification"] != "TEST_CONTRACT_FAILURE" || result["next_action"] != "REPAIR_TEST_SELECTION_OR_REQUIRED_TEST" {
		t.Fatal(result, err)
	}
}

func TestIWPAgentBriefIsCompactAndProgressive(t *testing.T) {
	_, packet, resolved := registeredIWPFixture(t)
	brief, err := BuildWorkPacketBrief(packet)
	if err != nil {
		t.Fatal(err)
	}
	if brief["schema_version"] != "ptsip-agent-implementation-brief/v1" || brief["projection_authority"] != false || brief["packet_id"] != packet["packet_id"] || len(List(brief["normative_rules"])) != len(List(Map(resolved["task_context"])["normative_rules"])) {
		t.Fatal(brief)
	}
	for _, raw := range List(brief["normative_rules"]) {
		row := Map(raw)
		if row["title"] == nil || row["section_text"] != nil {
			t.Fatal(row)
		}
	}
	if Map(brief["verification"])["commands"] != nil || Map(Map(brief["verification"])["core_regression"])["pytest_targets"] != nil || Map(brief["freshness"])["file_hashes"] != nil {
		t.Fatal("large verification payload leaked", brief)
	}
	command := Strings(Map(brief["on_demand"])["normative_rule"])
	if len(command) == 0 || command[0] != "go" {
		t.Fatal("brief does not use native Go entry", command)
	}
	packetBytes, err := CanonicalJSON(packet)
	if err != nil {
		t.Fatal(err)
	}
	briefBytes, err := CanonicalJSON(brief)
	if err != nil {
		t.Fatal(err)
	}
	if len(briefBytes)*2 >= len(packetBytes) {
		t.Fatalf("brief=%d packet=%d", len(briefBytes), len(packetBytes))
	}
}

func TestIWPMutationSourceContextAvoidsWholeFileReads(t *testing.T) {
	r, packet, _ := registeredIWPFixture(t)
	context, err := BuildWorkPacketSourceContext(r, packet, "mutation", "")
	if err != nil {
		t.Fatal(err)
	}
	items := List(context["items"])
	if context["schema_version"] != "ptsip-agent-source-context/v1" || context["role"] != "mutation" || len(items) != 2 {
		t.Fatal(context)
	}
	total := 0
	for _, raw := range items {
		row := Map(raw)
		if len(Text(row["source_sha256"])) != 64 {
			t.Fatal(row)
		}
		total += len([]byte(Text(row["source"])))
	}
	path, err := r.Path("src/ptsip/app/github_authority.py")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(total)*5 >= info.Size() {
		t.Fatalf("projected=%d whole=%d", total, info.Size())
	}
}

func TestIWPProjectionJSONWriterUsesUTF8(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	payload := Object{"text": "authority — local projection"}
	if err := r.WriteJSON("brief.json", payload, nil); err != nil {
		t.Fatal(err)
	}
	path, err := r.Path("brief.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(raw, []byte{0xff, 0xfe}) || !bytes.Contains(raw, []byte("—")) {
		t.Fatal("projection encoding changed")
	}
	var decoded Object
	if err := json.Unmarshal(raw, &decoded); err != nil || !reflect.DeepEqual(decoded, payload) {
		t.Fatal(decoded, err)
	}
}

func TestIWPReadContextSupportsOneStableSelectorProjection(t *testing.T) {
	r, packet, _ := registeredIWPFixture(t)
	first, err := BuildWorkPacketBrief(packet)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildWorkPacketBrief(packet)
	if err != nil {
		t.Fatal(err)
	}
	refs := List(first["read_context"])
	if len(refs) != 4 {
		t.Fatal(refs)
	}
	ids := []string{}
	for _, raw := range refs {
		id := Text(Map(raw)["context_id"])
		if !strings.HasPrefix(id, "read-") || Has(ids, id) {
			t.Fatal(id)
		}
		ids = append(ids, id)
	}
	if !reflect.DeepEqual(first["read_context"], second["read_context"]) {
		t.Fatal("read context IDs are unstable")
	}
	command := Strings(Map(first["on_demand"])["read_context"])
	if !Has(command, "--context-id") || !Has(command, "<CONTEXT_ID>") {
		t.Fatal(command)
	}
	all, err := BuildWorkPacketSourceContext(r, packet, "read", "")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := BuildWorkPacketSourceContext(r, packet, "read", ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(List(all["items"])) != 4 || selected["requested_context_id"] != ids[0] || len(List(selected["items"])) != 1 || Map(List(selected["items"])[0])["context_id"] != ids[0] {
		t.Fatal(selected)
	}
	total := 0
	for _, raw := range List(all["items"]) {
		total += len(Text(Map(raw)["source"]))
	}
	if len(Text(Map(List(selected["items"])[0])["source"])) >= total {
		t.Fatal("single selector did not bound source read")
	}
}

func iwpRegressionStringsEqual(value any, expected []string) bool {
	return reflect.DeepEqual(Strings(value), expected)
}
