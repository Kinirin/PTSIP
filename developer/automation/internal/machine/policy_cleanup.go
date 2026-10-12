package machine

import (
    "bytes"
    "crypto/sha1"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
)

const cleanupOwner = "developer/policy/CHANGE/MPD-CHANGE-0001.yaml"
const cleanupBaseline = "developer/policy/registries/root-family-approved-baseline.json"
const cleanupGraph = "developer/policy/registries/root-family-migration.json"

var cleanupPolicyID = regexp.MustCompile("^(MPD|SFP)-[A-Z0-9-]+$")

func cleanupRows(value any) []Object {
    result := []Object{}
    for _, raw := range List(value) {
        if row := Map(raw); row != nil { result = append(result, row) }
    }
    return result
}

func (r *Repository) cleanupGit(args ...string) ([]byte, error) {
    command := exec.Command("git", append([]string{"-C", r.Root}, args...)...)
    result, err := command.Output()
    if err != nil { return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err) }
    // Git object content must retain trailing bytes for exact SHA-256 proof.
    if len(args) > 0 && args[0] == "show" { return result, nil }
    return bytes.TrimSpace(result), nil
}

func cleanupGitBlob(content []byte) string {
    prefix := []byte(fmt.Sprintf("blob %d\x00", len(content)))
    sum := sha1.Sum(append(prefix, content...))
    return fmt.Sprintf("%x", sum)
}

func (r *Repository) cleanupContract() (Object, error) {
    owner, err := r.Read(cleanupOwner)
    if err != nil { return nil, err }
    state := Map(owner["policy"])
    if state["status"] != "APPROVED" || state["substatus"] != "Applying" {
        return nil, fmt.Errorf("cleanup owner is not APPROVED.Applying")
    }
    rule := Map(Map(owner["rules"])["removable_policy_cleanup"])
    if rule == nil || rule["conflict_resolution"] != "KEEP_LATEST" {
        return nil, fmt.Errorf("unapproved cleanup contract")
    }
    scope := Map(rule["scope"])
    execution := Map(rule["execution"])
    if scope["policy_classes"] != "ALL_REGISTERED_POLICY_CLASSES" ||
        scope["path_prefix_restriction"] != "NONE" ||
        execution["trigger"] != "EXPLICIT_USER_CLEANUP_REQUEST_ONLY" ||
        execution["automatic_execution_on_removable_transition"] != "FORBIDDEN" {
        return nil, fmt.Errorf("cleanup scope or trigger does not match approved contract")
    }
    return rule, nil
}

func (r *Repository) cleanupFind(id string) (Object, string, Object, error) {
    if !cleanupPolicyID.MatchString(id) { return nil, "", nil, fmt.Errorf("invalid policy identity") }
    paths := []string{"developer/policy/index.yaml", "src/policy/index.yaml"}
    var found Object
    var manifest string
    for _, indexPath := range paths {
        index, err := r.Read(indexPath)
        if err != nil { return nil, "", nil, err }
        for _, row := range cleanupRows(index["policies"]) {
            if Text(row["id"]) != id { continue }
            if found != nil { return nil, "", nil, fmt.Errorf("ambiguous registered policy") }
            found, manifest = row, indexPath
        }
    }
    if found != nil {
        root := strings.TrimSuffix(manifest, "index.yaml")
        path := Text(found["path"])
        if strings.HasPrefix(path, root) { // Developer catalog contains repository-relative paths.
            return found, path, nil, nil
        }
        if strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
            return nil, "", nil, fmt.Errorf("invalid catalog path")
        }
        return found, root+path, nil, nil
    }
    // A frozen migration source may be absent from the current authority catalog.
    if !strings.HasPrefix(id, "MPD-") { return nil, "", nil, fmt.Errorf("unregistered policy") }
    graph, err := r.Read(cleanupGraph)
    if err != nil { return nil, "", nil, err }
    for _, entry := range cleanupRows(graph["sources"]) {
        if entry["source_policy_id"] != id { continue }
        path := "developer/policy/" + Text(entry["archive_path"])
        if path != "developer/policy/legacy/"+id+".yaml" {
            return nil, "", nil, fmt.Errorf("unexpected migration archive path")
        }
        return Object{"id": id, "policy_class": "PTSIP_DEVELOPER_POLICY", "authority_role": "MIGRATION_SOURCE"}, path, entry, nil
    }
    return nil, "", nil, fmt.Errorf("policy not present in current or historical catalog")
}

