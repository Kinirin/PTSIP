// Package contractentry resolves the current repository's developer-policy
// contracts for an explicit target without loading unrelated contract domains.
// It is a temporary read-only Go entry, not policy or execution authority.
package contractentry

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
)

type Selection struct {
    Domain              string            `json:"domain"`
    Validator           string            `json:"validator"`
    Target              string            `json:"target"`
    PolicyKind          string            `json:"policy_kind"`
    RootFamily          string            `json:"root_family"`
    ContractRefs        map[string]string `json:"contract_refs"`
    ExecutionAuthorized bool              `json:"execution_authorized"`
    MutationAuthorized  bool              `json:"mutation_authorized"`
    PolicyConformance   string            `json:"policy_conformance"`
}

func localFile(root, relative string) (string, error) {
    if relative == "" || path.IsAbs(relative) || path.Clean(relative) != relative ||
        strings.ContainsAny(relative, "\\:\x00") {
        return "", fmt.Errorf("INVALID_CONTRACT_PATH: %s", relative)
    }
    parts := strings.Split(relative, "/")
    current := root
    for i, part := range parts {
        if part == "" || part == "." || part == ".." {
            return "", fmt.Errorf("INVALID_CONTRACT_PATH: %s", relative)
        }
        current = filepath.Join(current, part)
        info, err := os.Lstat(current)
        if err != nil {
            return "", fmt.Errorf("MISSING_REGISTERED_FILE: %s: %w", relative, err)
        }
        if info.Mode()&os.ModeSymlink != 0 {
            return "", fmt.Errorf("SYMLINK_CONTRACT_PATH: %s", relative)
        }
        if i != len(parts)-1 && !info.IsDir() {
            return "", fmt.Errorf("INVALID_CONTRACT_PARENT: %s", relative)
        }
        if i == len(parts)-1 && !info.Mode().IsRegular() {
            return "", fmt.Errorf("CONTRACT_NOT_REGULAR_FILE: %s", relative)
        }
    }
    return current, nil
}

func jsonObject(root, relative string) (map[string]any, error) {
    name, err := localFile(root, relative)
    if err != nil { return nil, err }
    raw, err := os.ReadFile(name)
    if err != nil { return nil, err }
    if len(raw) > 4*1024*1024 { return nil, fmt.Errorf("CONTRACT_TOO_LARGE: %s", relative) }
    var value map[string]any
    if err := json.Unmarshal(raw, &value); err != nil || value == nil {
        return nil, fmt.Errorf("INVALID_CONTRACT_JSON: %s: %v", relative, err)
    }
    return value, nil
}

func member(value any, key string) (any, error) {
    m, ok := value.(map[string]any)
    if !ok { return nil, fmt.Errorf("INVALID_CONTRACT_OBJECT: %s", key) }
    field, found := m[key]
    if !found { return nil, fmt.Errorf("MISSING_CONTRACT_BINDING: %s", key) }
    return field, nil
}

func field(value any, keys ...string) (string, error) {
    var err error
    for _, key := range keys {
        value, err = member(value, key)
        if err != nil { return "", err }
    }
    result, ok := value.(string)
    if !ok || result == "" { return "", fmt.Errorf("INVALID_CONTRACT_BINDING: %s", strings.Join(keys, ".")) }
    return result, nil
}

func object(value any, key string) (map[string]any, error) {
    v, err := member(value, key)
    if err != nil { return nil, err }
    m, ok := v.(map[string]any)
    if !ok { return nil, fmt.Errorf("INVALID_CONTRACT_OBJECT: %s", key) }
    return m, nil
}

func registered(v any, name string) bool {
    xs, ok := v.([]any)
    if !ok { return false }
    for _, x := range xs { if x == name { return true } }
    return false
}

