package update_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/check"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/update"
)

func buildDeveloperCLI(t *testing.T, settings string) string {
	t.Helper()
	module, err := filepath.Abs("../../../../../.ptsip/engine/developer")
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "ptsip-developer")
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cmd := exec.Command("go", "-C", module, "build", "-mod=readonly", "-ldflags", settings, "-o", name, ".")
	if raw, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building developer CLI: %v\n%s", err, raw)
	}
	return name
}

type cliReport struct {
	Status              string          `json:"status"`
	Problem             string          `json:"problem"`
	Approval            update.Approval `json:"approval"`
	Result              update.Result   `json:"result"`
	ExecutionAuthorized bool            `json:"execution_authorized"`
	NormativeApproval   bool            `json:"normative_approval"`
}

func runDeveloperCLI(t *testing.T, binary, root string, options ...string) (cliReport, int) {
	t.Helper()
	args := append([]string{"--validator", "policy-rules", "--operation", "update", "--root", root, "--input", "update-request.json"}, options...)
	cmd := exec.Command(binary, args...)
	raw, err := cmd.Output()
	var exit int
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			exit = failure.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	var report cliReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode CLI report (exit %d): %v\n%s", exit, err, raw)
	}
	if report.ExecutionAuthorized || report.NormativeApproval {
		t.Fatalf("policy update issued unrelated authority: %+v", report)
	}
	return report, exit
}

func marshal(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func signedEvidence(t *testing.T, signer ed25519.PrivateKey, approval update.DirectUserApproval) update.SignedDirectUserApproval {
	t.Helper()
	payload := marshal(t, approval)
	return update.SignedDirectUserApproval{
		Payload:   base64.StdEncoding.EncodeToString(payload),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(signer, payload)),
	}
}

