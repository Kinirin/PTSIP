package schema_check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

func routingRepositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPolicyRoutingContractInputs(t *testing.T) {
	root := routingRepositoryRoot(t)
	doc, schema, resources, registered, refs, sources, err := loadRoutingInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	if routingString(doc["schema_version"]) != "ptsip-policy-routing/v5" {
		t.Fatal("routing version mismatch")
	}
	if routingString(schema["$id"]) != "urn:ptsip:contract:policy-routing:v5" {
		t.Fatal("schema ID mismatch")
	}
	if len(resources) != 1 || len(sources) != 1 {
		t.Fatalf("missing identity sources: %d/%d", len(resources), len(sources))
	}
	if registered != 28 || refs != 2 {
		t.Fatalf("route coverage or schema reference count: %d/%d", registered, refs)
	}
}

func TestPolicyRoutingCannotUseRetiredOrUnsafePaths(t *testing.T) {
	root := routingRepositoryRoot(t)
	for _, file := range []string{
		".ptsip/contracts/routing/policy_routes.json",
		".ptsip/contracts/policy/routing/../identity/policy_kind.json",
		".ptsip/contracts/policy/routing/nonexistent.json",
		".ptsip/contracts/policy/routing/../../outside.json",
		".ptsip/contracts/policy/routing\\policy_routes.json",
		"/etc/passwd",
	} {
		if _, err := routingFile(root, file, ".ptsip/contracts/policy/routing/"); err == nil {
			t.Errorf("unexpected acceptance: %s", file)
		}
	}
}

func TestPolicyRoutingRejectsRetiredBinding(t *testing.T) {
	root := t.TempDir()
	boundary := filepath.Join(root, ".ptsip", "contracts", "policy", "boundary.json")
	if err := os.MkdirAll(filepath.Dir(boundary), 0755); err != nil {
		t.Fatal(err)
	}
	corrupted := `{"domain":"policy","namespaces":{"routing":"routing/"},"routing_binding":{"data":".ptsip/contracts/routing/policy_routes.json","schema":".ptsip/contracts/policy/routing/Policy_Routing.schema.json"}}`
	if err := os.WriteFile(boundary, []byte(corrupted), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, _, err := loadRoutingInputs(root)
	if err == nil {
		t.Fatal("retired routing path was accepted")
	}
	if !strings.Contains(err.Error(), "INVALID_ROUTING_CONTRACT_PATH") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestPolicyRoutingRequiresBoundSchemaFile(t *testing.T) {
	root := t.TempDir()
	boundary := filepath.Join(root, ".ptsip", "contracts", "policy", "boundary.json")
	if err := os.MkdirAll(filepath.Dir(boundary), 0755); err != nil {
		t.Fatal(err)
	}
	canonical := `{"domain":"policy","namespaces":{"routing":"routing/"},"routing_binding":{"data":".ptsip/contracts/policy/routing/policy_routes.json","schema":".ptsip/contracts/policy/routing/Policy_Routing.schema.json"}}`
	if err := os.WriteFile(boundary, []byte(canonical), 0644); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, filepath.FromSlash(policyRoutingData))
	if err := os.MkdirAll(filepath.Dir(data), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte(`{"identity_sources":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, _, err := loadRoutingInputs(root)
	if err == nil || !strings.Contains(err.Error(), "ROUTING_RESOURCE_MISSING") {
		t.Fatalf("missing routing schema must fail closed: %v", err)
	}
}

func TestV2PublicPolicySelectedWithoutV7Fallback(t *testing.T) {
	root := routingRepositoryRoot(t)
	public, err := routingFile(root, ".ptsip/contracts/policy/PTSIP_Public_Policy_Structure.schema.json", ".ptsip/contracts/policy/")
	if err != nil {
		t.Fatal(err)
	}
	if routingString(public["$id"]) != "urn:ptsip:candidate:contract:policy:public-structure:v2" {
		t.Fatal("v2 public schema not registered")
	}
	ruleRef := routingString(routingMembers(routingMembers(public["properties"])["Policy_Rules"])["$ref"])
	if ruleRef != "urn:ptsip:candidate:contract:policy:rules:v2" {
		t.Fatalf("wrong Policy_Rules schema: %s", ruleRef)
	}
	route, _, _, _, _, _, err := loadRoutingInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	refs := routingMembers(route["schema_refs"])
	if routingString(refs["public"]) != ".ptsip/contracts/policy/PTSIP_Public_Policy_Structure.schema.json" ||
		routingString(refs["rules"]) != ".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json" {
		t.Fatalf("v7 fallback in active route: %v", refs)
	}
}

func TestGOVPolicyDocumentsUseCandidateRootAndPreserveLifecycle(t *testing.T) {
	root := routingRepositoryRoot(t)
	for _, id := range []string{"0001", "0004", "0005", "0006"} {
		relative := "developer/policy/GOV/MPD-GOV-" + id + ".yaml"
		document, err := authoring.ReadDocument(root, relative)
		if err != nil {
			t.Fatal(err)
		}
		if !authoring.IsUnifiedPolicy(document) {
			t.Fatalf("%s is not in the current common structure", relative)
		}
		for _, field := range []string{"Policy_Identity", "Policy_Lifecycle", "Policy_Authority",
			"Policy_Rules", "Policy_Evaluation", "Policy_Relationships", "Policy_Content", "Policy_Migration"} {
			if _, ok := document[field]; !ok {
				t.Fatalf("%s missing common field %s", relative, field)
			}
		}
		source, err := authoring.RestoreUnifiedSource(root, document)
		if err != nil {
			t.Fatal(err)
		}
		lifecycle := source["Policy_Lifecycle"].(map[string]any)
		if lifecycle["status"] != "APPROVED.Applying" || lifecycle["Policy_Status_Schema_ver"] == nil ||
			!strings.HasPrefix(lifecycle["contract"].(string), "urn:ptsip:contract:policy:gov:") {
			t.Errorf("%s original GOV lifecycle changed: %v", relative, lifecycle)
		}
	}
}

func TestV2FamilySelectionSharesRulesAndDoesNotGrantAuthority(t *testing.T) {
	root := routingRepositoryRoot(t)
	routing, _, _, registered, refs, _, err := loadRoutingInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	if registered != 28 || refs != 2 {
		t.Fatalf("unexpected coverage: %d/%d", registered, refs)
	}
	developer := routingMembers(routingMembers(routing["routes"])["PTSIP_DEVELOPER_POLICY"])
	if len(developer) != 1 || developer["output_root"] != "developer/policy" {
		t.Fatalf("developer route has inferred authority: %v", developer)
	}
	selection, err := routingFile(root, ".ptsip/contracts/policy/Policy_Rules/Family_Selection.json", ".ptsip/contracts/policy/")
	if err != nil {
		t.Fatal(err)
	}
	if selection["active"] != true {
		t.Fatal("v2 selection inactive")
	}
	fam := routingMembers(selection["family_routes"])
	arch := routingMembers(fam["ARCH"])
	if arch["rules_schema_ref"] != "urn:ptsip:candidate:contract:policy:rules:v2" {
		t.Fatal("ARCH v2 family not bound")
	}
}
