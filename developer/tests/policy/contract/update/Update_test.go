package update_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/check"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/update"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
	"go.yaml.in/yaml/v3"
)

func policyFixture(t *testing.T, plane string) (string, string, []byte) {
	t.Helper()
	repository, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	paths := []string{
		".ptsip/index.json",
		".ptsip/contracts/index.json",
		".ptsip/contracts/policy/boundary.json",
		".ptsip/contracts/policy/routing/policy_routes.json",
		".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json",
		".ptsip/contracts/policy/unified/index.json",
		".ptsip/contracts/policy/unified/Unified_Policy.schema.json",
		".ptsip/contracts/policy/unified/Evaluation_Interface.json",
		".ptsip/contracts/policy/unified/evaluation-registry.json",
		".ptsip/contracts/policy/unified/authoring-decision.json",
	}
	for _, relative := range paths {
		raw, err := os.ReadFile(filepath.Join(repository, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		put(t, root, relative, raw)
	}
	base, id, kind := "developer/policy", "MPD-CNTR-0001", "PTSIP_DEVELOPER_POLICY"
	if plane == "support" {
		base, id, kind = "src/policy", "SFP-CNTR-0001", "PTSIP_SUPPORT_FEATURE"
	}
	target := base + "/CNTR/" + id + ".yaml"
	identity := source.Document{"id": id, "policy_kind": kind, "root_family": "CNTR", "title": "Synthetic policy update"}
	lifecycle := source.Document{"status": "DRAFT"}
	semantics := source.Document{"message": "Original unit", "limit": 2}
	original := source.Document{"Policy_Identity": identity, "Policy_Lifecycle": lifecycle,
		"authority_semantics": source.Document{"unit": semantics}}
	digest, err := source.Digest(original)
	if err != nil {
		t.Fatal(err)
	}
	binding := source.Document{"state": "UNRESOLVED", "basis_refs": []any{"SRC-0003"}}
	doc := source.Document{
		"schema_version":       source.Version,
		"Policy_Identity":      identity,
		"Policy_Lifecycle":     source.Document{"status": "DRAFT", "source_ref": "SRC-0002"},
		"Policy_Authority":     source.Document{"source_refs": []any{}},
		"Policy_Evaluation":    source.Document{"source_refs": []any{}, "state": "UNRESOLVED"},
		"Policy_Relationships": source.Document{"source_refs": []any{}},
		"Policy_Rules": source.Document{"rules": []any{source.Document{
			"id": id + "#/authority_semantics/unit", "kind": "SEMANTIC_UNIT", "identity_scheme": "SOURCE_COORDINATE", "source_ref": "SRC-0003",
			"contract_bindings": source.Document{"RA": binding, "RAC": binding, "AEC": binding},
		}}},
		"Policy_Content": source.Document{
			"items": []any{
				source.Document{"id": "SRC-0001", "source_pointer": "/Policy_Identity", "value": identity},
				source.Document{"id": "SRC-0002", "source_pointer": "/Policy_Lifecycle", "value": lifecycle},
				source.Document{"id": "SRC-0003", "source_pointer": "/authority_semantics/unit", "value": semantics},
			},
			"source_document": source.Document{"path": target, "sha256": digest},
		},
		"Policy_Migration": source.Document{"baseline_head": strings.Repeat("1", 40), "source_format": "synthetic", "source_registration": "INDEXED", "contract_state": "CANDIDATE"},
	}
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		t.Fatal(err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatal(err)
	}
	raw := encoded.Bytes()
	put(t, root, target, raw)
	return root, target, raw
}

func put(t *testing.T, root, relative string, raw []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, relative string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func request(target string, raw []byte, id string, replacement string) update.Request {
	return update.Request{
		Target: target, BaselineSHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), ApprovalRef: "fixture-approval",
		Values: map[string]json.RawMessage{id: json.RawMessage(replacement)},
	}
}

func allow(context.Context, update.Approval) error { return nil }

func document(t *testing.T, data []byte) source.Document {
	t.Helper()
	doc, err := source.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func expectFailure(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %s, got %v", expected, err)
	}
}

func TestAuthorizedPolicyContentAndCanonicalDigestMaterializeTogether(t *testing.T) {
	ctx := context.Background()
	for _, plane := range []string{"developer", "support"} {
		t.Run(plane, func(t *testing.T) {
			root, target, raw := policyFixture(t, plane)
			canonicalRoot, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			before := document(t, raw)
			selection, err := entry.Resolve(root, target)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := check.CheckUnifiedDocument(root, before); err != nil {
				t.Fatal(err)
			}
			if err := authoring.WriteDocument(root, target, before); err == nil {
				t.Fatal("new-file authoring overwrote an existing policy")
			}
			approved := false
			req := request(target, raw, "SRC-0003", `{"message":"Updated 단위","limit":3}`)
			result, err := update.Apply(ctx, root, req, func(_ context.Context, actual update.Approval) error {
				approved = true
				if actual.Target != target || actual.Root != canonicalRoot || actual.Reference != req.ApprovalRef ||
					actual.PolicyID != source.Object(before["Policy_Identity"])["id"] ||
					actual.PolicyKind != selection.PolicyKind || actual.RootFamily != selection.RootFamily ||
					actual.BaselineSHA256 != req.BaselineSHA256 || actual.ProposedSHA256 == actual.BaselineSHA256 ||
					actual.SourceBefore == actual.SourceAfter || !reflect.DeepEqual(actual.ChangedItems, []string{"SRC-0003"}) {
					t.Fatalf("approval not bound to exact update: %+v", actual)
				}
				if !bytes.Equal(raw, read(t, root, target)) {
					t.Fatal("policy changed before authorization")
				}
				return nil
			})
			if err != nil || !approved || result.Status != "UPDATED" {
				t.Fatalf("update=%+v, err=%v", result, err)
			}
			updated := read(t, root, target)
			after := document(t, updated)
			if _, err := check.CheckUnifiedDocument(root, after); err != nil {
				t.Fatal(err)
			}
			reconstructed, err := source.Restore(after)
			if err != nil {
				t.Fatal(err)
			}
			digest, err := source.Digest(reconstructed)
			if err != nil || result.DigestAfter != digest || result.DigestBefore == digest {
				t.Fatalf("source digest=%s, update=%+v, err=%v", digest, result, err)
			}
			actual := source.Object(source.Object(reconstructed["authority_semantics"])["unit"])
			if actual["message"] != "Updated 단위" || actual["limit"] != 3 {
				t.Fatalf("wrong applied values: %+v", actual)
			}
			for _, field := range []string{"Policy_Identity", "Policy_Lifecycle", "Policy_Rules", "Policy_Authority", "Policy_Evaluation", "Policy_Relationships", "Policy_Migration"} {
				if !reflect.DeepEqual(before[field], after[field]) {
					t.Fatalf("unapproved change to %s", field)
				}
			}
			noOp, err := update.Apply(ctx, root, request(target, updated, "SRC-0003", `{"limit":3,"message":"Updated 단위"}`), allow)
			if err != nil || noOp.Status != "UNCHANGED" || !bytes.Equal(updated, read(t, root, target)) {
				t.Fatalf("replay was not byte-stable: %+v %v", noOp, err)
			}
			second, err := update.Apply(ctx, root, request(target, updated, "SRC-0003", `{"message":"Again","limit":4}`), allow)
			if err != nil || second.DigestBefore != result.DigestAfter || second.DigestAfter == result.DigestAfter {
				t.Fatalf("repeated edit was not rebound to current digest: %+v %v", second, err)
			}
		})
	}
}

func TestStaleOrUnauthorizedPolicyUpdateDoesNotPersist(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, code string
		run        func(*testing.T, string, string, []byte) error
	}{
		{"missing-provider", "POLICY_UPDATE_APPROVAL_REQUIRED", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-0003", `{"limit":3}`), nil)
			return err
		}},
		{"rejected-owner", "POLICY_UPDATE_APPROVAL_REJECTED", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-0003", `{"limit":3}`), func(context.Context, update.Approval) error { return errors.New("delta exceeds approval") })
			return err
		}},
		{"stale-bytes", "STALE_POLICY_UPDATE_BASELINE", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, []byte("different bytes"), "SRC-0003", `{"limit":3}`), allow)
			return err
		}},
		{"bad-content", "POLICY_UPDATE_INVALID_ITEM_VALUE", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-0003", `{"limit":3,"limit":4}`), allow)
			return err
		}},
		{"unknown-item", "POLICY_UPDATE_UNKNOWN_CONTENT_ITEM", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-9999", `{"limit":3}`), allow)
			return err
		}},
		{"changed-lifecycle", "SOURCE_POLICY_STATUS_MISMATCH", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-0002", `{"status":"APPROVED.Applying"}`), allow)
			return err
		}},
		{"changed-identity", "SOURCE_POLICY_IDENTITY_MISMATCH", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-0001", `{"id":"OTHER"}`), allow)
			return err
		}},
		{"changed-rule-id", "POLICY_RULE_ID_SOURCE_MISMATCH", func(t *testing.T, root, target string, raw []byte) error {
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-0003", `{"id":"SOME-R-ID"}`), allow)
			return err
		}},
		{"competing-writer", "POLICY_UPDATE_LOCK_UNAVAILABLE", func(t *testing.T, root, target string, raw []byte) error {
			put(t, root, target+".ptsip-update.lock", nil)
			_, err := update.Apply(ctx, root, request(target, raw, "SRC-0003", `{"limit":3}`), allow)
			return err
		}},
		{"cross-plane", "POLICY_TARGET_OUTSIDE_REGISTERED_ROUTE", func(t *testing.T, root, target string, raw []byte) error {
			foreign := "src/policy/CNTR/" + filepath.Base(target)
			put(t, root, foreign, raw)
			_, err := update.Apply(ctx, root, request(foreign, raw, "SRC-0003", `{"limit":3}`), allow)
			if !bytes.Equal(raw, read(t, root, foreign)) {
				t.Fatal("foreign policy modified")
			}
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, target, raw := policyFixture(t, "developer")
			expectFailure(t, tc.run(t, root, target, raw), tc.code)
			if !bytes.Equal(raw, read(t, root, target)) {
				t.Fatal("rejected update changed baseline")
			}
		})
	}
}

