package lifecycle

import (
    "bytes"
    "fmt"
    "os/exec"
    "path/filepath"
    "strings"
)

// ResolvePinnedMigrationSource proves exact original source bytes against the
// immutable approved SHA-256 and a reachable Git object. A fixed base revision
// is checked first; archives are used only as historical Git object locations,
// never as mutable fallback authority.
func ResolvePinnedMigrationSource(root string, graph, baseline, source Object) ([]byte, string, error) {
    id := Text(source["source_policy_id"])
    wanted := Text(source["source_sha256"])
    base := Text(graph["base_head"])
    if len(base) != 40 || len(wanted) != 64 || id == "" {
        return nil, "", fmt.Errorf("invalid source pin for %s", id)
    }
    if baseline != nil && base != Text(baseline["approved_migration_base_head"]) {
        return nil, "", fmt.Errorf("approved Git revision mismatch")
    }
    relativeRoot := "developer/policy"
    if graph["policy_class"] == "PTSIP_SUPPORT_FEATURE" {
        relativeRoot = "src/policy"
    }
    origin := filepath.ToSlash(filepath.Join(relativeRoot, Text(source["original_path"])))
    archive := filepath.ToSlash(filepath.Join(relativeRoot, Text(source["archive_path"])))
    for _, name := range []string{origin, archive} {
        if !strings.HasPrefix(name, relativeRoot+"/") || strings.Contains(name, "..") {
            return nil, "", fmt.Errorf("untrusted pinned source path")
        }
    }
    git := func(args ...string) ([]byte, error) {
        command := exec.Command("git", append([]string{"-C", root}, args...)...)
        return command.Output()
    }
    valid := func(data []byte) bool {
        normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
        return SHA256(normalized) == wanted
    }
    if data, err := git("show", base+":"+origin); err == nil && valid(data) {
        return data, base, nil
    }
    // Content is pinned by the approved source digest. Other Git revisions
    // can prove it by exact hash; a name or semantic similarity cannot.
    revisions, err := git("log", "--all", "--format=%H", "--", origin, archive)
    if err != nil { return nil, "", fmt.Errorf("Git evidence history unavailable: %w", err) }
    commits := strings.Fields(string(revisions))
    if len(commits) > 400 {
        return nil, "", fmt.Errorf("Git evidence search exceeds bounded history")
    }
    for _, revision := range commits {
        if len(revision) != 40 { return nil, "", fmt.Errorf("invalid Git evidence revision") }
        for _, path := range []string{origin, archive} {
            data, err := git("show", revision+":"+path)
            if err == nil && valid(data) {
                return data, revision, nil
            }
        }
    }
    return nil, "", fmt.Errorf("approved Git object matching frozen SHA-256 not found: %s", id)
}
