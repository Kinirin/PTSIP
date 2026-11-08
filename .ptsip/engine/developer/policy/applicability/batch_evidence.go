package applicability

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/review"
)

const (
	ContractItemNone            = "NONE"
	ContractItemWholeDefinition = "WHOLE_DEFINITION"
	ContractItemExactInactive   = "EXACT_INACTIVE_ITEM"
	ContractItemExactActive     = "EXACT_ACTIVE_ITEM"
	ContractItemReverseBasis    = "REVERSE_BASIS"
)

// BatchEvidence contains exact evidence for review of one binding. It does not
// classify consumption or grant evaluation, execution, or mutation authority.
type BatchEvidence struct {
	RuleSource        SourceItem     `json:"rule_source"`
	RuleSourceValue   any            `json:"rule_source_value"`
	Basis             []SourceItem   `json:"basis"`
	BasisValues       map[string]any `json:"basis_values"`
	BindingState      string         `json:"binding_state"`
	ContractReference *Reference     `json:"contract_reference,omitempty"`
	ContractItemClass string         `json:"contract_item_class"`
	ContractItemValue any            `json:"contract_item_value,omitempty"`
	ReverseBasis      []string       `json:"reverse_basis"`
	ActivationState   string         `json:"activation_state"`
}

type batchSourceItem struct {
	SourceItem
	Value any `json:"value"`
}

type batchPolicySource struct {
	Rules map[string]Rule
	Items map[string]batchSourceItem
}

// BatchEvidenceResolver reads only exact Git commits. A resolver belongs to
// one captured snapshot and is used serially to reuse immutable document reads.
type BatchEvidenceResolver struct {
	root      string
	snapshot  Input
	policies  map[string]*Policy
	sources   map[string]batchPolicySource
	documents map[string]map[string]any
	commits   map[string]bool
}

func NewBatchEvidenceResolver(root string, snapshot Input) *BatchEvidenceResolver {
	resolver := &BatchEvidenceResolver{
		root: root, snapshot: snapshot,
		policies: map[string]*Policy{}, sources: map[string]batchPolicySource{},
		documents: map[string]map[string]any{}, commits: map[string]bool{},
	}
	for i := range snapshot.Policies {
		policy := &snapshot.Policies[i]
		if _, exists := resolver.policies[policy.ID]; exists {
			resolver.policies[policy.ID] = nil
		} else {
			resolver.policies[policy.ID] = policy
		}
	}
	return resolver
}

func (r *BatchEvidenceResolver) document(revision, rel string) (map[string]any, error) {
	if !sha40.MatchString(revision) || !safePath(rel) || strings.ContainsAny(rel, ":\x00\r\n") {
		return nil, fmt.Errorf("BATCH_INVALID_GIT_SOURCE: %s:%s", revision, rel)
	}
	key := revision + ":" + rel
	if doc, found := r.documents[key]; found {
		return doc, nil
	}
	if !r.commits[revision] {
		kind, err := gitOutput(r.root, "cat-file", "-t", revision)
		if err != nil || strings.TrimSpace(string(kind)) != "commit" {
			return nil, fmt.Errorf("BATCH_SOURCE_COMMIT_UNRESOLVED: %s", revision)
		}
		r.commits[revision] = true
	}
	raw, err := gitOutput(r.root, "show", "--no-textconv", key)
	if err != nil {
		return nil, err
	}
	doc, err := parseSnapshotDocument(raw)
	if err != nil {
		return nil, fmt.Errorf("BATCH_INVALID_SOURCE_DOCUMENT: %s: %w", key, err)
	}
	r.documents[key] = doc
	return doc, nil
}

