package enginewheel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalVerificationDoesNotGrantPublication(t *testing.T) {
	source, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	file := filepath.Join(root, VerificationContract)
	raw, err := os.ReadFile(filepath.Join(source, VerificationContract))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err := Admit(root); err != nil {
		t.Fatalf("local verification depended on retired authority: %v", err)
	}
	if err := AdmitPublication(root); err == nil || !strings.Contains(err.Error(), "SUCCESSOR_AUTHORITY_UNRESOLVED") {
		t.Fatalf("publication gate: %v", err)
	}
	forged := strings.Replace(string(raw), `"runtime_activation_authorized": false`, `"runtime_activation_authorized": true`, 1)
	if forged == string(raw) {
		t.Fatal("fixture mutation did not apply")
	}
	if err := os.WriteFile(file, []byte(forged), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Admit(root); err == nil {
		t.Fatal("runtime activation flag accepted")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := Admit(root); err == nil {
		t.Fatal("missing boundary accepted")
	}
}
