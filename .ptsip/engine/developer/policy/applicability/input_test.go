package applicability

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

func gitFixture(t *testing.T, root string, args ...string) string {
    t.Helper()
    cmd := exec.Command("git", args...)
    cmd.Dir = root
    raw, err := cmd.CombinedOutput()
    if err != nil { t.Fatalf("git %v: %v: %s", args, err, raw) }
    return strings.TrimSpace(string(raw))
}

func TestRequireSourceSnapshot(t *testing.T) {
    root := t.TempDir()
    gitFixture(t, root, "init", "-q")
    policy := filepath.Join(root, "src", "policy", "index.yaml")
    if err := os.MkdirAll(filepath.Dir(policy), 0755); err != nil { t.Fatal(err) }
    if err := os.WriteFile(policy, []byte("policies: []\n"), 0644); err != nil { t.Fatal(err) }
    gitFixture(t, root, "add", ".")
    gitFixture(t, root, "-c", "user.name=Fixture", "-c",
        "user.email=fixture@example.invalid", "commit", "-q", "-m", "snapshot fixture")
    head := gitFixture(t, root, "rev-parse", "HEAD")
    if err := os.MkdirAll(filepath.Join(root, ".agent"), 0755); err != nil { t.Fatal(err) }
    if err := os.WriteFile(filepath.Join(root, ".agent", "candidate.jsonl"),
        []byte("{}\n"), 0600); err != nil { t.Fatal(err) }
    if err := RequireSourceSnapshot(root, head); err != nil { t.Fatal(err) }
    if err := RequireSourceSnapshot(root, strings.Repeat("0", 40)); err == nil ||
        !strings.Contains(err.Error(), "SOURCE_HEAD_CHANGED_DURING_ASSESSMENT") {
        t.Fatalf("expected HEAD mismatch: %v", err)
    }
    if err := os.WriteFile(policy, []byte("policies: [changed]\n"), 0644); err != nil {
        t.Fatal(err)
    }
    if err := RequireSourceSnapshot(root, head); err == nil ||
        !strings.Contains(err.Error(), "SOURCE_SNAPSHOT_DIRTY") {
        t.Fatalf("expected unstaged change rejection: %v", err)
    }
    gitFixture(t, root, "add", "src/policy/index.yaml")
    if err := RequireSourceSnapshot(root, head); err == nil ||
        !strings.Contains(err.Error(), "SOURCE_SNAPSHOT_DIRTY") {
        t.Fatalf("expected staged change rejection: %v", err)
    }
}

func TestStrictGoYAMLInput(t *testing.T) {
    test := []struct {
        name string
        document string
        wantErr string
    }{
        {"valid", "policy: {id: A, status: ACTIVE}\n", ""},
        {"duplicate", "policy: {id: A, id: B}\n", "DUPLICATE_MAPPING_KEY"},
        {"non-string-key", "policy: {1: abc}\n", "MAPPING_KEY_MUST_BE_STRING"},
        {"alias", "a: &x [1]\nb: *x\n", ""},
        {"scalar", "hello\n", "POLICY_DOCUMENT_MUST_BE_OBJECT"},
    }
    for _, item := range test {
        t.Run(item.name, func(t *testing.T) {
            _, err := parseSnapshotDocument([]byte(item.document))
            if item.wantErr == "" && err != nil { t.Fatal(err) }
            if item.wantErr != "" && (err == nil || !strings.Contains(err.Error(), item.wantErr)) {
                t.Fatalf("want %q: %v", item.wantErr, err)
            }
        })
    }
}