var identityName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// identityFromYAML reads only the two flat selection scalars. Complete YAML
// syntax/schema/semantics is checked by the existing policy-check validator.
// Unsupported or ambiguous selector representation fails closed.
func identityFromYAML(root, target string) (string, string, error) {
    file, err := localFile(root, target)
    if err != nil { return "", "", err }
    info, err := os.Stat(file)
    if err != nil { return "", "", err }
    if info.Size() > 4*1024*1024 { return "", "", fmt.Errorf("POLICY_DOCUMENT_TOO_LARGE") }
    f, err := os.Open(file)
    if err != nil { return "", "", err }
    defer f.Close()
    scanner := bufio.NewScanner(f)
    scanner.Buffer(make([]byte, 4096), 4*1024*1024+1)
    inside, seen := false, false
    values := make(map[string]string)
    for scanner.Scan() {
        line := strings.TrimSuffix(scanner.Text(), "\r")
        if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") { continue }
        if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
            if strings.TrimSpace(line) == "Policy_Identity:" {
                if seen { return "", "", fmt.Errorf("DUPLICATE_POLICY_IDENTITY") }
                seen, inside = true, true
                continue
            }
            inside = false
            continue
        }
        if !inside { continue }
        if strings.Contains(line[:1], "\t") { return "", "", fmt.Errorf("UNSUPPORTED_POLICY_IDENTITY_INDENT") }
        if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
            return "", "", fmt.Errorf("UNSUPPORTED_POLICY_IDENTITY_SHAPE")
        }
        pair := strings.SplitN(strings.TrimSpace(line), ":", 2)
        if len(pair) != 2 { return "", "", fmt.Errorf("UNSUPPORTED_POLICY_IDENTITY_SHAPE") }
        key := strings.TrimSpace(pair[0])
        if key != "policy_kind" && key != "root_family" { continue }
        if _, exists := values[key]; exists { return "", "", fmt.Errorf("DUPLICATE_POLICY_IDENTITY_FIELD: %s", key) }
        scalar := strings.TrimSpace(pair[1])
        if strings.HasPrefix(scalar, "\"") {
            scalar, err = strconv.Unquote(scalar)
            if err != nil { return "", "", fmt.Errorf("INVALID_POLICY_IDENTITY_VALUE: %s", key) }
        } else if strings.HasPrefix(scalar, "'") && strings.HasSuffix(scalar, "'") && len(scalar)>1 {
            scalar = scalar[1:len(scalar)-1]
        }
        if !identityName.MatchString(scalar) { return "", "", fmt.Errorf("INVALID_POLICY_IDENTITY_VALUE: %s", key) }
        values[key] = scalar
    }
    if err := scanner.Err(); err != nil { return "", "", err }
    if !seen || values["policy_kind"] == "" || values["root_family"] == "" {
        return "", "", fmt.Errorf("POLICY_IDENTITY_SELECTION_UNRESOLVED")
    }
    return values["policy_kind"], values["root_family"], nil
}