func (r *BatchEvidenceResolver) policySource(policy *Policy) (batchPolicySource, error) {
	if source, ok := r.sources[policy.ID]; ok {
		return source, nil
	}
	source := batchPolicySource{Rules: map[string]Rule{}, Items: map[string]batchSourceItem{}}
	doc, err := r.document(r.snapshot.SourceHead, policy.Path)
	if err != nil {
		return source, err
	}
	identity := asMap(doc["Policy_Identity"])
	content := asMap(doc["Policy_Content"])
	if asString(identity["id"]) != policy.ID || asString(identity["policy_kind"]) != policy.Plane ||
		asString(identity["root_family"]) != policy.Family ||
		asString(asMap(content["source_document"])["path"]) != policy.Path {
		return source, fmt.Errorf("BATCH_POLICY_SOURCE_MISMATCH: %s", policy.ID)
	}
	var rules []Rule
	if err := decodeSelected(asMap(doc["Policy_Rules"])["rules"], &rules); err != nil {
		return source, err
	}
	for _, rule := range rules {
		if rule.ID == "" || source.Rules[rule.ID].ID != "" {
			return source, fmt.Errorf("BATCH_RULE_ID_CONFLICT: %s/%s", policy.ID, rule.ID)
		}
		source.Rules[rule.ID] = rule
	}
	for _, raw := range asList(content["items"]) {
		entry := asMap(raw)
		id := asString(entry["id"])
		address, hasPointer := entry["source_pointer"].(string)
		value, hasValue := entry["value"]
		if id == "" || !hasPointer || !pointer(address) || !hasValue || source.Items[id].ID != "" {
			return source, fmt.Errorf("BATCH_SOURCE_ITEM_CONFLICT: %s/%s", policy.ID, id)
		}
		source.Items[id] = batchSourceItem{SourceItem: SourceItem{ID: id, SourcePointer: address}, Value: value}
	}
	r.sources[policy.ID] = source
	return source, nil
}

// Resolve verifies the assessment's source coordinates and selected binding
// against the canonical policy at the snapshot commit before reading any ref.
func (r *BatchEvidenceResolver) Resolve(row Decision) (BatchEvidence, error) {
	out := BatchEvidence{
		Basis: []SourceItem{}, BasisValues: map[string]any{},
		ContractItemClass: ContractItemNone, ReverseBasis: []string{},
	}
	policy := r.policies[row.PolicyID]
	if policy == nil || policy.Path != row.PolicyPath || row.EvaluationAuthorized || row.ExecutionAuthorized {
		return out, fmt.Errorf("BATCH_POLICY_DECISION_MISMATCH: %s", row.PolicyID)
	}
	source, err := r.policySource(policy)
	if err != nil {
		return out, err
	}
	rule := source.Rules[row.RuleID]
	binding, found := rule.ContractBindings[row.ContractKind]
	ruleItem := source.Items[rule.SourceRef]
	if !found || ruleItem.ID == "" || ruleItem.SourcePointer != row.SourcePointer {
		return out, fmt.Errorf("BATCH_RULE_SOURCE_MISMATCH: %s/%s", row.PolicyID, row.RuleID)
	}
	basisRefs := slices.Clone(binding.BasisRefs)
	slices.Sort(basisRefs)
	if !slices.Equal(basisRefs, row.EvidenceRefs) ||
		!reflect.DeepEqual(binding.ContractRef, row.ContractRef) {
		return out, fmt.Errorf("BATCH_BINDING_EVIDENCE_MISMATCH: %s/%s/%s", row.PolicyID, row.RuleID, row.ContractKind)
	}
	for _, id := range basisRefs {
		item := source.Items[id]
		_, duplicate := out.BasisValues[id]
		if item.ID == "" || duplicate {
			return out, fmt.Errorf("BATCH_BINDING_SOURCE_CONFLICT: %s/%s", row.PolicyID, id)
		}
		out.Basis = append(out.Basis, item.SourceItem)
		out.BasisValues[id] = item.Value
	}
	if _, present := out.BasisValues[rule.SourceRef]; !present {
		return out, fmt.Errorf("BATCH_RULE_BASIS_REQUIRED: %s/%s", row.PolicyID, row.RuleID)
	}
	out.RuleSource, out.RuleSourceValue = ruleItem.SourceItem, ruleItem.Value
	out.BindingState, out.ContractReference = binding.State, binding.ContractRef
	err = r.resolveContract(row, &out)
	return out, err
}

var reverseBasisSections = map[string][]string{
	"RA": {"development_basis"},
	"RAC": {"policy_basis"},
	"AEC": {"policy_basis", "execution_boundary"},
}

func gitItem(doc map[string]any, address string) (any, error) {
	if address == "#" {
		address = ""
	}
	return review.Pointer(doc, address)
}

