package schema_validation

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "reflect"
    "sort"
    "strings"
    "testing"

    "github.com/santhosh-tekuri/jsonschema/v6"
    "go.yaml.in/yaml/v3"
)

const (
    govStatusRef = ".ptsip/contracts/schemas/Policy_Status/root-family/GOV_status.schema.json"
    govRulesRef = ".ptsip/contracts/schemas/Policy_Rules/root-family/GOV_rules.schema.json"
    govPublicRef = ".ptsip/contracts/schemas/root-family/GOV_public.schema.json"
    govSharedRulesRef = ".ptsip/contracts/schemas/Policy_Rules/Public_Structure_Rules.schema.json"
    govRoutingRef = ".ptsip/contracts/policy/routing/policy_routes.json"
)

var govPolicies = map[string]struct {
    obligations int
    prohibitions int
    statusVersion string
    ruleVersion string
}{
    "MPD-GOV-0001": {20, 4, "2", "22"},
    "MPD-GOV-0004": {8, 3, "3", "3"},
    "MPD-GOV-0005": {9, 5, "3", "2"},
    "MPD-GOV-0006": {17, 5, "1", "11"},
}

func govRepositoryRoot(t *testing.T) string {
    t.Helper()
    root, err := filepath.Abs(filepath.Join("..", "..", ".."))
    if err != nil { t.Fatal(err) }
    return root
}

func govFile(t *testing.T, relative string) []byte {
    t.Helper()
    data, err := os.ReadFile(filepath.Join(govRepositoryRoot(t), filepath.FromSlash(relative)))
    if err != nil { t.Fatalf("%s: %v", relative, err) }
    return data
}

func govObject(t *testing.T, value any, context string) map[string]any {
    t.Helper()
    obj, ok := value.(map[string]any)
    if !ok { t.Fatalf("%s: expected object, got %T", context, value) }
    return obj
}

func govArray(t *testing.T, value any, context string) []any {
    t.Helper()
    rows, ok := value.([]any)
    if !ok { t.Fatalf("%s: expected array, got %T", context, value) }
    return rows
}

func govJSON(t *testing.T, relative string) map[string]any {
    t.Helper()
    var obj map[string]any
    if err := json.Unmarshal(govFile(t, relative), &obj); err != nil { t.Fatal(err) }
    return obj
}

func govYAML(t *testing.T, relative string) map[string]any {
    t.Helper()
    var obj map[string]any
    if err := yaml.Unmarshal(govFile(t, relative), &obj); err != nil { t.Fatalf("%s: %v", relative, err) }
    return obj
}

func govCompiler(t *testing.T, primary string, dependencies ...string) *jsonschema.Schema {
    t.Helper()
    compiler := jsonschema.NewCompiler()
    compiler.DefaultDraft(jsonschema.Draft2020)
    for _, path := range append([]string{primary}, dependencies...) {
        resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(govFile(t, path)))
        if err != nil { t.Fatalf("%s: %v", path, err) }
        obj := govObject(t, resource, path)
        id, ok := obj["$id"].(string)
        if !ok || id == "" { t.Fatalf("%s: missing $id", path) }
        if err := compiler.AddResource(id, resource); err != nil { t.Fatalf("%s: %v", path, err) }
    }
    id := govJSON(t, primary)["$id"].(string)
    schema, err := compiler.Compile(id)
    if err != nil { t.Fatalf("compile %s: %v", primary, err) }
    return schema
}