func TestPolicyDigestMismatchRequiresSeparateRepair(t *testing.T) {
	root, target, raw := policyFixture(t, "support")
	drift := bytes.Replace(raw, []byte("Original unit"), []byte("Changed directly"), 1)
	if bytes.Equal(drift, raw) {
		t.Fatal("drift fixture was unchanged")
	}
	put(t, root, target, drift)
	_, err := check.CheckUnifiedDocument(root, document(t, drift))
	expectFailure(t, err, "SOURCE_DIGEST_MISMATCH")
	_, err = update.Apply(context.Background(), root, request(target, drift, "SRC-0003", `{"message":"New unit","limit":3}`), func(context.Context, update.Approval) error {
		t.Fatal("stale policy admitted for ordinary update")
		return nil
	})
	expectFailure(t, err, "POLICY_UPDATE_BASELINE_REJECTED: UNIFIED_POLICY_REJECTED: SOURCE_DIGEST_MISMATCH")
	if !bytes.Equal(drift, read(t, root, target)) {
		t.Fatal("stale policy was rewritten")
	}
}

func TestUpdateOperationRequiresTrustedProviderAndExactRequest(t *testing.T) {
	root, target, raw := policyFixture(t, "developer")
	req := request(target, raw, "SRC-0003", `{"limit":3}`)
	input, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	const requestFile = "update-request.json"
	put(t, root, requestFile, input)
	report, code := update.Run(context.Background(), root, requestFile, nil)
	if code != 2 || report.(map[string]any)["problem"] != "POLICY_UPDATE_APPROVAL_PROVIDER_UNAVAILABLE" {
		t.Fatalf("untrusted CLI accepted a request: %v", report)
	}
	unauthorized := bytes.Replace(input, []byte(`"values":`), []byte(`"policy_lifecycle":{"status":"APPROVED"},"values":`), 1)
	put(t, root, requestFile, unauthorized)
	report, code = update.Run(context.Background(), root, requestFile, func(context.Context, update.Approval) error {
		t.Fatal("unexpected field reached the authorizer")
		return nil
	})
	if code != 2 || !strings.Contains(report.(map[string]any)["problem"].(string), "unknown field") ||
		!bytes.Equal(raw, read(t, root, target)) {
		t.Fatalf("CLI did not reject an unbound lifecycle change: %v", report)
	}
	put(t, root, requestFile, input)
	report, code = update.Run(context.Background(), root, requestFile, allow)
	if code != 0 || report.(map[string]any)["status"] != "UPDATED" {
		t.Fatalf("authorized CLI request failed: %v", report)
	}
}

