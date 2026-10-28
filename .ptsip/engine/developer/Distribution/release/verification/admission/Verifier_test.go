package ctrl_admission

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWithdrawnCTRLPolicyFailsClosed(t *testing.T) {
	root, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{policyRef, approvalRef} {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(ref)))
		if !os.IsNotExist(err) {
			t.Fatalf("withdrawn artifact still present or inaccessible: %s: %v", ref, err)
		}
	}
	if err := Verify(root); err == nil {
		t.Fatal("withdrawn CTRL policy verified")
	}
	report, code := Run(root)
	if code == 0 {
		t.Fatalf("withdrawn CTRL policy admitted: %v", report)
	}
	m := report.(map[string]any)
	if m["status"] != "BLOCKED" || m["publication_authorized"] != false {
		t.Fatalf("withdrawn CTRL policy did not fail closed: %v", report)
	}
}

func TestOrphanedCTRLPolicySchemaDoesNotGrantAuthority(t *testing.T) {
	root, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	// Read this orphaned schema as a test fixture. The retired authority
	// reader intentionally refuses this current schema path.
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(schemaRef)))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	if err := validateSchema(schema, map[string]any{}, "ctrl"); err == nil {
		t.Fatal("an empty policy passed the former closed schema")
	}
	if err := Verify(root); err == nil {
		t.Fatal("schema existence implicitly reactivated withdrawn CTRL policy")
	}
}