func govFullSchema(t *testing.T) *jsonschema.Schema {
    t.Helper()
    return govCompiler(t, govPublicRef,
        ".ptsip/contracts/schemas/Policy_Identity/PTSIP_Policy_Identity.schema.json",
        ".ptsip/contracts/policy/identity/policy_kind.json",
        ".ptsip/contracts/policy/identity/root_family.json",
        ".ptsip/contracts/schemas/Authority/PTSIP_Policy_Authority.schema.json",
        ".ptsip/contracts/schemas/Authority/Authority_Category/Authority_Major_Category.schema.json",
        ".ptsip/contracts/policy/authority/Authority_Major_Category.json",
        // Register transitive authoritative category resources, even when their
        // fields are optional: schema compilation resolves their $ref targets.
        ".ptsip/contracts/policy/authority/Authority_Minor_Category.json",
        ".ptsip/contracts/policy/authority/Authority_Micro_Category.json",
        govStatusRef, govRulesRef, govSharedRulesRef)
}

func govKeys(obj map[string]any) []string {
    keys := make([]string, 0, len(obj))
    for key := range obj { keys = append(keys, key) }
    sort.Strings(keys)
    return keys
}

func govEqualKeys(obj map[string]any, keys ...string) bool {
    sort.Strings(keys)
    return reflect.DeepEqual(govKeys(obj), keys)
}

func govPolicy(t *testing.T, id string) map[string]any {
    t.Helper()
    return govYAML(t, "developer/policy/GOV/"+id+".yaml")
}

func TestGOVPolicyMigrationAndFullSchema(t *testing.T) {
    rules := govCompiler(t, govRulesRef, govSharedRulesRef)
    status := govCompiler(t, govStatusRef)
    full := govFullSchema(t)
    for id, expected := range govPolicies {
        t.Run(id, func(t *testing.T) {
            document := govPolicy(t, id)
            if !govEqualKeys(document, "Policy_Identity", "Policy_Status", "Authority", "Policy_Rules") {
                t.Fatalf("noncanonical top-level fields: %v", govKeys(document))
            }
            identity := govObject(t, document["Policy_Identity"], "Policy_Identity")
            if identity["id"] != id || identity["root_family"] != "GOV" { t.Fatalf("identity mismatch: %v", identity) }

            state := govObject(t, document["Policy_Status"], "Policy_Status")
            if !govEqualKeys(state, "Policy_Status_Schema_ver", "status", "contract") ||
                state["Policy_Status_Schema_ver"] != expected.statusVersion ||
                state["status"] != "APPROVED.Applying" ||
                !strings.HasPrefix(fmt.Sprint(state["contract"]), "urn:ptsip:contract:policy:gov:") {
                t.Fatalf("GOV status mismatch: %v", state)
            }
            if err := status.Validate(state); err != nil { t.Fatalf("status schema: %v", err) }

            rule := govObject(t, document["Policy_Rules"], "Policy_Rules")
            ruleIdentity := govObject(t, rule["Identity"], "Policy_Rules.Identity")
            contractID, _ := ruleIdentity["contract_id"].(string)
            if !govEqualKeys(ruleIdentity, "contract_id", "Rule_ver") ||
                !strings.HasPrefix(contractID, "urn:ptsip:policy:rule:gov:") ||
                strings.Contains(contractID, ":v1") || ruleIdentity["Rule_ver"] != expected.ruleVersion {
                t.Fatalf("GOV rule identity mismatch: %v", ruleIdentity)
            }
            if err := rules.Validate(rule); err != nil { t.Fatalf("rules schema: %v", err) }
            constraints := govObject(t, rule["Constraints"], "Constraints")
            if _, legacy := constraints["prohibitions"]; legacy { t.Fatal("GOV must not revive legacy separate prohibitions") }
            obligations := govArray(t, constraints["obligations"], "Constraints.obligations")
            positive, negative := 0, 0
            seen := make(map[string]bool)
            for _, raw := range obligations {
                item := govObject(t, raw, "Constraints.obligations[]")
                itemID, ok := item["id"].(string)
                if !ok || seen[itemID] { t.Fatalf("missing or duplicate obligation ID: %v", itemID) }
                seen[itemID] = true
                if fmt.Sprint(item["rule"]) == "" { t.Fatalf("empty rule: %s", itemID) }
                switch {
                case strings.HasPrefix(itemID, "OBLIGATION/"):
                    positive++
                    if !govEqualKeys(item, "id", "applies_when", "rule", "Rule_state") ||
                        item["Rule_state"] != "APPROVED.provisional" || fmt.Sprint(item["applies_when"]) == "" {
                        t.Fatalf("invalid obligation: %s: %v", itemID, item)
                    }
                case strings.HasPrefix(itemID, "PROHIBITION/"):
                    negative++
                    if !govEqualKeys(item, "id", "rule") { t.Fatalf("invalid prohibition: %s: %v", itemID, item) }
                default:
                    t.Fatalf("unknown GOV constraint identity: %s", itemID)
                }
            }
            if positive != expected.obligations || negative != expected.prohibitions {
                t.Fatalf("migration lost constraints: obligations=%d prohibitions=%d", positive, negative)
            }
            if err := full.Validate(document); err != nil { t.Fatalf("full GOV document: %v", err) }
        })
    }
}

