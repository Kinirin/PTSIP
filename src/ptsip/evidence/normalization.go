// Package evidence normalizes observations into the existing evidence-only format.
package evidence

import (
	"cmp"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

const format = "ptsip-normalized-evidence/v1"

type SnapshotBinding struct {
	RepositoryRoot            string  `json:"repository_root"`
	Revision                  *string `json:"revision"`
	StatusFingerprint         string  `json:"status_fingerprint"`
	TrackedContentFingerprint string  `json:"tracked_content_fingerprint"`
}

type SourceGenerationBinding struct {
	ProfilePath           string `json:"profile_path"`
	Version               string `json:"version"`
	SpecificationRevision string `json:"specification_revision"`
	ContentSHA256         string `json:"content_sha256"`
}

type EvaluationContext struct {
	EvaluationID     string                   `json:"evaluation_id"`
	Snapshot         SnapshotBinding          `json:"snapshot"`
	SourceGeneration *SourceGenerationBinding `json:"source_generation"`
}

func BuildEvaluationContext(snapshot SnapshotBinding, generation *SourceGenerationBinding) (EvaluationContext, error) {
	digest, err := digestJSON(map[string]any{"snapshot": snapshot, "source_generation": generation})
	if err != nil {
		return EvaluationContext{}, err
	}
	return EvaluationContext{"evaluation:" + digest[:24], snapshot, generation}, nil
}

type Origin struct {
	Provenance     string   `json:"provenance"`
	Adapter        string   `json:"adapter"`
	EvidenceID     string   `json:"evidence_id"`
	SourcePath     *string  `json:"source_path"`
	Line           *int     `json:"line"`
	DocumentSHA256 *string  `json:"document_sha256"`
	ProducerID     *string  `json:"producer_id"`
	BasisIDs       []string `json:"basis_ids"`
	Detail         *string  `json:"detail"`
}

type Fact struct {
	Subject    string
	Predicate  string
	Qualifiers map[string]json.RawMessage
	Value      json.RawMessage
	Origin     Origin
}

type Assertion struct {
	ID      string          `json:"id"`
	Value   json.RawMessage `json:"value"`
	Origins []Origin        `json:"origins"`
}

type Record struct {
	SemanticID string                     `json:"semantic_id"`
	Subject    string                     `json:"subject"`
	Predicate  string                     `json:"predicate"`
	Qualifiers map[string]json.RawMessage `json:"qualifiers"`
	Assertions []Assertion                `json:"assertions"`
	Status     string                     `json:"status"`
	Authority  string                     `json:"authority"`
}

type Channel struct {
	ID            string  `json:"id"`
	Status        string  `json:"status"`
	EvidenceCount int     `json:"evidence_count"`
	Reason        *string `json:"reason"`
}

type Issue struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Channel     *string  `json:"channel"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type Content struct {
	Format   string            `json:"format"`
	Context  EvaluationContext `json:"context"`
	Records  []Record          `json:"records"`
	Channels []Channel         `json:"channels"`
	Issues   []Issue           `json:"issues"`
}

type NormalizedSet struct {
	Content
	DeterministicDigest string `json:"deterministic_digest"`
	RecordCount         int    `json:"record_count"`
	ConflictCount       int    `json:"conflict_count"`
}

type evidenceGroup struct {
	record Record
	// Canonical value -> canonical origin -> origin.
	assertions map[string]map[string]Origin
}

func Normalize(facts []Fact, context EvaluationContext, channels []Channel, issues []Issue) (NormalizedSet, error) {
	groups := make(map[string]*evidenceGroup)
	for _, fact := range facts {
		qualifiers := make(map[string]json.RawMessage, len(fact.Qualifiers))
		for key, value := range fact.Qualifiers {
			canonical, err := canonicalJSON(value)
			if err != nil {
				return NormalizedSet{}, fmt.Errorf("evidence %q qualifier %q: %w", fact.Origin.EvidenceID, key, err)
			}
			qualifiers[key] = json.RawMessage(canonical)
		}
		digest, err := digestJSON(map[string]any{"subject": fact.Subject, "predicate": fact.Predicate, "qualifiers": qualifiers})
		if err != nil {
			return NormalizedSet{}, err
		}
		semanticID := "evidence:" + digest[:24]
		group := groups[semanticID]
		if group == nil {
			group = &evidenceGroup{
				record: Record{SemanticID: semanticID, Subject: fact.Subject, Predicate: fact.Predicate,
					Qualifiers: qualifiers, Authority: "EVIDENCE_ONLY", Status: "CONSISTENT"},
				assertions: make(map[string]map[string]Origin),
			}
			groups[semanticID] = group
		}
		value, err := canonicalJSON(fact.Value)
		if err != nil {
			return NormalizedSet{}, fmt.Errorf("evidence %q value: %w", fact.Origin.EvidenceID, err)
		}
		origin := fact.Origin
		origin.BasisIDs = append([]string{}, origin.BasisIDs...)
		slices.Sort(origin.BasisIDs)
		origin.BasisIDs = slices.Compact(origin.BasisIDs)
		key, err := canonicalJSON(origin)
		if err != nil {
			return NormalizedSet{}, err
		}
		if group.assertions[value] == nil {
			group.assertions[value] = make(map[string]Origin)
		}
		group.assertions[value][key] = origin
	}

	content := Content{
		Format: format, Context: context, Records: make([]Record, 0, len(groups)),
		Channels: append([]Channel{}, channels...), Issues: append([]Issue{}, issues...),
	}
	slices.SortStableFunc(content.Channels, func(a, b Channel) int { return cmp.Compare(a.ID, b.ID) })
	for n := range content.Issues {
		content.Issues[n].EvidenceIDs = append([]string{}, content.Issues[n].EvidenceIDs...)
	}
	slices.SortStableFunc(content.Issues, compareIssues)

	conflicts := 0
	for _, id := range slices.Sorted(maps.Keys(groups)) {
		group := groups[id]
		for _, value := range slices.Sorted(maps.Keys(group.assertions)) {
			rawValue := json.RawMessage(value)
			digest, err := digestJSON(map[string]any{"semantic_id": id, "value": rawValue})
			if err != nil {
				return NormalizedSet{}, err
			}
			assertion := Assertion{ID: "assertion:" + digest[:24], Value: rawValue}
			origins := group.assertions[value]
			for _, key := range slices.Sorted(maps.Keys(origins)) {
				assertion.Origins = append(assertion.Origins, origins[key])
			}
			group.record.Assertions = append(group.record.Assertions, assertion)
		}
		if len(group.record.Assertions) > 1 {
			group.record.Status = "CONFLICT"
			conflicts++
		}
		content.Records = append(content.Records, group.record)
	}
	digest, err := digestJSON(content)
	if err != nil {
		return NormalizedSet{}, err
	}
	return NormalizedSet{content, digest, len(content.Records), conflicts}, nil
}

func compareIssues(a, b Issue) int {
	var aChannel, bChannel string
	if a.Channel != nil {
		aChannel = *a.Channel
	}
	if b.Channel != nil {
		bChannel = *b.Channel
	}
	return cmp.Or(cmp.Compare(aChannel, bChannel), cmp.Compare(a.Code, b.Code),
		cmp.Compare(a.Message, b.Message), slices.Compare(a.EvidenceIDs, b.EvidenceIDs))
}

func digestJSON(value any) (string, error) {
	raw, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(raw))), nil
}