func (r *Repository) cleanupEvidence(id, path string, frozen Object) error {
    baseline, err := r.Read(cleanupBaseline)
    if err != nil { return err }
    graph, err := r.Read(cleanupGraph)
    if err != nil { return err }
    if frozen == nil {
        for _, entry := range cleanupRows(graph["sources"]) {
            if entry["source_policy_id"] == id { frozen = entry; break }
        }
    }
    if frozen == nil {
        return fmt.Errorf("approved non-legacy Git object pin is not registered")
    }
    revision := Text(baseline["approved_migration_base_head"])
    if len(revision) != 40 || revision != Text(graph["base_head"]) {
        return fmt.Errorf("untrusted frozen revision")
    }
    checked := false
    for _, row := range cleanupRows(baseline["source_records"]) {
        if row["id"] != id { continue }
        if row["archive_path"] != frozen["archive_path"] ||
            row["source_sha256"] != frozen["source_sha256"] ||
            row["record_sha256"] != frozen["record_sha256"] {
            return fmt.Errorf("frozen source does not match approved baseline")
        }
        checked = true
    }
    if !checked { return fmt.Errorf("source missing from approved migration baseline") }
    graphPath, err := r.Path(cleanupGraph)
    if err != nil { return err }
    graphBytes, err := os.ReadFile(graphPath)
    if err != nil { return err }
    if cleanupGitBlob(graphBytes) != Text(baseline["approved_migration_graph_git_blob"]) {
        return fmt.Errorf("approved migration graph Git blob mismatch")
    }
    origin := Text(frozen["original_path"])
    if strings.HasPrefix(origin, "/") || strings.Contains(origin, "..") || origin == "" {
        return fmt.Errorf("invalid pinned original path")
    }
    source, err := r.cleanupGit("show", revision+":developer/policy/"+origin)
    if err != nil { return fmt.Errorf("approved Git source unavailable: %w", err) }
    if LFDigest(source) != Text(frozen["source_sha256"]) {
        return fmt.Errorf("approved historical Git source digest mismatch")
    }
    // Both the immutable source and the current deletion target must have exact identities.
    target, err := r.Path(path)
    if err != nil { return err }
    if _, err := os.Lstat(target); err != nil { return err }
    return nil
}

// cleanupReferences intentionally over-approximates dependencies: ambiguous source references
// are blocked rather than guessed to be historical. Approved immutable audit registries are exempt.
func (r *Repository) cleanupReferences(id, target string) ([]string, error) {
    matches := []string{}
    for _, root := range []string{"developer", "src", ".github"} {
        absolute, err := r.Path(root)
        if err != nil { return nil, err }
        err = filepath.WalkDir(absolute, func(path string, entry os.DirEntry, walkErr error) error {
            if walkErr != nil { return walkErr }
            rel, err := filepath.Rel(r.Root, path)
            if err != nil { return err }
            normalized := filepath.ToSlash(rel)
            if entry.IsDir() {
                if entry.Name() == ".git" || entry.Name() == "legacy" || entry.Name() == "__pycache__" ||
                    entry.Name() == ".venv" || entry.Name() == "node_modules" { return filepath.SkipDir }
                return nil
            }
            if normalized == target ||
                strings.Contains(normalized, "/approvals/") ||
                strings.Contains(normalized, "/analysis/") ||
                strings.Contains(normalized, "/evidence/") ||
                normalized == cleanupGraph || normalized == cleanupBaseline { return nil }
            switch filepath.Ext(path) {
            case ".yaml", ".yml", ".json", ".go", ".py", ".toml":
            default:
                return nil
            }
            info, err := entry.Info()
            if err != nil { return err }
            if info.Size() > 2*1024*1024 { return fmt.Errorf("reference scan cannot inspect large file %s", normalized) }
            data, err := os.ReadFile(path)
            if err != nil { return err }
            if bytes.Contains(data, []byte(id)) { matches = append(matches, normalized) }
            return nil
        })
        if err != nil { return nil, err }
    }
    sort.Strings(matches)
    return matches, nil
}

