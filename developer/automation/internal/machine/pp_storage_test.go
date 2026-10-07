package machine

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func ppMoveStorageFixture(t *testing.T, r *Repository, files map[string][]byte) {
	t.Helper()
	for path, raw := range files {
		if strings.HasPrefix(path, "profiles/") {
			target := PPSourceRoot + strings.TrimPrefix(path, "profiles")
			if path == PPCatalog {
				raw = bytes.Replace(raw, []byte("root: profiles"), []byte("root: "+PPSourceRoot), 1)
			}
			if err := r.AtomicWrite(target, raw, nil); err != nil {
				t.Fatal(err)
			}
			original, err := r.Path(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(original); err != nil {
				t.Fatal(err)
			}
		}
	}
	registry, err := ppYAML(files[PPRegistry], PPRegistry, true)
	if err != nil {
		t.Fatal(err)
	}
	registry["profile_source"] = Object{"root": PPSourceRoot, "catalog": PPSourceRoot + "/index.yaml", "history_root": PPSourceRoot + "/history", "history_distribution": "SOURCE_ONLY"}
	for _, raw := range List(registry["contracts"]) {
		row := Map(raw)
		if baseline := Text(row["baseline"]); baseline != "" {
			row["baseline"] = "src/ptsip/" + baseline
		}
	}
	data, err := yaml.Marshal(registry)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{PPRegistry, PPEmbeddedRegistry} {
		if err := r.AtomicWrite(path, data, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPPStorageMovePreservesIdentityHistoryAndFutureTransitions(t *testing.T) {
	r, files := ppRetirementFixture(t)
	ppMoveStorageFixture(t, r, files)
	if _, err := ppGit(r.Root, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	plan, err := r.BuildPPTransition("HEAD")
	if err != nil || plan.Status != "NO_CHANGE" || plan.Target != "pp.1.02" || len(plan.Outputs) != 0 {
		t.Fatal(plan, err)
	}
	if _, err := ppGit(r.Root, "commit", "-m", "move physical profile storage"); err != nil {
		t.Fatal(err)
	}
	result, err := r.VerifyPPCommit("HEAD")
	if err != nil || result["classification"] != "NO_T2_AUTHORITY_DELTA" || result["current"] != "pp.1.02" {
		t.Fatal(result, err)
	}
	path := PPSourceRoot + "/sample.ptsip.yaml"
	if err := r.AtomicWrite(path, bytes.ReplaceAll(files["profiles/sample.ptsip.yaml"], []byte("original"), []byte("changed")), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ppGit(r.Root, "add", "--", path); err != nil {
		t.Fatal(err)
	}
	plan, err = r.ReconcilePP(true)
	if err != nil || plan.Status != "RECONCILED" || plan.Target != "pp.1.03" {
		t.Fatal(plan, err)
	}
	if _, err := ppGit(r.Root, "commit", "-m", "semantic transition at new source root"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.VerifyPPCommit("HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.VerifyPPRange("HEAD~2", "HEAD"); err != nil {
		t.Fatal(err)
	}
}

func TestPPStorageMoveFailsClosedWithoutRewritingHistoryOrRegistry(t *testing.T) {
	for _, defect := range []string{"history_bytes", "history_removed", "history_added", "obsolete_history", "registry_semantics", "catalog_semantics", "unsupported_root", "mixed_baseline"} {
		t.Run(defect, func(t *testing.T) {
			r, files := ppRetirementFixture(t)
			ppMoveStorageFixture(t, r, files)
			ref := PPSourceRoot + "/history/pp.1.02/sample.ptsip.yaml"
			switch defect {
			case "history_bytes":
				if err := r.AtomicWrite(ref, append(files["profiles/sample.ptsip.yaml"], []byte("# altered\n")...), nil); err != nil {
					t.Fatal(err)
				}
			case "history_removed":
				path, _ := r.Path(ref)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "history_added", "obsolete_history":
				if defect == "history_added" {
					ref = PPSourceRoot + "/history/pp.1.02/extra.ptsip.yaml"
				} else {
					ref = "profiles/history/pp.1.02/sample.ptsip.yaml"
				}
				if err := r.AtomicWrite(ref, files["profiles/sample.ptsip.yaml"], nil); err != nil {
					t.Fatal(err)
				}
			case "catalog_semantics":
				catalog, _ := r.Read(PPSourceRoot + "/index.yaml")
				catalog["authority"] = "OTHER"
				policyTestWrite(t, r, PPSourceRoot+"/index.yaml", catalog)
			default:
				registry, _ := r.Read(PPRegistry)
				switch defect {
				case "registry_semantics":
					Map(List(registry["contracts"])[0])["lifecycle"] = "SUPERSEDED"
				case "unsupported_root":
					Map(registry["profile_source"])["root"] = "guessed/profiles"
				case "mixed_baseline":
					Map(List(registry["contracts"])[0])["baseline"] = "profiles/history/pp.1.02"
				}
				policyTestWrite(t, r, PPRegistry, registry)
				policyTestWrite(t, r, PPEmbeddedRegistry, registry)
			}
			if _, err := ppGit(r.Root, "add", "-A"); err != nil {
				t.Fatal(err)
			}
			before, err := ppGit(r.Root, "write-tree")
			if err != nil {
				t.Fatal(err)
			}
			if plan, err := r.ReconcilePP(true); err == nil {
				t.Fatal("invalid move accepted", plan)
			}
			after, err := ppGit(r.Root, "write-tree")
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("rejected move changed staged data", err)
			}
		})
	}
}
