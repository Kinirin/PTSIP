package machine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPPDeltaPreservesEveryPythonTransitionVectorWithoutPythonRuntime(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(r.Root, "developer/automation/internal/machine/testdata/pp_delta_protocol_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		ProjectionAuthority bool   `json:"projection_authority"`
		Role                string `json:"fixture_role"`
		Vectors             []struct {
			Name      string            `json:"name"`
			Base      *PPAuthorityState `json:"base"`
			Candidate *PPAuthorityState `json:"candidate"`
			OldSchema []byte            `json:"old_schema"`
			Expected  PPDelta           `json:"expected"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.ProjectionAuthority || fixture.Role != "FROZEN_PRE_RETIREMENT_PROTOCOL_EQUIVALENCE" || len(fixture.Vectors) != 9 {
		t.Fatal("frozen protocol coverage or evidence boundary changed")
	}
	for _, vector := range fixture.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			result, err := EvaluatePPDelta(vector.Base, vector.Candidate, vector.OldSchema)
			if err != nil || !reflect.DeepEqual(result, vector.Expected) {
				t.Fatalf("PP semantics changed: %v\nactual=%#v\nexpected=%#v", err, result, vector.Expected)
			}
		})
	}
}

func ppRetirementFixture(t *testing.T) (*Repository, map[string][]byte) {
	t.Helper()
	r := planningFixture(t, false)
	files := ppTestFiles(t)
	for ref, raw := range files {
		if err := r.AtomicWrite(ref, raw, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.invalid"}, {"config", "user.name", "PTSIP Test"}, {"config", "core.hooksPath", "disabled-hooks"}, {"add", "."}, {"commit", "-m", "baseline"}} {
		if _, err := ppGit(r.Root, args...); err != nil {
			t.Fatal(err)
		}
	}
	return r, files
}

func TestPPReconciliationPreservesSchemaHistoryAndUnstagedWriteGuards(t *testing.T) {
	for _, defect := range []string{"schema_change", "unstaged_output", "manual_registry", "historical_baseline"} {
		t.Run(defect, func(t *testing.T) {
			r, files := ppRetirementFixture(t)
			profileRef := "profiles/sample.ptsip.yaml"
			if defect == "schema_change" {
				ref := ppSchemaPath("pp.1.02")
				var schema Object
				if err := json.Unmarshal(files[ref], &schema); err != nil {
					t.Fatal(err)
				}
				schema["x-semantic-change"] = "changed"
				raw, err := json.Marshal(schema)
				if err != nil {
					t.Fatal(err)
				}
				if err := r.AtomicWrite(ref, raw, nil); err != nil {
					t.Fatal(err)
				}
				if _, err := ppGit(r.Root, "add", "--", ref); err != nil {
					t.Fatal(err)
				}
			} else {
				profile := bytes.ReplaceAll(files[profileRef], []byte("original"), []byte("changed"))
				if err := r.AtomicWrite(profileRef, profile, nil); err != nil {
					t.Fatal(err)
				}
				if _, err := ppGit(r.Root, "add", "--", profileRef); err != nil {
					t.Fatal(err)
				}
			}
			switch defect {
			case "unstaged_output":
				path, _ := r.Path(profileRef)
				raw, _ := os.ReadFile(path)
				if err := r.AtomicWrite(profileRef, append(raw, []byte("\n# unstaged user edit\n")...), nil); err != nil {
					t.Fatal(err)
				}
			case "manual_registry":
				if err := r.AtomicWrite(PPRegistry, bytes.ReplaceAll(files[PPRegistry], []byte("pp.1.02"), []byte("pp.1.03")), nil); err != nil {
					t.Fatal(err)
				}
				if _, err := ppGit(r.Root, "add", "--", PPRegistry); err != nil {
					t.Fatal(err)
				}
			case "historical_baseline":
				ref := ppHistory("pp.1.02") + "/sample.ptsip.yaml"
				if err := r.AtomicWrite(ref, append(files[ref], []byte("\n# forbidden history edit\n")...), nil); err != nil {
					t.Fatal(err)
				}
				if _, err := ppGit(r.Root, "add", "--", ref); err != nil {
					t.Fatal(err)
				}
			}
			before, err := ppGit(r.Root, "diff", "--binary", "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			result, err := r.ReconcilePP(true)
			if defect == "schema_change" {
				if err != nil || result.Target != "pp.1.03" || result.Status != "RECONCILED" {
					t.Fatal(result, err)
				}
				path, _ := r.Path(ppSchemaPath("pp.1.02"))
				restored, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(restored, files[ppSchemaPath("pp.1.02")]) {
					t.Fatal("original schema generation was not restored", err)
				}
				path, _ = r.Path(ppSchemaPath("pp.1.03"))
				generated, err := os.ReadFile(path)
				if err != nil || !bytes.Contains(generated, []byte(`"x-semantic-change": "changed"`)) {
					t.Fatal("schema semantics did not move to the new generation", err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid staged transition or user write conflict accepted", defect, result)
			}
			after, err := ppGit(r.Root, "diff", "--binary", "HEAD")
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("rejected reconciliation changed user files or staged content", err)
			}
		})
	}
}

func TestPPRemoteAndReleaseVerificationUseExactHeadWithoutMutation(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	before, err := ppGit(r.Root, "status", "--porcelain=v1")
	if err != nil {
		t.Fatal(err)
	}
	headRaw, err := ppGit(r.Root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(string(headRaw))
	result, err := r.VerifyPPCommit("HEAD")
	if err != nil || result["commit"] != head || len(head) != 40 {
		t.Fatal(result, err)
	}
	parentRaw, err := ppGit(r.Root, "rev-parse", "HEAD^")
	if err != nil {
		t.Fatal(err)
	}
	parent := strings.TrimSpace(string(parentRaw))
	rangeResult, err := r.VerifyPPRange(parent, head)
	if err != nil || len(List(rangeResult["commits"])) != 1 || Map(List(rangeResult["commits"])[0])["commit"] != head {
		t.Fatal(rangeResult, err)
	}
	release, err := r.VerifyPPRelease("HEAD")
	if err != nil || release["status"] != "PASS" || release["source_sha"] != head {
		t.Fatal(release, err)
	}
	if _, err := r.VerifyPPRelease(parent); err == nil {
		t.Fatal("non-HEAD release SHA accepted")
	}
	after, err := ppGit(r.Root, "status", "--porcelain=v1")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("verify-only operations mutated the repository", err)
	}
}
