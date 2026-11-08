package applicability

// Native Go snapshot adapter for the read-only applicability assessor.
// YAML decoding is performed by Go. The registered structural/source-integrity
// validator remains in charge of accepting unified policy documents.
import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "time"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
    "gopkg.in/yaml.v3"
)

var sha40Input = regexp.MustCompile(`^[0-9a-f]{40}$`)

var sourceSnapshotPaths = []string{
    ".ptsip/index.json", ".ptsip/contracts", "developer/policy", "src/policy",
    "src/ptsip/common_contract",
}

// gitOutput never evaluates a shell. Every source is read from the exact HEAD.
func gitOutput(root string, args ...string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks"}, args...)...)
    cmd.Dir = root
    data, err := cmd.Output()
    if err != nil {
        if e, ok := err.(*exec.ExitError); ok {
            return nil, fmt.Errorf("GIT_COMMAND_FAILED: %v: %s", err, strings.TrimSpace(string(e.Stderr)))
        }
        return nil, fmt.Errorf("GIT_COMMAND_FAILED: %w", err)
    }
    return data, nil
}

// RequireSourceSnapshot rejects staged, unstaged and untracked canonical inputs.
func RequireSourceSnapshot(root, head string) error {
    args := []string{"status", "--porcelain=v1", "--untracked-files=all", "-z", "--"}
    args = append(args, sourceSnapshotPaths...)
    changed, err := gitOutput(root, args...)
    if err != nil { return err }
    if len(changed) > 0 {
        first := strings.SplitN(string(changed), "\x00", 2)[0]
        return fmt.Errorf("SOURCE_SNAPSHOT_DIRTY: %s", first)
    }
    current, err := gitOutput(root, "rev-parse", "HEAD")
    if err != nil { return err }
    if strings.TrimSpace(string(current)) != head {
        return fmt.Errorf("SOURCE_HEAD_CHANGED_DURING_ASSESSMENT")
    }
    return nil
}

// rejectYAMLConflicts rejects duplicate and non-string mapping keys.
// YAML aliases remain allowed: the old SafeLoader accepted them, and the
// shared registered source-integrity verifier validates the restored meaning.
func rejectYAMLConflicts(node *yaml.Node) error {
    if node.Kind == yaml.AliasNode { return nil }
    if node.Kind == yaml.MappingNode {
        seen := map[string]bool{}
        for i := 0; i < len(node.Content); i += 2 {
            key := node.Content[i]
            if key.Kind != yaml.ScalarNode || (key.Tag != "!!str" && key.Tag != "!!merge") {
                return fmt.Errorf("MAPPING_KEY_MUST_BE_STRING")
            }
            if seen[key.Value] { return fmt.Errorf("DUPLICATE_MAPPING_KEY: %s", key.Value) }
            seen[key.Value] = true
        }
    }
    for _, child := range node.Content {
        if err := rejectYAMLConflicts(child); err != nil { return err }
    }
    return nil
}

func parseSnapshotDocument(raw []byte) (map[string]any, error) {
    if len(raw) > 16*1024*1024 { return nil, fmt.Errorf("POLICY_DOCUMENT_TOO_LARGE") }
    var node yaml.Node
    if err := yaml.Unmarshal(raw, &node); err != nil {
        return nil, fmt.Errorf("INVALID_POLICY_YAML: %w", err)
    }
    if err := rejectYAMLConflicts(&node); err != nil { return nil, err }
    if node.Kind != yaml.DocumentNode || len(node.Content) != 1 ||
        node.Content[0].Kind != yaml.MappingNode {
        return nil, fmt.Errorf("POLICY_DOCUMENT_MUST_BE_OBJECT")
    }
    var object map[string]any
    if err := node.Decode(&object); err != nil { return nil, fmt.Errorf("INVALID_POLICY_YAML: %w", err) }
    if object == nil { return nil, fmt.Errorf("POLICY_DOCUMENT_MUST_BE_OBJECT") }
    // Ensure exactly the JSON-compatible information model expected by Assess.
    if _, err := json.Marshal(object); err != nil { return nil, fmt.Errorf("INVALID_JSON_COMPATIBLE_POLICY: %w", err) }
    return object, nil
}

