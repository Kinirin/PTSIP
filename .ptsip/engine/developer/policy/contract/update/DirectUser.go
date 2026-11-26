package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

const DirectUserApprovalFormat = "ptsip-direct-policy-update-approval/v1"

// DirectUserApproval is the signed payload. The envelope signs the raw JSON
// bytes of this value so verifiers do not depend on JSON member ordering.
type DirectUserApproval struct {
	Format    string    `json:"format"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Approval  Approval  `json:"approval"`
}

// SignedDirectUserApproval carries the standard base64 payload and detached
// Ed25519 signature. It contains no caller-selected key or trust source.
type SignedDirectUserApproval struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

// DirectUserAuthorizer verifies evidence from an admitted direct user's
// deployment-pinned key. The same operator manages the separate replay
// directory. Neither setting derives from a request or policy document.
func DirectUserAuthorizer(trustedKey, replayDirectory, root, evidencePath string) (Authorizer, error) {
	if trustedKey == "" || replayDirectory == "" {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_PROVIDER_UNAVAILABLE")
	}
	key, err := base64.StdEncoding.Strict().DecodeString(trustedKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_TRUST_KEY_INVALID")
	}
	if evidencePath == "" {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_REQUIRED")
	}
	dir, err := approvalReplayDirectory(replayDirectory, root)
	if err != nil {
		return nil, err
	}
	approval, err := readDirectUserApproval(ed25519.PublicKey(key), evidencePath)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, actual Approval) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := time.Now()
		if approval.IssuedAt.IsZero() || now.Before(approval.IssuedAt) || !now.Before(approval.ExpiresAt) {
			return fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_STALE")
		}
		if !reflect.DeepEqual(approval.Approval, actual) {
			return fmt.Errorf("POLICY_UPDATE_APPROVAL_BINDING_MISMATCH")
		}
		return claimDirectUserApproval(dir, actual.Reference)
	}, nil
}

func approvalReplayDirectory(dir, root string) (string, error) {
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("POLICY_UPDATE_APPROVAL_REPLAY_STORE_INVALID")
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("POLICY_UPDATE_APPROVAL_REPLAY_STORE_INVALID")
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("POLICY_UPDATE_APPROVAL_REPLAY_STORE_INVALID: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, dir)
	// An independent drive cannot be inside the worktree.
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("POLICY_UPDATE_APPROVAL_REPLAY_STORE_IN_WORKTREE")
	}
	return dir, nil
}

func claimDirectUserApproval(dir, reference string) error {
	name := filepath.Join(dir, bytesDigest([]byte(reference))+".used")
	marker, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_REPLAYED")
	}
	if err != nil {
		return fmt.Errorf("POLICY_UPDATE_APPROVAL_REPLAY_STORE_UNAVAILABLE: %w", err)
	}
	// Never remove a claim on a failed write or interrupted update. A retry
	// after consumption needs fresh direct approval even if the baseline was
	// recovered by the update's existing atomic-write logic.
	if err := errors.Join(marker.Sync(), marker.Close(), syncDirectory(dir)); err != nil {
		return fmt.Errorf("POLICY_UPDATE_APPROVAL_REPLAY_STORE_UNAVAILABLE: %w", err)
	}
	return nil
}

func readDirectUserApproval(key ed25519.PublicKey, path string) (DirectUserApproval, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return DirectUserApproval{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_NOT_REGULAR")
	}
	file, err := readSnapshot(path)
	if err != nil {
		return DirectUserApproval{}, err
	}
	var evidence SignedDirectUserApproval
	if err := decodeExactApprovalJSON(file.data, &evidence); err != nil {
		return DirectUserApproval{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_INVALID: %w", err)
	}
	payload, err := base64.StdEncoding.Strict().DecodeString(evidence.Payload)
	if err != nil {
		return DirectUserApproval{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_INVALID: %w", err)
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(evidence.Signature)
	if err != nil || !ed25519.Verify(key, payload, signature) {
		return DirectUserApproval{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_SIGNATURE_INVALID")
	}
	var approval DirectUserApproval
	if err := decodeExactApprovalJSON(payload, &approval); err != nil {
		return DirectUserApproval{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_INVALID: %w", err)
	}
	if approval.Format != DirectUserApprovalFormat {
		return DirectUserApproval{}, fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_FORMAT_UNSUPPORTED")
	}
	return approval, nil
}

// Reject duplicate, missing, case-aliased, and unrecognized fields in signed
// evidence. JSON size is bounded by its outer regular file snapshot.
func decodeExactApprovalJSON(raw []byte, value any) error {
	observed, err := jsondoc.Decode(raw)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(value); err != nil {
		return err
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return err
	}
	expected, err := jsondoc.Decode(normalized)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(observed, expected) {
		return fmt.Errorf("unexpected approval evidence fields or values")
	}
	return nil
}