func TestDirectUserApprovalThroughDeveloperCLI(t *testing.T) {
	root, target, raw := policyFixture(t, "developer")
	req := request(target, raw, "SRC-0003", `{"message":"Direct user fixture","limit":3}`)
	const requestFile = "update-request.json"
	put(t, root, requestFile, marshal(t, req))

	pub, signer, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, foreignSigner, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	store := t.TempDir()
	evidenceDir := t.TempDir()
	const evidenceFile = "approval.json"
	path := filepath.Join(evidenceDir, evidenceFile)
	settings := fmt.Sprintf(`-X main.policyUpdateApprovalPublicKey=%s -X "main.policyUpdateApprovalReplayDirectory=%s"`,
		base64.StdEncoding.EncodeToString(pub), filepath.ToSlash(store))
	binary := buildDeveloperCLI(t, settings)

	preview, code := runDeveloperCLI(t, binary, root, "--dry-run")
	if code != 0 || preview.Status != "PREPARED_NOT_AUTHORIZED" ||
		preview.Approval.Target != target || preview.Approval.RequestSHA256 == "" ||
		preview.Approval.ProposedSHA256 == req.BaselineSHA256 || !bytes.Equal(raw, read(t, root, target)) {
		t.Fatalf("non-mutating preview failed: %+v", preview)
	}
	now := time.Now().UTC()
	approval := update.DirectUserApproval{
		Format: update.DirectUserApprovalFormat, IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
		Approval: preview.Approval,
	}
	writeEvidence := func(evidence update.SignedDirectUserApproval) {
		put(t, evidenceDir, evidenceFile, marshal(t, evidence))
	}
	requireReject := func(cli, problem string, options ...string) {
		t.Helper()
		report, code := runDeveloperCLI(t, cli, root, options...)
		if code != 2 || !strings.Contains(report.Problem, problem) || !bytes.Equal(raw, read(t, root, target)) {
			t.Fatalf("expected %s and original bytes, got exit %d: %+v", problem, code, report)
		}
	}
	requireReject(binary, "POLICY_UPDATE_APPROVAL_EVIDENCE_REQUIRED")
	writeEvidence(signedEvidence(t, signer, approval))
	unconfigured := buildDeveloperCLI(t, "")
	t.Setenv("PTSIP_POLICY_UPDATE_APPROVAL_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pub))
	t.Setenv("PTSIP_POLICY_UPDATE_APPROVAL_REPLAY_DIRECTORY", store)
	requireReject(unconfigured, "POLICY_UPDATE_APPROVAL_PROVIDER_UNAVAILABLE", "--approval-evidence", path)
	writeEvidence(signedEvidence(t, foreignSigner, approval))
	requireReject(binary, "POLICY_UPDATE_APPROVAL_SIGNATURE_INVALID", "--approval-evidence", path)

	tampered := signedEvidence(t, signer, approval)
	alternative := approval
	alternative.Approval.PolicyID = "MPD-CNTR-0002"
	tampered.Payload = base64.StdEncoding.EncodeToString(marshal(t, alternative))
	writeEvidence(tampered)
	requireReject(binary, "POLICY_UPDATE_APPROVAL_SIGNATURE_INVALID", "--approval-evidence", path)

	for _, tc := range []struct {
		name    string
		alter   func(*update.DirectUserApproval)
		problem string
	}{
		{"expired", func(a *update.DirectUserApproval) { a.ExpiresAt = now.Add(-time.Second) }, "POLICY_UPDATE_APPROVAL_EVIDENCE_STALE"},
		{"future", func(a *update.DirectUserApproval) { a.IssuedAt = now.Add(time.Hour) }, "POLICY_UPDATE_APPROVAL_EVIDENCE_STALE"},
		{"wrong-root", func(a *update.DirectUserApproval) { a.Approval.Root = filepath.Dir(root) }, ""},
		{"wrong-target", func(a *update.DirectUserApproval) { a.Approval.Target += ".bak" }, ""},
		{"wrong-policy-id", func(a *update.DirectUserApproval) { a.Approval.PolicyID = "MPD-CNTR-0002" }, ""},
		{"wrong-kind", func(a *update.DirectUserApproval) { a.Approval.PolicyKind = "PTSIP_SUPPORT_FEATURE" }, ""},
		{"wrong-family", func(a *update.DirectUserApproval) { a.Approval.RootFamily = "GOV" }, ""},
		{"wrong-baseline", func(a *update.DirectUserApproval) { a.Approval.BaselineSHA256 = strings.Repeat("0", 64) }, ""},
		{"wrong-proposal", func(a *update.DirectUserApproval) { a.Approval.ProposedSHA256 = strings.Repeat("0", 64) }, ""},
		{"wrong-source-before", func(a *update.DirectUserApproval) { a.Approval.SourceBefore = strings.Repeat("0", 64) }, ""},
		{"wrong-source-after", func(a *update.DirectUserApproval) { a.Approval.SourceAfter = strings.Repeat("0", 64) }, ""},
		{"wrong-changed-items", func(a *update.DirectUserApproval) { a.Approval.ChangedItems = []string{"SRC-0001"} }, ""},
		{"wrong-request", func(a *update.DirectUserApproval) { a.Approval.RequestSHA256 = strings.Repeat("0", 64) }, ""},
		{"wrong-approval-reference", func(a *update.DirectUserApproval) { a.Approval.Reference = "a-different-reference" }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wrong := approval
			tc.alter(&wrong)
			writeEvidence(signedEvidence(t, signer, wrong))
			problem := tc.problem
			if problem == "" {
				problem = "POLICY_UPDATE_APPROVAL_BINDING_MISMATCH"
			}
			requireReject(binary, problem, "--approval-evidence", path)
		})
	}

	writeEvidence(signedEvidence(t, signer, approval))
	changed := request(target, raw, "SRC-0003", `{"message":"Different request","limit":4}`)
	put(t, root, requestFile, marshal(t, changed))
	requireReject(binary, "POLICY_UPDATE_APPROVAL_BINDING_MISMATCH", "--approval-evidence", path)
	put(t, root, requestFile, marshal(t, req))

	report, code := runDeveloperCLI(t, binary, root, "--approval-evidence", path)
	updated := read(t, root, target)
	if code != 0 || report.Status != "UPDATED" || bytes.Equal(raw, updated) ||
		report.Result.DigestAfter != approval.Approval.SourceAfter {
		t.Fatalf("valid direct approval was not applied: %+v", report)
	}
	if _, err := check.CheckUnifiedDocument(root, document(t, updated)); err != nil {
		t.Fatal(err)
	}
	report, code = runDeveloperCLI(t, binary, root, "--approval-evidence", path)
	if code != 2 || !strings.Contains(report.Problem, "STALE_POLICY_UPDATE_BASELINE") ||
		!bytes.Equal(updated, read(t, root, target)) {
		t.Fatalf("stale baseline replay changed the file: %+v", report)
	}
	put(t, root, target, raw)
	requireReject(binary, "POLICY_UPDATE_APPROVAL_EVIDENCE_REPLAYED", "--approval-evidence", path)
	if entries, err := os.ReadDir(store); err != nil || len(entries) != 1 {
		t.Fatalf("expected one consumed approval across CLI invocations: %v, %v", entries, err)
	}
}
