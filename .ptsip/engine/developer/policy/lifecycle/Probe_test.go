package lifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(schemaPath))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"type":"object"}`), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestProvisionalCasesAndNoConformanceClaim(t *testing.T) {
	cases := examples()
	want := map[string]bool{}
	for _, c := range cases {
		key, err := json.Marshal(c.document)
		if err != nil {
			t.Fatal(err)
		}
		want[string(key)] = c.valid
	}
	calls := 0
	report, exit := runWith(fixtureRoot(t), func(_ any, item any) (bool, error) {
		raw, err := json.Marshal(item)
		if err != nil {
			return false, err
		}
		calls++
		return want[string(raw)], nil
	})
	if exit != 0 || report.Status != "PROVISIONAL_CASES_PASS" || report.Failed != 0 || calls != len(cases) {
		t.Fatalf("unexpected provisional report: %+v exit=%d calls=%d", report, exit, calls)
	}
	if report.PolicyConformance != "NOT_ASSERTED" || report.Completion != "PROVISIONAL_NOT_FINAL" {
		t.Fatalf("must not claim completed validation: %+v", report)
	}
}

func TestUnavailableEngineFailsClosed(t *testing.T) {
	report, exit := runWith(fixtureRoot(t), func(any, any) (bool, error) {
		return false, errors.New("engine not installed")
	})
	if exit != 2 || report.Status != "UNRESOLVED" || report.Error == "" {
		t.Fatalf("engine error must not claim PASS: %+v exit=%d", report, exit)
	}
}

func TestUnavailableSchemaFailsClosed(t *testing.T) {
	report, exit := runWith(t.TempDir(), func(any, any) (bool, error) {
		t.Fatal("engine must not be called for missing schema")
		return true, nil
	})
	if exit != 2 || report.Status != "UNRESOLVED" {
		t.Fatalf("missing schema: %+v", report)
	}
}