func TestGOVRoutingIsolatedFromUnmigratedFamilies(t *testing.T) {
    routes := govObject(t, govObject(t, govJSON(t, govRoutingRef)["routes"], "routes")["PTSIP_DEVELOPER_POLICY"], "PTSIP_DEVELOPER_POLICY")
    if len(routes) != 14 { t.Fatalf("expected 14 declared routes; got %d", len(routes)) }
    for family, raw := range routes {
        route := govObject(t, raw, family)
        refs := govObject(t, route["schema_refs"], family+".schema_refs")
        if family == "GOV" {
            if refs["public"] != govPublicRef || refs["rules"] != govRulesRef { t.Fatalf("GOV route not migrated: %v", refs) }
        } else if refs["public"] != ".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json" {
            t.Fatalf("%s: unrelated family public schema changed without migration: %v", family, refs["public"])
        }
    }
    public := govJSON(t, govPublicRef)
    props := govObject(t, public["properties"], "GOV public properties")
    if govObject(t, props["Policy_Status"], "Policy_Status")["$ref"] != govJSON(t, govStatusRef)["$id"] ||
        govObject(t, props["Policy_Rules"], "Policy_Rules")["$ref"] != govJSON(t, govRulesRef)["$id"] {
        t.Fatal("GOV public contract must reference GOV-specific status and rules")
    }
    shared := govJSON(t, ".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json")
    if shared["$id"] != "urn:ptsip:policy:Status:v1" ||
        !reflect.DeepEqual(shared["required"], []any{"Schema", "constant"}) {
        t.Fatal("shared legacy status contract changed outside GOV migration")
    }
}