func TestPolicyUpdateRejectsConcurrentFileAndRecoversAfterFailedReadback(t *testing.T) {
	ctx := context.Background()
	t.Run("concurrent edit", func(t *testing.T) {
		root, target, raw := policyFixture(t, "developer")
		external := append(bytes.Clone(raw), []byte("\n# concurrent edit\n")...)
		_, err := update.Apply(ctx, root, request(target, raw, "SRC-0003", `{"limit":3}`), func(context.Context, update.Approval) error {
			put(t, root, target, external)
			return nil
		})
		expectFailure(t, err, "STALE_POLICY_UPDATE_BASELINE")
		if !bytes.Equal(external, read(t, root, target)) {
			t.Fatal("concurrent policy edit was lost")
		}
	})
	t.Run("verification registration changes after approval", func(t *testing.T) {
		root, target, raw := policyFixture(t, "support")
		_, err := update.Apply(ctx, root, request(target, raw, "SRC-0003", `{"limit":3}`), func(context.Context, update.Approval) error {
			put(t, root, ".ptsip/contracts/policy/unified/Unified_Policy.schema.json", []byte(`{"$id":"changed"}`))
			return nil
		})
		expectFailure(t, err, "POLICY_UPDATE_READBACK_FAILED_PRIOR_FILE_RESTORED")
		if !bytes.Equal(raw, read(t, root, target)) {
			t.Fatal("baseline was lost after failed independent readback")
		}
	})
}