func (r *Repository) CleanupInspect(id string) (Object, error) {
    report := Object{"schema_version": "ptsip-policy-cleanup-inspection/v1", "policy_id": id, "status": "BLOCKED", "blockers": []string{}, "references": []string{}}
    block := func(message string) { report["blockers"] = append(report["blockers"].([]string), message) }
    rule, err := r.cleanupContract()
    if err != nil { block("INVALID_CLEANUP_CONTRACT: "+err.Error()); return report, nil }
    _ = rule
    record, target, _, err := r.cleanupFind(id)
    if err != nil { block("TARGET_RESOLUTION: "+err.Error()); return report, nil }
    report["policy_class"], report["path"] = record["policy_class"], target
    payload, err := r.Read(target)
    if err != nil { block("POLICY_FILE_UNAVAILABLE: "+err.Error()); return report, nil }
    identity := Map(payload["policy"])
    if identity["id"] != id || payload["policy_class"] != record["policy_class"] {
        block("IDENTITY_OR_CLASS_MISMATCH")
        return report, nil
    }
    if identity["status"] != "RETIRED" || identity["substatus"] != "Removable" {
        block("STATUS_NOT_RETIRED_REMOVABLE")
        return report, nil
    }
    if record["authority_role"] != "MIGRATION_SOURCE" {
        if record["status"] != identity["status"] || record["substatus"] != identity["substatus"] {
            block("CATALOG_STATUS_MISMATCH")
        }
    }
    if record["policy_class"] != "PTSIP_DEVELOPER_POLICY" {
        block("POLICY_CLASS_REMOVABLE_LIFECYCLE_NOT_REGISTERED")
    } else {
        if err := r.Validate("developer/contracts/lifecycle/policy_status.json",
            Object{"status": identity["status"], "substatus": identity["substatus"]}); err != nil {
            block("INVALID_GOV_LIFECYCLE: "+err.Error())
        }
    }
    if err := r.cleanupEvidence(id, target, nil); err != nil {
        block("GIT_EVIDENCE: "+err.Error())
    }
    refs, err := r.cleanupReferences(id, target)
    if err != nil { block("REFERENCE_SCAN: "+err.Error()) } else if len(refs) != 0 {
        report["references"] = refs
        block(fmt.Sprintf("UNRESOLVED_REFERENCES: %d files", len(refs)))
    }
    if len(report["blockers"].([]string)) == 0 { report["status"] = "ELIGIBLE" }
    return report, nil
}

func (r *Repository) CleanupApply(id, expectedHead string) (Object, error) {
    if len(expectedHead) != 40 {
        return Object{"status": "BLOCKED", "policy_id": id, "blockers": []string{"EXACT_EXPECTED_HEAD_REQUIRED"}}, nil
    }
    actual, err := r.cleanupGit("rev-parse", "HEAD")
    if err != nil { return nil, err }
    if string(actual) != expectedHead {
        return Object{"status": "BLOCKED", "policy_id": id, "blockers": []string{"STALE_HEAD"}}, nil
    }
    dirty, err := r.cleanupGit("status", "--porcelain", "--untracked-files=all")
    if err != nil { return nil, err }
    if len(dirty) != 0 {
        return Object{"status": "BLOCKED", "policy_id": id, "blockers": []string{"WORKTREE_NOT_CLEAN"}}, nil
    }
    report, err := r.CleanupInspect(id)
    if err != nil || report["status"] != "ELIGIBLE" { return report, err }
    path := Text(report["path"])
    target, err := r.Path(path)
    if err != nil { return nil, err }
    previous, err := os.ReadFile(target)
    if err != nil { return nil, err }
    info, err := os.Stat(target)
    if err != nil { return nil, err }
    if err := os.Remove(target); err != nil { return nil, err }
    restore := func() { _ = os.WriteFile(target, previous, info.Mode()) }
    // A migration-source catalog identity is retained for immutable audit coverage.
    // Canonical removal needs a separate registered Git pin and catalog projection;
    // these are rejected by CleanupInspect until independently admitted.
    if Text(report["policy_class"]) != "PTSIP_DEVELOPER_POLICY" || !strings.Contains(path, "/legacy/") {
        restore()
        report["status"] = "BLOCKED"
        report["blockers"] = []string{"UNSUPPORTED_CATALOG_RECONCILIATION"}
        return report, nil
    }
    corpus, err := r.LoadConsistentPolicyCorpus()
    if err == nil { err = r.policyCheckDiscovery(corpus, "") }
    if err != nil {
        restore()
        report["status"] = "BLOCKED"
        report["blockers"] = []string{"POST_DELETE_CATALOG_VALIDATION: "+err.Error()}
        return report, nil
    }
    report["status"] = "APPLIED"
    report["removed_path"] = path
    report["source_head"] = expectedHead
    return report, nil
}

// JSON output is closed and machine readable; never synthesize approval from success.
func cleanupJSON(value any) ([]byte, error) { return json.Marshal(value) }