func Resolve(root, target string) (Selection, error) {
    fail := func(err error) (Selection, error) { return Selection{}, err }
    if !strings.HasSuffix(target, ".yaml") && !strings.HasSuffix(target, ".yml") {
        return fail(fmt.Errorf("UNSUPPORTED_POLICY_TARGET: %s", target))
    }
    if _, err := localFile(root, target); err != nil { return fail(err) }
    index, err := jsonObject(root, ".ptsip/index.json")
    if err != nil { return fail(err) }
    contracts, err := object(index, "namespaces")
    if err != nil { return fail(err) }
    namespace, err := object(contracts, "contracts")
    if err != nil { return fail(err) }
    if namespace["status"] != "ACTIVE" { return fail(fmt.Errorf("CONTRACT_NAMESPACE_NOT_ACTIVE")) }
    ref, err := field(namespace, "index")
    if err != nil { return fail(err) }
    catalogPath := ".ptsip/" + ref
    if catalogPath != ".ptsip/contracts/index.json" { return fail(fmt.Errorf("UNEXPECTED_CONTRACT_INDEX")) }
    catalog, err := jsonObject(root, catalogPath)
    if err != nil { return fail(err) }
    authoring, err := object(catalog, "authoring")
    if err != nil { return fail(err) }
    if authoring["execution_authorized"] != false { return fail(fmt.Errorf("AUTHORING_AUTHORITY_BOUNDARY_CHANGED")) }
    enginePath, err := field(authoring, "entrypoint")
    if err != nil { return fail(err) }
    if enginePath != ".ptsip/engine/developer" { return fail(fmt.Errorf("UNREGISTERED_DEVELOPER_ENTRYPOINT")) }
    engine, err := jsonObject(root, enginePath+"/index.json")
    if err != nil { return fail(err) }
    if engine["policy_authority"] != "NONE" || engine["execution_authorized"] != false {
        return fail(fmt.Errorf("DEVELOPER_ENGINE_AUTHORITY_BOUNDARY_CHANGED"))
    }
    validators, err := object(engine, "validators")
    if err != nil { return fail(err) }
    if validators["policy-check"] != "policy/index.json" || validators["contract-entry"] != "policy/index.json" {
        return fail(fmt.Errorf("UNREGISTERED_POLICY_ENTRY_VALIDATOR"))
    }
    if !registered(authoring["validators"], "policy-rules") { return fail(fmt.Errorf("UNREGISTERED_POLICY_AUTHORING")) }
    policyRef, err := field(catalog, "policy")
    if err != nil { return fail(err) }
    boundaryRef := ".ptsip/contracts/" + policyRef
    boundary, err := jsonObject(root, boundaryRef)
    if err != nil { return fail(err) }
    if boundary["domain"] != "policy" { return fail(fmt.Errorf("UNEXPECTED_CONTRACT_DOMAIN")) }
    routeRef, err := field(boundary, "routing_binding", "data")
    if err != nil { return fail(err) }
    routes, err := jsonObject(root, routeRef)
    if err != nil { return fail(err) }
    kind, family, err := identityFromYAML(root, target)
    if err != nil { return fail(err) }
    vocabularyKind, err := field(routes, "identity_sources", "policy_kind")
    if err != nil { return fail(err) }
    vocabularyFamily, err := field(boundary, "rules_binding", "family_source")
    if err != nil { return fail(err) }
    kindList, err := jsonObject(root, vocabularyKind)
    if err != nil { return fail(err) }
    familyList, err := jsonObject(root, vocabularyFamily)
    if err != nil { return fail(err) }
    if !registered(kindList["enum"], kind) || !registered(familyList["enum"], family) {
        return fail(fmt.Errorf("UNREGISTERED_POLICY_IDENTITY"))
    }
    routeMap, err := object(routes, "routes")
    if err != nil { return fail(err) }
    route, err := object(routeMap, kind)
    if err != nil { return fail(fmt.Errorf("POLICY_KIND_ROUTE_UNRESOLVED: %s", kind)) }
    outputRoot, err := field(route, "output_root")
    if err != nil { return fail(err) }
    if !strings.HasPrefix(target, outputRoot+"/") { return fail(fmt.Errorf("POLICY_TARGET_OUTSIDE_REGISTERED_ROUTE")) }
    rules, err := object(boundary, "rules_binding")
    if err != nil { return fail(err) }
    folder, err := field(rules, "directory")
    if err != nil { return fail(err) }
    fileTemplate, err := field(rules, "filename")
    if err != nil { return fail(err) }
    if fileTemplate != "{root_family}_rules.json" { return fail(fmt.Errorf("UNSUPPORTED_FAMILY_CONTRACT_TEMPLATE")) }
    familyRef := folder + "/" + strings.ReplaceAll(fileTemplate, "{root_family}", family)
    familyDoc, err := jsonObject(root, familyRef)
    if err != nil { return fail(err) }
    if familyDoc["root_family"] != family { return fail(fmt.Errorf("FAMILY_CONTRACT_IDENTITY_MISMATCH")) }
    publicRef, err := field(routes, "schema_refs", "public")
    if err != nil { return fail(err) }
    selectionRef, err := field(rules, "selection_schema")
    if err != nil { return fail(err) }
    profileRef, err := field(rules, "responsibility_profile")
    if err != nil { return fail(err) }
    for _, ref := range []string{publicRef, selectionRef, profileRef} {
        if _, err := localFile(root, ref); err != nil { return fail(err) }
    }
    return Selection{Domain: "developer_policy", Validator: "policy-check", Target: target,
        PolicyKind: kind, RootFamily: family,
        ContractRefs: map[string]string{"boundary": boundaryRef, "routing": routeRef,
            "public_schema": publicRef, "selection_schema": selectionRef,
            "family": familyRef, "responsibility_profile": profileRef},
        PolicyConformance: "NOT_ASSERTED"}, nil
}