func TestGOVFlattenedLifecycleStateVocabulary(t *testing.T) {
    schemaDocument := govJSON(t, govStatusRef)
    values := govArray(t, govObject(t, govObject(t, schemaDocument["properties"], "status schema properties")["status"], "status")["enum"], "enum")
    actual := make(map[string]bool)
    for _, value := range values {
        text, ok := value.(string)
        if !ok || actual[text] { t.Fatalf("invalid or duplicate GOV status: %v", value) }
        actual[text] = true
    }
    derived := make(map[string]bool)
    for _, major := range []string{"APPROVED", "DRAFT", "RETIRED"} {
        source := govJSON(t, ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_"+major+".json")
        subs := govArray(t, govObject(t, govObject(t, source["properties"], "substatus properties")["substatus"], "substatus")["enum"], "substatus enum")
        for _, sub := range subs { derived[major+"."+sub.(string)] = true }
    }
    if !reflect.DeepEqual(actual, derived) { t.Fatalf("GOV lifecycle vocabulary does not match approved lifecycle sources") }
    compiled := govCompiler(t, govStatusRef)
    valid := map[string]any{"Policy_Status_Schema_ver":"3", "status":"APPROVED.Applying", "contract":"urn:ptsip:contract:policy:gov:identity:integrity"}
    if err := compiled.Validate(valid); err != nil { t.Fatalf("valid status: %v", err) }
    changes := []map[string]any{
        {"status":"APPROVED.nonexistent"},
        {"constant":map[string]any{"status":"APPROVED", "substatus":"provisional"}},
        {"Policy_Status_Schema_ver":""},
    }
    for i, change := range changes {
        record := make(map[string]any, len(valid)+len(change))
        for k, v := range valid { record[k] = v }
        for k, v := range change { record[k] = v }
        if compiled.Validate(record) == nil { t.Errorf("accepted invalid flat status case %d", i) }
    }
}

func TestGOVRuleSchemaRejectsLegacyOrInvalidConstraints(t *testing.T) {
    compiled := govCompiler(t, govRulesRef, govSharedRulesRef)
    original := govObject(t, govPolicy(t, "MPD-GOV-0006")["Policy_Rules"], "Policy_Rules")
    changes := []struct{name string; mutate func(map[string]any)}{
        {"missing-obligation-state", func(rule map[string]any) {
            constraints := govObject(t, rule["Constraints"], "Constraints")
            first := govObject(t, govArray(t, constraints["obligations"], "obligations")[0], "obligation")
            delete(first, "Rule_state")
        }},
        {"invalid-prohibition-field", func(rule map[string]any) {
            constraints := govObject(t, rule["Constraints"], "Constraints")
            for _, raw := range govArray(t, constraints["obligations"], "obligations") {
                item := govObject(t, raw, "constraint")
                if strings.HasPrefix(fmt.Sprint(item["id"]), "PROHIBITION/") {
                    item["operation"] = item["rule"]
                    return
                }
            }
            t.Fatal("fixture contains no prohibition")
        }},
        {"versioned-rule-id", func(rule map[string]any) {
            identity := govObject(t, rule["Identity"], "Identity")
            identity["contract_id"] = identity["contract_id"].(string)+":v1"
        }},
        {"legacy-prohibitions-array", func(rule map[string]any) {
            constraints := govObject(t, rule["Constraints"], "Constraints")
            constraints["prohibitions"] = []any{}
        }},
    }
    for _, change := range changes {
        t.Run(change.name, func(t *testing.T) {
            encoded, err := json.Marshal(original)
            if err != nil { t.Fatal(err) }
            var copyRule map[string]any
            if err := json.Unmarshal(encoded, &copyRule); err != nil { t.Fatal(err) }
            change.mutate(copyRule)
            if err := compiled.Validate(copyRule); err == nil { t.Fatal("invalid GOV rule was accepted") }
        })
    }
}

func TestGOVAuthorityFieldOwnership(t *testing.T) {
    authority := govJSON(t, ".ptsip/contracts/schemas/Authority/PTSIP_Policy_Authority.schema.json")
    if authority["$id"] != "urn:ptsip:policy:Policy_Authority:v3" { t.Fatal("unexpected authority contract identity") }
    required := govArray(t, authority["required"], "Authority.required")
    seen := false
    for _, value := range required { if value == "authority_id" { seen = true } }
    if !seen { t.Fatal("authority_id must be required") }
    props := govObject(t, authority["properties"], "Authority.properties")
    if _, obsolete := props["authority_Identity"]; obsolete { t.Fatal("obsolete nested authority_Identity") }
    if _, ok := props["authority_id"].(map[string]any); !ok { t.Fatal("missing authority_id schema") }
    parent := govJSON(t, ".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json")
    if govObject(t, govObject(t, parent["properties"], "public properties")["Authority"], "Authority")["$ref"] != authority["$id"] {
        t.Fatal("legacy public schema uses a different Authority contract")
    }
}

func TestGOVObligationRuleStateOwner(t *testing.T) {
    shared := govJSON(t, govSharedRulesRef)
    defs := govObject(t, shared["$defs"], "$defs")
    obligation := govObject(t, defs["ObligationState"], "ObligationState")
    found := false
    for _, value := range govArray(t, obligation["required"], "required") { if value == "Rule_state" { found = true } }
    if !found { t.Fatal("shared obligation state not required") }
    if govObject(t, defs["RuleState"], "RuleState")["minLength"] == nil { t.Fatal("RuleState requires a nonempty value") }

    govRules := govJSON(t, govRulesRef)
    if govRules["$id"] != "urn:ptsip:policy:GOV_rules:v6" { t.Fatal("GOV rule version is incorrect") }
    doc := govObject(t, govObject(t, govRules["$defs"], "$defs")["PolicyDocumentRules"], "PolicyDocumentRules")
    layers := govArray(t, doc["allOf"], "allOf")
    specifics := govObject(t, layers[1], "GOV-specific")
    properties := govObject(t, specifics["properties"], "properties")
    identity := govObject(t, properties["Identity"], "Identity")
    if !govEqualKeys(govObject(t, identity["properties"], "identity.properties"), "contract_id", "Rule_ver") {
        t.Fatal("GOV Rule_state must not be owned by rule Identity")
    }
    constraints := govObject(t, properties["Constraints"], "Constraints")
    constraintProps := govObject(t, constraints["properties"], "Constraints.properties")
    if _, exists := constraintProps["prohibitions"]; exists { t.Fatal("GOV separates prohibitions from obligations") }
    item := govObject(t, govObject(t, constraintProps["obligations"], "obligations")["items"], "items")
    conditional := govArray(t, item["allOf"], "item.allOf")
    required := govArray(t, govObject(t, govObject(t, conditional[0], "if/then")["then"], "then")["required"], "then.required")
    state, applicable := false, false
    for _, v := range required {
        state = state || v == "Rule_state"
        applicable = applicable || v == "applies_when"
    }
    if !state || !applicable { t.Fatal("GOV obligations must carry both Rule_state and applies_when") }
}

func TestGOVPolicyReaderAcceptsWindowsLineEndings(t *testing.T) {
    for _, data := range []string{
        "Policy_Identity:\n  id: MPD-GOV-0006\n",
        "Policy_Identity:\r\n  id: MPD-GOV-0006\r\n",
        "\ufeffPolicy_Identity:\r\n  id: MPD-GOV-0006\r\n",
    } {
        source := strings.TrimPrefix(strings.ReplaceAll(data, "\r\n", "\n"), "\ufeff")
        var document map[string]any
        if err := yaml.Unmarshal([]byte(source), &document); err != nil { t.Fatal(err) }
        if govObject(t, document["Policy_Identity"], "Policy_Identity")["id"] != "MPD-GOV-0006" {
            t.Fatal("GOV policy reader lost canonical identity")
        }
    }
}

// The index is a lookup projection; its state must track each owning document.
func TestGOVCatalogStatusMatchesPolicyDocuments(t *testing.T) {
    catalog := govYAML(t, "developer/policy/index.yaml")
    entries := govArray(t, catalog["policies"], "policies")
    found := make(map[string]bool)
    for _, raw := range entries {
        entry := govObject(t, raw, "index entry")
        id, _ := entry["id"].(string)
        if _, relevant := govPolicies[id]; !relevant { continue }
        if found[id] { t.Fatalf("duplicate GOV catalog entry: %s", id) }
        found[id] = true

        document := govPolicy(t, id)
        policyStatus := govObject(t, document["Policy_Status"], "Policy_Status")
        fullStatus, _ := policyStatus["status"].(string)
        pieces := strings.SplitN(fullStatus, ".", 2)
        if len(pieces) != 2 || entry["status"] != pieces[0] || entry["substatus"] != pieces[1] {
            t.Fatalf("%s: index status/substatus disagrees with owner policy status %q", id, fullStatus)
        }
    }
    for id := range govPolicies {
        if !found[id] { t.Errorf("missing GOV catalog entry: %s", id) }
    }
}
