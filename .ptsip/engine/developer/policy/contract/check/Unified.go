package check

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

type Document = source.Document

func unifiedPolicyOperation(root string, doc Document) (Document, Document, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	if _, err := entry.SelectPolicyDocument(absolute, doc); err != nil {
		return nil, nil, err
	}
	verifier, err := source.NewVerifier(absolute)
	if err != nil {
		return nil, nil, err
	}
	report, original, err := verifier.Check(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("UNIFIED_POLICY_REJECTED: %w", err)
	}
	return report, original, nil
}

// CheckUnifiedDocument calls the shared candidate verifier. Structural validity
// does not approve semantics, lifecycle transitions, registration or execution.
func CheckUnifiedDocument(root string, doc Document) (Document, error) {
	report, _, err := unifiedPolicyOperation(root, doc)
	return report, err
}

// RestoreUnifiedSource is an explicit read-only historical view for consumers
// testing original contracts. ReadDocument always returns the canonical document.
func RestoreUnifiedSource(root string, doc Document) (Document, error) {
	if !entry.IsUnifiedPolicy(doc) {
		return cloneLegacyDocument(doc)
	}
	_, original, err := unifiedPolicyOperation(root, doc)
	return original, err
}

func cloneLegacyDocument(doc Document) (Document, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var clone Document
	if err := json.Unmarshal(raw, &clone); err != nil {
		return nil, err
	}
	if clone == nil {
		clone = Document{}
	}
	return clone, nil
}
