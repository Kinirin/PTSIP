package contracts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func profileSessionNativePath(installed profileRuntime, kind string) string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return filepath.Join(installed.scripts, "ptsip-engine-"+kind+suffix)
}

func profileSessionRawCommand(t *testing.T, executable string, args []string, input []byte) ([]object, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("native session failed to terminate at EOF or rejected input: %v", ctx.Err())
	}
	var responses []object
	decoder := json.NewDecoder(&stdout)
	for {
		var response object
		if decodeErr := decoder.Decode(&response); decodeErr == io.EOF {
			break
		} else if decodeErr != nil {
			t.Fatalf("session emitted invalid JSON: %v (process %v), stderr %s", decodeErr, err, stderr.String())
		}
		responses = append(responses, response)
	}
	return responses, stderr.String(), err
}

func sessionEnvelopeBytes(t *testing.T, kind, operation string, input object) []byte {
	t.Helper()
	raw, err := json.Marshal(profileEnvelope(t, kind, operation, input))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestProfileSessionRequestBoundaries(t *testing.T) {
	const limit = 1 << 20
	installed := installedProfileRuntime(t, "validation")
	const operation = "validation.profile.rules"
	input := object{"records": []any{object{"ID": "opaque", "rule_value": ""}}}
	base := sessionEnvelopeBytes(t, "validation", operation, input)
	input["records"].([]any)[0].(object)["rule_value"] = strings.Repeat("a", limit-len(base))
	exact := sessionEnvelopeBytes(t, "validation", operation, input)
	if len(exact) != limit {
		t.Fatalf("boundary test must provide exactly %d bytes, got %d", limit, len(exact))
	}
	for _, transport := range []string{"native", "router"} {
		t.Run(transport, func(t *testing.T) {
			executable := profileSessionNativePath(installed, "validation")
			args := []string{"--serve"}
			if transport == "router" {
				executable = installed.cli
				args = []string{"engine-serve", installed.packageRoot, installed.scripts, operation}
			}
			for _, ending := range []string{"\n", "\r\n"} {
				responses, stderr, err := profileSessionRawCommand(t, executable, args, append(append([]byte{}, exact...), []byte(ending)...))
				if err != nil || len(responses) != 1 || responses[0]["status"] != "PASS" {
					t.Fatalf("exact request-byte boundary with ending %q was rejected: %v, responses=%d, stderr=%s", ending, err, len(responses), stderr)
				}
			}
			for _, extra := range []int{1, 8192} {
				oversized := append(append([]byte{}, exact...), bytes.Repeat([]byte{' '}, extra)...)
				oversized = append(oversized, '\n')
				responses, stderr, err := profileSessionRawCommand(t, executable, args, oversized)
				if err == nil {
					t.Fatalf("oversized %d-byte session request was accepted", limit+extra)
				}
				if transport == "native" {
					if len(responses) != 1 || responses[0]["status"] != "FAIL" || !strings.Contains(stderr, "byte limit") {
						t.Fatalf("native byte-limit rejection was not explicit: responses=%v stderr=%s", responses, stderr)
					}
				} else if len(responses) != 0 || !(strings.Contains(stderr, "transport limit") || strings.Contains(stderr, "token too long")) {
					t.Fatalf("router did not reject the bounded line before native dispatch: responses=%d stderr=%s", len(responses), stderr)
				}
			}
		})
	}
}

func TestProfileNativeSessionMalformedLineContinuation(t *testing.T) {
	cases := []struct {
		kind, operation string
		input           object
	}{
		{"discovery", "discovery.profile.rules", object{"records": []any{object{"ID": "opaque"}}}},
		{"validation", "validation.profile.rules", object{"records": []any{object{"ID": "opaque"}}}},
		{"projection", "projection.profile.metadata", object{"profile_contract_version": "pp.2.00", "specification_source": "https://github.com/Kinirin/PTSIP", "specification_revision": strings.Repeat("a", 40), "schema_ref": "urn:profile:selected"}},
	}
	for _, item := range cases {
		t.Run(item.kind, func(t *testing.T) {
			installed := installedProfileRuntime(t, item.kind)
			valid := sessionEnvelopeBytes(t, item.kind, item.operation, item.input)
			duplicate := append([]byte(`{"operation_id":"`+item.operation+`",`), valid[1:]...)
			lines := append([]byte("{\"operation_id\":\n"), duplicate...)
			lines = append(lines, '\n')
			lines = append(lines, valid...)
			lines = append(lines, '\n')
			responses, stderr, err := profileSessionRawCommand(t, profileSessionNativePath(installed, item.kind), []string{"--serve"}, lines)
			if err != nil || len(responses) != 3 {
				t.Fatalf("native session did not continue after malformed and duplicate JSON: %v, responses=%d, stderr=%s", err, len(responses), stderr)
			}
			for index, expected := range []string{"FAIL", "FAIL", "PASS"} {
				if responses[index]["status"] != expected {
					t.Fatalf("line %d status=%v, expected %s, native diagnostic=%v", index, responses[index]["status"], expected, responses[index]["error"])
				}
			}
		})
	}
}

func TestProfileRouterSessionOwnershipAndDuplicateJSON(t *testing.T) {
	installed := installedProfileRuntime(t, "validation")
	const operation = "validation.profile.rules"
	valid := sessionEnvelopeBytes(t, "validation", operation, object{"records": []any{object{"ID": "opaque"}}})
	duplicate := append([]byte(`{"operation_id":"`+operation+`",`), valid[1:]...)
	lines := append(duplicate, '\n')
	lines = append(lines, valid...)
	lines = append(lines, '\n')
	args := []string{"engine-serve", installed.packageRoot, installed.scripts, operation}
	responses, stderr, err := profileSessionRawCommand(t, installed.cli, args, lines)
	if err != nil || len(responses) != 2 || responses[0]["status"] != "FAIL" || responses[1]["status"] != "PASS" {
		t.Fatalf("router changed or dropped duplicate-key request instead of preserving native diagnostics: %v responses=%v stderr=%s", err, responses, stderr)
	}
	wrongOwner := sessionEnvelopeBytes(t, "discovery", "discovery.profile.rules", object{"records": []any{}})
	responses, stderr, err = profileSessionRawCommand(t, installed.cli, args, append(wrongOwner, '\n'))
	if err == nil || len(responses) != 0 || !strings.Contains(stderr, "differs from the selected route") {
		t.Fatalf("wrong-operation envelope reached the selected native owner: %v responses=%v stderr=%s", err, responses, stderr)
	}
	unknownArgs := []string{"engine-serve", installed.packageRoot, installed.scripts, "validation.profile.unknown"}
	responses, stderr, err = profileSessionRawCommand(t, installed.cli, unknownArgs, append(valid, '\n'))
	if err == nil || len(responses) != 0 || !strings.Contains(stderr, "unregistered engine operation") {
		t.Fatalf("unregistered session operation was admitted: %v responses=%v stderr=%s", err, responses, stderr)
	}
	wrongEnvelope := profileEnvelope(t, "discovery", operation, object{"records": []any{}})
	wrongRaw, _ := json.Marshal(wrongEnvelope)
	lines = append(wrongRaw, '\n')
	lines = append(lines, valid...)
	lines = append(lines, '\n')
	responses, stderr, err = profileSessionRawCommand(t, profileSessionNativePath(installed, "validation"), []string{"--serve"}, lines)
	if err != nil || len(responses) != 2 || responses[0]["status"] != "FAIL" || responses[1]["status"] != "PASS" {
		t.Fatalf("native owner/version mismatch did not fail and preserve subsequent valid requests: %v responses=%v stderr=%s", err, responses, stderr)
	}
}

func TestProfileSessionUnsupportedNativeKinds(t *testing.T) {
	for _, item := range []struct{ kind, operation string }{{"storage", "storage.mediate"}, {"context", "context.snapshot"}} {
		t.Run(item.kind, func(t *testing.T) {
			installed := installedProfileRuntime(t, item.kind)
			responses, stderr, err := profileSessionRawCommand(t, installed.cli,
				[]string{"engine-serve", installed.packageRoot, installed.scripts, item.operation}, nil)
			if err == nil || len(responses) != 0 || !strings.Contains(stderr, "no registered persistent session transport") {
				t.Fatalf("undeclared persistent native kind was admitted: %v responses=%v stderr=%s", err, responses, stderr)
			}
			responses, stderr, err = profileSessionRawCommand(t, profileSessionNativePath(installed, item.kind), []string{"--serve"}, nil)
			if err == nil || len(responses) != 1 || responses[0]["status"] != "FAIL" {
				t.Fatalf("unsupported native --serve argument was admitted: %v responses=%v stderr=%s", err, responses, stderr)
			}
		})
	}
}