func (r *BatchEvidenceResolver) resolveContract(row Decision, out *BatchEvidence) error {
	contract := r.snapshot.Contracts[row.ContractKind]
	if !strings.HasPrefix(contract.SourceRef, ".ptsip/contracts/compile-sealed-rbac/") {
		return fmt.Errorf("BATCH_CONTRACT_OUTSIDE_REGISTERED_ROOT: %s", row.ContractKind)
	}
	registry, err := r.document(r.snapshot.SourceHead, ".ptsip/contracts/policy/unified/applicability-registry.json")
	if err != nil {
		return err
	}
	if registry["status"] != "CANDIDATE_READ_ONLY" ||
		asString(asMap(registry["contract_sources"])[row.ContractKind]) != contract.SourceRef {
		return fmt.Errorf("BATCH_CONTRACT_SOURCE_NOT_REGISTERED: %s", row.ContractKind)
	}
	current, err := r.document(r.snapshot.SourceHead, contract.SourceRef)
	if err != nil {
		return err
	}
	out.ActivationState = asString(current["activation_state"])
	if out.ActivationState == "" || out.ActivationState != contract.ActivationState {
		return fmt.Errorf("BATCH_CONTRACT_ACTIVATION_MISMATCH: %s", row.ContractKind)
	}
	for _, section := range reverseBasisSections[row.ContractKind] {
		for name, raw := range asMap(current[section]) {
			basis := asMap(raw)
			address, hasAddress := basis["item_ref"].(string)
			if asString(basis["source_ref"]) != row.PolicyPath || !hasAddress || address != row.SourcePointer {
				continue
			}
			ref := Reference{SourceRef: row.PolicyPath, ItemRef: address, RevisionRef: asString(basis["revision_ref"])}
			if _, _, err := r.pinnedItem(ref); err != nil {
				return fmt.Errorf("BATCH_REVERSE_BASIS_UNRESOLVED: %s: %w", name, err)
			}
			name = strings.NewReplacer("~", "~0", "/", "~1").Replace(name)
			out.ReverseBasis = append(out.ReverseBasis, "/"+section+"/"+name)
		}
	}
	slices.Sort(out.ReverseBasis)
	ref := out.ContractReference
	if ref == nil {
		return nil
	}
	if ref.SourceRef != contract.SourceRef {
		return fmt.Errorf("BATCH_CONTRACT_REFERENCE_OWNER_MISMATCH: %s", ref.SourceRef)
	}
	pinned, item, err := r.pinnedItem(*ref)
	if err != nil {
		return err
	}
	out.ContractItemValue = item
	switch ref.ItemRef {
	case "", "#":
		out.ContractItemClass = ContractItemWholeDefinition
	default:
		out.ContractItemClass = ContractItemExactInactive
		for _, section := range reverseBasisSections[row.ContractKind] {
			if ref.ItemRef == "/"+section || strings.HasPrefix(ref.ItemRef, "/"+section+"/") {
				out.ContractItemClass = ContractItemReverseBasis
				return nil
			}
		}
		currentItem, currentErr := gitItem(current, ref.ItemRef)
		if currentErr == nil && reflect.DeepEqual(item, currentItem) &&
			out.ActivationState == "REGISTERED" && pinned["activation_state"] == "REGISTERED" &&
			asString(asMap(item)["contract_id"]) != "" && asMap(item)["activation_state"] == "ACTIVE" {
			out.ContractItemClass = ContractItemExactActive
		}
	}
	return nil
}

func (r *BatchEvidenceResolver) pinnedItem(ref Reference) (map[string]any, any, error) {
	if !exactRef(ref) {
		return nil, nil, fmt.Errorf("BATCH_REFERENCE_NOT_PINNED: %s", ref.SourceRef)
	}
	doc, err := r.document(strings.TrimPrefix(ref.RevisionRef, "git:"), ref.SourceRef)
	if err != nil {
		return nil, nil, err
	}
	item, err := gitItem(doc, ref.ItemRef)
	if err != nil {
		return nil, nil, fmt.Errorf("BATCH_PINNED_ITEM_UNRESOLVED: %s#%s: %w", ref.SourceRef, ref.ItemRef, err)
	}
	return doc, item, nil
}
