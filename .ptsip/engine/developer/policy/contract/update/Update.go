// Package update prepares and persists explicitly authorized Unified Policy edits.
package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/check"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
	"gopkg.in/yaml.v3"
)

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Request replaces the value of each named existing content item. Baseline
// bytes are pinned separately from the canonical policy source digest.
type Request struct {
	Target         string                     `json:"target"`
	BaselineSHA256 string                     `json:"baseline_file_sha256"`
	ApprovalRef    string                     `json:"approval_ref"`
	Values         map[string]json.RawMessage `json:"values"`
}

// Approval describes the actual materialization. A trusted owner verifies
// the reference against the repository, policy identity, and exact delta.
type Approval struct {
	Reference      string   `json:"reference"`
	Root           string   `json:"root"`
	Target         string   `json:"target"`
	PolicyID       string   `json:"policy_id"`
	PolicyKind     string   `json:"policy_kind"`
	RootFamily     string   `json:"root_family"`
	BaselineSHA256 string   `json:"baseline_file_sha256"`
	ProposedSHA256 string   `json:"proposed_file_sha256"`
	SourceBefore   string   `json:"source_sha256_before"`
	SourceAfter    string   `json:"source_sha256_after"`
	ChangedItems   []string `json:"changed_items"`
	RequestSHA256  string   `json:"request_sha256"`
}

// Authorizer is supplied by the trusted policy owner, never by the request.
type Authorizer func(context.Context, Approval) error

type Result struct {
	Status       string   `json:"status"`
	Target       string   `json:"target"`
	ChangedItems []string `json:"changed_items"`
	DigestBefore string   `json:"source_sha256_before"`
	DigestAfter  string   `json:"source_sha256_after"`
}

type fileSnapshot struct {
	data []byte
	info os.FileInfo
}

