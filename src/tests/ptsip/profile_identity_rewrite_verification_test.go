package contracts_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const profileIdentityRewriteVerification = "validation.profile.identity-rewrite"

func TestNativeProfileIdentityRewriteVerification(t *testing.T) {
	canonical := func(path string) string {
		t.Helper()
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		return resolved
	}
	packageRoot := canonical(installedProfileRuntime(t, "validation").packageRoot)
	write := func(path string, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	encode := func(instance object) string {
		t.Helper()
		bytes, err := json.MarshalIndent(instance, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		return "\ufeff# Profile rewrite verification fixture\r\n" + strings.ReplaceAll(string(bytes), "\n", "\r\n") + "\r\n"
	}
	sha := func(text string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(text))) }
	for _, edge := range []struct{ from, to, specification string }{
		{"pp.1.02", "pp.2.00", strings.Repeat("1", 40)},
		{"0.3.6-draft", "pp.1.01", "d6995ed232e845b88d8235b851e80ab54b7804ea"},
	} {
		t.Run(edge.from, func(t *testing.T) {
			root := canonical(t.TempDir())
			profile := filepath.Join(root, "ptsip.yaml")
			instance := historicalProfileInstance(edge.to)
			metadata := instance["ptsip"].(object)
			spec := metadata["specification"].(object)
			spec["revision"] = edge.specification
			if edge.to == "pp.1.01" {
				spec["family"] = edge.from
			}
			after := encode(instance)
			metadata["version"] = edge.from
			delete(spec, "family")
			before := encode(instance)
			write(profile, before)
			snapshot := object{
				"captured_at": "2026-11-22T00:00:00Z", "is_git": false, "head": nil,
				"status_fingerprint": sha(""), "tracked_content_fingerprint": sha(before),
				"tracked_files": 1, "observation_errors": array{},
			}
			plan := object{
				"kind": "PP_IDENTITY_ONLY_REWRITE", "source_path": "ptsip.yaml", "source_sha256": sha(before),
				"source_declared_version": edge.from, "target_contract": edge.to, "specification_family": edge.from,
				"specification_revision": edge.specification, "repository_snapshot": snapshot,
			}
			input := object{
				"root": root, "package_root": packageRoot, "plan": plan, "phase": "PROPOSED",
				"before_content": before, "after_content": after, "after_sha256": sha(after),
			}
			result := nativeProfileOutput(t, profileIdentityRewriteVerification, input)
			if result["valid"] != true || result["observed_sha256"] != sha(before) {
				t.Fatalf("exact candidate failed independent validation: %v", result)
			}
			observed, err := os.ReadFile(profile)
			if err != nil || string(observed) != before {
				t.Fatalf("validation changed the source profile: %v", err)
			}
			changed := strings.Replace(after, "An independent historical contract declaration", "Unauthorized field edit", 1)
			input["after_content"], input["after_sha256"] = changed, sha(changed)
			if result := nativeProfileOutput(t, profileIdentityRewriteVerification, input); result["valid"] != false {
				t.Fatalf("semantically different target was accepted: %v", result)
			}
			input["phase"], input["after_sha256"] = "POST_WRITE", sha(after)
			delete(input, "after_content")
			write(profile, after)
			if result := nativeProfileOutput(t, profileIdentityRewriteVerification, input); result["valid"] != true || result["observed_sha256"] != sha(after) {
				t.Fatalf("on-disk target failed independent validation: %v", result)
			}
			corruptPackage := canonical(t.TempDir())
			copyJSONTree(t, filepath.Join(packageRoot, "schemas"), filepath.Join(corruptPackage, "schemas"))
			schemaPath := filepath.Join(corruptPackage, "schemas", "storage", "profile", "history", "project-profile-"+edge.to+".schema.json")
			originalSchema, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatal(err)
			}
			write(schemaPath, string(originalSchema)+"\n")
			input["package_root"] = corruptPackage
			if result := nativeProfileOutput(t, profileIdentityRewriteVerification, input); result["valid"] != false ||
				!strings.Contains(fmt.Sprint(result["errors"]), "SHA-256 mismatch") {
				t.Fatalf("unpinned target schema bytes were accepted: %v", result)
			}
			input["package_root"] = packageRoot
			plan["target_contract"] = "pp.3.00"
			if result := nativeProfileOutput(t, profileIdentityRewriteVerification, input); result["valid"] != false {
				t.Fatalf("unregistered identity-only edge was accepted: %v", result)
			}
			plan["target_contract"] = edge.to
			write(profile, after+"# changed after evidence was bound\r\n")
			if result := nativeProfileOutput(t, profileIdentityRewriteVerification, input); result["valid"] != false {
				t.Fatalf("stale write evidence was accepted: %v", result)
			}
		})
	}
}