func asMap(value any) map[string]any {
    item, _ := value.(map[string]any)
    return item
}

func asString(value any) string {
    item, _ := value.(string)
    return item
}

func asList(value any) []any {
    item, _ := value.([]any)
    return item
}

func decodeSelected(value any, result any) error {
    data, err := json.Marshal(value)
    if err != nil { return err }
    return json.Unmarshal(data, result)
}

// CaptureInput preserves the previous input.py protocol with native Go decoding.
// The existing registered unified validator is retained: migration of that
// shared validator is NOT implied by migration of this tool.
func CaptureInput(root string) (Input, error) {
    var out Input
    var err error
    if root == "" {
        root, err = defaultRepoRoot()
        if err != nil { return out, err }
    }
    absolute, err := filepath.Abs(root)
    if err != nil { return out, err }
    actual, err := gitOutput(absolute, "rev-parse", "--show-toplevel")
    if err != nil { return out, err }
    if filepath.Clean(strings.TrimSpace(string(actual))) != filepath.Clean(absolute) {
        return out, fmt.Errorf("EXACT_REPOSITORY_ROOT_REQUIRED")
    }
    current, err := gitOutput(absolute, "rev-parse", "HEAD")
    if err != nil { return out, err }
    head := strings.TrimSpace(string(current))
    if !sha40Input.MatchString(head) { return out, fmt.Errorf("SOURCE_HEAD_INVALID") }
    if err := RequireSourceSnapshot(absolute, head); err != nil { return out, err }

    read := func(rel string) (map[string]any, error) {
        if rel == "" || strings.ContainsAny(rel, "\\\x00:\r\n") ||
            strings.HasPrefix(rel, "/") || filepath.ToSlash(filepath.Clean(rel)) != rel ||
            strings.HasPrefix(rel, "../") {
            return nil, fmt.Errorf("INVALID_SOURCE_PATH: %s", rel)
        }
        raw, err := gitOutput(absolute, "show", "--no-textconv", head+":"+rel)
        if err != nil { return nil, err }
        return parseSnapshotDocument(raw)
    }
    unifiedIndex, err := read(".ptsip/contracts/policy/unified/index.json")
    if err != nil { return out, err }
    registryPath := ".ptsip/contracts/policy/unified/applicability-registry.json"
    if unifiedIndex["applicability_assessment_ref"] != registryPath {
        return out, fmt.Errorf("APPLICABILITY_ASSESSOR_NOT_REGISTERED")
    }
    registry, err := read(registryPath)
    if err != nil { return out, err }
    if registry["schema_version"] != "ptsip-policy-applicability-registration/v1" ||
        registry["status"] != "CANDIDATE_READ_ONLY" ||
        registry["policy_mutation_authorized"] != false ||
        registry["normative_policy_activation"] != false ||
        registry["evaluation_authorized"] != false ||
        registry["execution_authorized"] != false {
        return out, fmt.Errorf("APPLICABILITY_REGISTRY_NOT_CANDIDATE_READ_ONLY")
    }
    contracts := map[string]Contract{}
    for _, kind := range []string{"RA", "RAC", "AEC"} {
        ref := asString(asMap(registry["contract_sources"])[kind])
        if !strings.HasPrefix(ref, ".ptsip/contracts/compile-sealed-rbac/") ||
            strings.Contains(ref, "..") {
            return out, fmt.Errorf("INVALID_CONTRACT_SOURCE: %s", kind)
        }
        contract, err := read(ref)
        if err != nil { return out, err }
        contracts[kind] = Contract{
            SourceRef: ref,
            ActivationState: asString(contract["activation_state"]),
            DefinitionState: asString(contract["definition_state"]),
        }
    }
    familiesDoc, err := read(".ptsip/contracts/policy/Policy_Rules/Family_Selection.json")
    if err != nil { return out, err }
    families := asMap(familiesDoc["family_routes"])
    if len(families) != 14 { return out, fmt.Errorf("ROOT_FAMILY_REGISTRY_MISMATCH") }
    policies := []Policy{}
    seen := map[string]bool{}
    for _, plane := range []struct {
        Kind string
        Index string
        Prefix string
    }{
        {"PTSIP_DEVELOPER_POLICY", "developer/policy/index.yaml", "developer/policy/"},
        {"PTSIP_SUPPORT_FEATURE", "src/policy/index.yaml", "src/policy/"},
    } {
        index, err := read(plane.Index)
        if err != nil { return out, err }
        for _, raw := range asList(index["policies"]) {
            entry := asMap(raw)
            id, rel := asString(entry["id"]), asString(entry["path"])
            if plane.Kind == "PTSIP_SUPPORT_FEATURE" { rel = "src/policy/" + rel }
            parts := strings.Split(rel, "/")
            if len(parts) != 4 { continue } // exclude independent BOUND records
            if _, ok := families[parts[2]]; !ok { continue }
            if strings.Join(parts[:2], "/") != strings.TrimSuffix(plane.Prefix, "/") ||
                !strings.HasPrefix(rel, plane.Prefix) ||
                filepath.Base(rel) != id+".yaml" {
                return out, fmt.Errorf("POLICY_INDEX_PATH_OUTSIDE_SCOPE: %s", rel)
            }
            if id == "" || seen[id] { return out, fmt.Errorf("DUPLICATE_INDEXED_POLICY: %s", id) }
            seen[id] = true
            doc, err := read(rel)
            if err != nil { return out, err }
            if doc["schema_version"] != unifiedIndex["document_schema_version"] {
                return out, fmt.Errorf("NON_UNIFIED_INDEXED_POLICY: %s", rel)
            }
            identity := asMap(doc["Policy_Identity"])
            if identity["id"] != id || identity["policy_kind"] != plane.Kind ||
                asString(identity["root_family"]) != parts[2] {
                return out, fmt.Errorf("POLICY_IDENTITY_MISMATCH: %s", rel)
            }
            if asMap(asMap(doc["Policy_Content"])["source_document"])["path"] != rel {
                return out, fmt.Errorf("POLICY_SOURCE_PATH_MISMATCH: %s", rel)
            }
            // Use the existing registered unified source-integrity verifier.
            if _, err := authoring.CheckUnifiedDocument(absolute, authoring.Document(doc)); err != nil {
                return out, fmt.Errorf("SOURCE_POLICY_CONTRACT_INVALID: %s: %w", rel, err)
            }
            var rules []Rule
            if err := decodeSelected(asMap(doc["Policy_Rules"])["rules"], &rules); err != nil {
                return out, fmt.Errorf("POLICY_RULES_INVALID: %s: %w", rel, err)
            }
            var items []SourceItem
            if err := decodeSelected(asMap(doc["Policy_Content"])["items"], &items); err != nil {
                return out, fmt.Errorf("POLICY_ITEMS_INVALID: %s: %w", rel, err)
            }
            policies = append(policies, Policy{
                ID:id, Path:rel, Plane:plane.Kind, Family:parts[2],
                CatalogStatus:asString(entry["status"]),
                DocumentStatus:asString(asMap(doc["Policy_Lifecycle"])["status"]),
                Rules:rules, Items:items,
            })
        }
    }
    if err := RequireSourceSnapshot(absolute, head); err != nil { return out, err }
    if len(policies) == 0 { return out, fmt.Errorf("NO_INDEXED_POLICIES") }
    out = Input{
        SchemaVersion:InputVersion, SourceHead:head, Contracts:contracts, Policies:policies,
    }
    return out, nil
}

// captureToJSON is used by both snapshot CLI and the local runner.
func captureToJSON(input Input) ([]byte, error) {
    return json.MarshalIndent(input, "", "  ")
}

func defaultRepoRoot() (string, error) {
    current, err := os.Getwd()
    if err != nil { return "", err }
    data, err := gitOutput(current, "rev-parse", "--show-toplevel")
    if err != nil { return "", err }
    return strings.TrimSpace(string(data)), nil
}