func readSnapshot(name string) (fileSnapshot, error) {
	file, err := os.Open(name)
	if err != nil {
		return fileSnapshot{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > source.MaxBytes {
		return fileSnapshot{}, fmt.Errorf("POLICY_UPDATE_REQUIRES_BOUNDED_REGULAR_FILE")
	}
	raw, err := io.ReadAll(io.LimitReader(file, source.MaxBytes+1))
	if err != nil {
		return fileSnapshot{}, err
	}
	if len(raw) > source.MaxBytes {
		return fileSnapshot{}, fmt.Errorf("POLICY_UPDATE_REQUIRES_BOUNDED_REGULAR_FILE")
	}
	return fileSnapshot{data: raw, info: info}, nil
}

func bytesDigest(raw []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

// Apply verifies the existing source before deriving a digest. Historical
// mismatches require a separate reconciliation operation.
func Apply(ctx context.Context, root string, request Request, authorize Authorizer) (Result, error) {
	if authorize == nil {
		return Result{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_REQUIRED")
	}
	prepared, err := prepare(root, request)
	if err != nil {
		return Result{}, err
	}
	if err := authorize(ctx, prepared.approval); err != nil {
		return Result{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_REJECTED: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	root = prepared.approval.Root
	result := prepared.result
	if len(result.ChangedItems) == 0 {
		if err := confirmBaseline(root, request.Target, prepared.prior); err != nil {
			return Result{}, err
		}
		return result, nil
	}
	if err := persist(root, request.Target, prepared.name, prepared.prior, prepared.proposed, prepared.document); err != nil {
		return Result{}, err
	}
	result.Status = "UPDATED"
	return result, nil
}

type preparedUpdate struct {
	approval Approval
	result   Result
	name     string
	prior    fileSnapshot
	proposed []byte
	document source.Document
}

// prepare derives the exact approval coordinates without writing any files.
func prepare(root string, request Request) (preparedUpdate, error) {
	if strings.TrimSpace(request.ApprovalRef) == "" {
		return preparedUpdate{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_REQUIRED")
	}
	if !sha256Pattern.MatchString(request.BaselineSHA256) || len(request.Values) == 0 {
		return preparedUpdate{}, fmt.Errorf("POLICY_UPDATE_REQUIRES_BASELINE_AND_CONTENT_DELTA")
	}
	root, err := canonicalRepositoryRoot(root)
	if err != nil {
		return preparedUpdate{}, err
	}
	name, err := authoring.LocalFile(root, request.Target)
	if err != nil {
		return preparedUpdate{}, err
	}
	prior, err := readSnapshot(name)
	if err != nil {
		return preparedUpdate{}, err
	}
	if bytesDigest(prior.data) != request.BaselineSHA256 {
		return preparedUpdate{}, fmt.Errorf("STALE_POLICY_UPDATE_BASELINE")
	}
	document, err := source.Decode(prior.data)
	if err != nil {
		return preparedUpdate{}, err
	}
	selection, err := entry.SelectPolicyDocument(root, document)
	if err != nil {
		return preparedUpdate{}, err
	}
	if !entry.IsUnifiedPolicy(document) || selection.SchemaIDs["public"] != entry.UnifiedPolicySchemaID {
		return preparedUpdate{}, fmt.Errorf("POLICY_UPDATE_REQUIRES_REGISTERED_UNIFIED_POLICY")
	}
	identity := source.Object(document["Policy_Identity"])
	content := source.Object(document["Policy_Content"])
	sourceDocument := source.Object(content["source_document"])
	id := source.String(identity["id"])
	extension := filepath.Ext(request.Target)
	if !strings.HasPrefix(request.Target, selection.OutputRoot+"/") {
		return preparedUpdate{}, fmt.Errorf("POLICY_TARGET_OUTSIDE_REGISTERED_ROUTE")
	}
	if filepath.Base(id) != id || (extension != ".yaml" && extension != ".yml") ||
		identity["policy_kind"] != selection.PolicyKind || identity["root_family"] != selection.RootFamily ||
		sourceDocument["path"] != request.Target ||
		request.Target != selection.OutputRoot+"/"+selection.RootFamily+"/"+id+extension {
		return preparedUpdate{}, fmt.Errorf("POLICY_UPDATE_TARGET_IDENTITY_MISMATCH")
	}
	if _, err := check.CheckUnifiedDocument(root, document); err != nil {
		return preparedUpdate{}, fmt.Errorf("POLICY_UPDATE_BASELINE_REJECTED: %w", err)
	}
	result := Result{Status: "UNCHANGED", Target: request.Target,
		DigestBefore: source.String(sourceDocument["sha256"])}
	proposed, err := applyValues(prior.data, document, request.Values, &result)
	if err != nil {
		return preparedUpdate{}, err
	}
	if _, err := check.CheckUnifiedDocument(root, document); err != nil {
		return preparedUpdate{}, fmt.Errorf("POLICY_UPDATE_CANDIDATE_REJECTED: %w", err)
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return preparedUpdate{}, err
	}
	approval := Approval{Reference: request.ApprovalRef, Root: root, Target: request.Target,
		PolicyID: id, PolicyKind: selection.PolicyKind, RootFamily: selection.RootFamily,
		BaselineSHA256: request.BaselineSHA256, ProposedSHA256: bytesDigest(proposed),
		SourceBefore: result.DigestBefore, SourceAfter: result.DigestAfter, ChangedItems: slices.Clone(result.ChangedItems),
		RequestSHA256: bytesDigest(requestBytes)}
	return preparedUpdate{approval: approval, result: result, name: name, prior: prior, proposed: proposed, document: document}, nil
}

func canonicalRepositoryRoot(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func applyValues(raw []byte, document source.Document, values map[string]json.RawMessage, result *Result) ([]byte, error) {
	var tree yaml.Node
	if err := yaml.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}
	content := source.Object(document["Policy_Content"])
	records := source.Array(content["items"])
	contentNode := member(tree.Content[0], "Policy_Content")
	itemsNode := member(contentNode, "items")
	if itemsNode == nil || itemsNode.Kind != yaml.SequenceNode || len(itemsNode.Content) != len(records) {
		return nil, fmt.Errorf("UNSUPPORTED_POLICY_CONTENT_SERIALIZATION")
	}
	seen := make(map[string]bool, len(values))
	for i, record := range records {
		item := source.Object(record)
		id := source.String(item["id"])
		replacement, found := values[id]
		if !found {
			continue
		}
		seen[id] = true
		value, node, err := parseValue(replacement)
		if err != nil {
			return nil, fmt.Errorf("POLICY_UPDATE_INVALID_ITEM_VALUE %s: %w", id, err)
		}
		if reflect.DeepEqual(item["value"], value) {
			continue
		}
		existing := member(itemsNode.Content[i], "value")
		if existing == nil || existing.Anchor != "" {
			return nil, fmt.Errorf("UNSUPPORTED_POLICY_ITEM_SERIALIZATION: %s", id)
		}
		node.HeadComment, node.LineComment, node.FootComment = existing.HeadComment, existing.LineComment, existing.FootComment
		*existing = *node
		item["value"] = value
		result.ChangedItems = append(result.ChangedItems, id)
	}
	if len(seen) != len(values) {
		return nil, fmt.Errorf("POLICY_UPDATE_UNKNOWN_CONTENT_ITEM")
	}
	if len(result.ChangedItems) == 0 {
		result.DigestAfter = result.DigestBefore
		return raw, nil
	}
	digest, err := source.ReconstructedDigest(document)
	if err != nil {
		return nil, err
	}
	result.DigestAfter = digest
	source.Object(content["source_document"])["sha256"] = digest
	digestNode := member(member(contentNode, "source_document"), "sha256")
	if digestNode == nil || digestNode.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("UNSUPPORTED_POLICY_DIGEST_SERIALIZATION")
	}
	digestNode.Value = digest
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(&tree); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	if err := checkMaterialization(encoded.Bytes(), document); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func parseValue(raw json.RawMessage) (any, *yaml.Node, error) {
	if len(raw) > source.MaxBytes {
		return nil, nil, fmt.Errorf("POLICY_UPDATE_VALUE_TOO_LARGE")
	}
	if _, err := jsondoc.Decode(raw); err != nil {
		return nil, nil, err
	}
	wrapped := make([]byte, 0, len(raw)+10)
	wrapped = append(wrapped, `{"value":`...)
	wrapped = append(wrapped, raw...)
	wrapped = append(wrapped, '}')
	values, err := source.Decode(wrapped)
	if err != nil {
		return nil, nil, err
	}
	var tree yaml.Node
	if err := yaml.Unmarshal(wrapped, &tree); err != nil {
		return nil, nil, err
	}
	return values["value"], member(tree.Content[0], "value"), nil
}

func member(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func checkMaterialization(raw []byte, expected source.Document) error {
	observed, err := source.Decode(raw)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(observed, expected) {
		return fmt.Errorf("POLICY_UPDATE_MATERIALIZATION_MISMATCH")
	}
	return nil
}

func confirmBaseline(root, relative string, expected fileSnapshot) error {
	name, err := authoring.LocalFile(root, relative)
	if err != nil {
		return fmt.Errorf("STALE_POLICY_UPDATE_BASELINE: %w", err)
	}
	observed, err := readSnapshot(name)
	if err != nil {
		return err
	}
	if !os.SameFile(observed.info, expected.info) || !bytes.Equal(observed.data, expected.data) {
		return fmt.Errorf("STALE_POLICY_UPDATE_BASELINE")
	}
	return nil
}

func persist(root, relative, name string, prior fileSnapshot, data []byte, expected source.Document) error {
	if err := confirmBaseline(root, relative, prior); err != nil {
		return err
	}
	// Cooperating writers are serialized. Recheck bytes and file identity
	// under the lock, since authorization may have arrived while it was edited.
	lockName := name + ".ptsip-update.lock"
	lock, err := os.OpenFile(lockName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("POLICY_UPDATE_LOCK_UNAVAILABLE: %w", err)
	}
	defer os.Remove(lockName)
	if err := lock.Close(); err != nil {
		return err
	}
	proposed, err := stage(name, prior.info.Mode().Perm(), data)
	if err != nil {
		return err
	}
	defer os.Remove(proposed)
	recovery, err := stage(name, prior.info.Mode().Perm(), prior.data)
	if err != nil {
		return err
	}
	keepRecovery := false
	defer func() {
		if !keepRecovery {
			_ = os.Remove(recovery)
		}
	}()
	staged, err := readSnapshot(proposed)
	if err != nil {
		return err
	}
	proposedInfo := staged.info
	if !bytes.Equal(staged.data, data) {
		return fmt.Errorf("POLICY_UPDATE_STAGED_BYTES_MISMATCH")
	}
	if err := confirmBaseline(root, relative, prior); err != nil {
		return err
	}
	if err := replaceFile(proposed, name); err != nil {
		return err
	}
	readback, err := readSnapshot(name)
	if err == nil && (!os.SameFile(readback.info, proposedInfo) || !bytes.Equal(readback.data, data)) {
		err = fmt.Errorf("POLICY_UPDATE_READBACK_BYTES_MISMATCH")
	}
	if err == nil {
		err = checkMaterialization(readback.data, expected)
	}
	if err == nil {
		observed, decodeErr := source.Decode(readback.data)
		if decodeErr != nil {
			err = decodeErr
		} else {
			_, err = check.CheckUnifiedDocument(root, observed)
		}
	}
	if err == nil {
		err = syncDirectory(filepath.Dir(name))
	}
	if err == nil {
		return nil
	}
	current, currentErr := readSnapshot(name)
	if currentErr != nil || !os.SameFile(current.info, proposedInfo) || !bytes.Equal(current.data, data) {
		keepRecovery = true
		return fmt.Errorf("POLICY_UPDATE_READBACK_FAILED; RECOVERY_BLOCKED_BY_TARGET_CHANGE (%s): %w", filepath.Base(recovery), err)
	}
	if rollback := replaceFile(recovery, name); rollback != nil {
		keepRecovery = true
		return errors.Join(fmt.Errorf("POLICY_UPDATE_READBACK_FAILED: %w", err), fmt.Errorf("POLICY_UPDATE_RECOVERY_FAILED (%s): %w", filepath.Base(recovery), rollback))
	}
	return errors.Join(fmt.Errorf("POLICY_UPDATE_READBACK_FAILED_PRIOR_FILE_RESTORED: %w", err), syncDirectory(filepath.Dir(name)))
}

func stage(name string, mode os.FileMode, contents []byte) (string, error) {
	temp, err := os.CreateTemp(filepath.Dir(name), ".policy-update-*")
	if err != nil {
		return "", err
	}
	stageName := temp.Name()
	_, writeErr := temp.Write(contents)
	err = errors.Join(writeErr, temp.Chmod(mode), temp.Sync(), temp.Close())
	if err != nil {
		_ = os.Remove(stageName)
		return "", err
	}
	return stageName, nil
}
