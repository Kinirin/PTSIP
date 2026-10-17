package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, rootFamilyRef, rootFamilyJSON string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"developer/contracts/routing/policy_routes.json": `{"identity_sources":{"root_family":``}`,
		"developer/contracts/identity/root_family.json": rootFamilyJSON,
		"developer/contracts/identity/policy_kind.json": `{"$id":"urn:example:kind","type":"string","enum":["TYPE_A"]}`,
	}
	routing := map[string]any{
		"identity_sources": map[string]any{
			"root_family": rootFamilyRef,
			"policy_kind": "developer/contracts/identity/policy_kind.json",
		},
	}
	data, err := json.Marshal(routing)
	if err != nil {
		t.Fatal(err)
	}
	files["developer/contracts/routing/policy_routes.json"] = string(data)
	for name, value := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestRepositoryFamilyMembershipIsContractDriven(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	domain, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(domain.RootFamilies()); got != 14 {
		t.Fatalf("repository contract currently declares 14 families, got %d", got)
	}
	if !domain.HasRootFamily("ARCH") || domain.HasRootFamily("UNLISTED_FAMILY") {
		t.Fatal("family membership differs from current contract")
	}
	if !domain.HasPolicyKind("PTSIP_DEVELOPER_POLICY") {
		t.Fatal("policy_kind was not loaded from declared contract")
	}
}

func TestDomainDoesNotHardcodeOldFamilyListOrRequireAllRoutes(t *testing.T) {
	root := fixture(t,
		"developer/contracts/identity/root_family.json",
		`{"$id":"urn:example:family","type":"string","enum":["OMEGA","BETA"]}`,
	)
	domain, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !domain.HasRootFamily("OMEGA") || domain.HasRootFamily("ARCH") {
		t.Fatal("membership must follow declared contract, not the old RootID regular expression")
	}
	if domain.HasPolicyKind("PTSIP_DEVELOPER_POLICY") || !domain.HasPolicyKind("TYPE_A") {
		t.Fatal("policy kind must be read from the declared contract")
	}
	copyOfFamilies := domain.RootFamilies()
	copyOfFamilies[0] = "MUTATED"
	if !domain.HasRootFamily("OMEGA") {
		t.Fatal("caller must not mutate domain vocabulary")
	}
}

func TestDomainRejectsMalformedOrUnsafeContractSources(t *testing.T) {
	cases := []struct {
		name   string
		ref    string
		source string
	}{
		{"traversal", "../../tmp/identity.json", `{"$id":"urn:example:family","type":"string","enum":["ONE"]}`},
		{"empty-ref", "", `{"$id":"urn:example:family","type":"string","enum":["ONE"]}`},
		{"duplicate-values", "developer/contracts/identity/root_family.json", `{"$id":"urn:example:family","type":"string","enum":["ONE","ONE"]}`},
		{"no-id", "developer/contracts/identity/root_family.json", `{"type":"string","enum":["ONE"]}`},
		{"not-a-string-schema", "developer/contracts/identity/root_family.json", `{"$id":"urn:example:family","type":"object","enum":["ONE"]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t, tc.ref, tc.source)
			if _, err := Load(root); err == nil {
				t.Fatal("invalid declared source must fail closed")
			}
		})
	}
}

func TestDomainRejectsMalformedRoutingContract(t *testing.T) {
	root := fixture(t,
		"developer/contracts/identity/root_family.json",
		`{"$id":"urn:example:family","type":"string","enum":["ONE"]}`,
	)
	file := filepath.Join(root, filepath.FromSlash(routingContract))
	if err := os.WriteFile(file, []byte(`{"identity_sources":`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "routing contract malformed") {
		t.Fatalf("malformed routing contract must fail closed, err=%v", err)
	}
}
