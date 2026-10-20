package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerReadiness(t *testing.T) {
	cases := []struct {
		name, content, status, reason string
		write                         bool
		code                          int
	}{
		{"missing", "", "BLOCKED", "MANAGER_NOT_FOUND", false, 2},
		{"empty", "", "BLOCKED", "MANAGER_EMPTY", true, 2},
		{"malformed", "{", "BLOCKED", "MANAGER_INVALID_JSON", true, 2},
		{"valid", "{}", "DOCUMENT_PARSEABLE", "", true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(managerPath))
			if tc.write {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			raw, code := Run(root)
			r, ok := raw.(Report)
			if !ok {
				t.Fatalf("unexpected result type %T", raw)
			}
			if code != tc.code || r.Status != tc.status || r.Reason != tc.reason || r.ReleaseVerification != "NOT_ASSERTED" {
				t.Fatalf("code=%d report=%+v", code, r)
			}
		})
	}
}
