package releasetest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func TestNewDraftUsesReturnedIDWithoutRediscoveringItInList(t *testing.T) {
	tag := "ptsip-release-" + strings.Repeat("a", 40)
	calls := []string{}
	api := func(method, endpoint string, payload any, out any) error {
		calls = append(calls, method+" "+endpoint)
		switch {
		case method == "GET" && strings.Contains(endpoint, "?per_page="):
			return json.Unmarshal([]byte(`[]`), out)
		case method == "POST":
			p := payload.(map[string]any)
			if p["tag_name"] != tag || p["draft"] != true {
				t.Fatal("creation not bound to verified draft tag")
			}
			if _, ok := p["target_commitish"]; ok {
				t.Fatal("verified historical tag redundantly requested workflow mutation permission")
			}
			data, _ := json.Marshal(release.Carrier{ID: 42, Tag: tag, Draft: true})
			return json.Unmarshal(data, out)
		case method == "GET" && strings.HasSuffix(endpoint, "/42"):
			data, _ := json.Marshal(release.Carrier{ID: 42, Tag: tag, Draft: true})
			return json.Unmarshal(data, out)
		default:
			return fmt.Errorf("unexpected API call: %s %s", method, endpoint)
		}
	}
	c, err := release.ResolveCarrier(tag, strings.Repeat("a", 40), "title", "body", 0, api)
	if err != nil || c.ID != 42 || len(calls) != 3 {
		t.Fatalf("creation did not continue by returned ID: %v %v", calls, err)
	}
}

func TestDraftResumeRejectsWrongSourceAndPreservesExistingAssets(t *testing.T) {
	tag := "ptsip-release-" + strings.Repeat("b", 40)
	for _, wrong := range []bool{false, true} {
		calls := 0
		api := func(method, endpoint string, payload any, out any) error {
			calls++
			if method != "GET" || !strings.HasSuffix(endpoint, "/99") {
				t.Fatal("resume created or selected another release")
			}
			actual := tag
			if wrong {
				actual = "ptsip-release-" + strings.Repeat("c", 40)
			}
			data, _ := json.Marshal(release.Carrier{ID: 99, Tag: actual, Draft: true, Assets: []release.Asset{{Name: "original.whl", Digest: "sha256:" + strings.Repeat("d", 64)}}})
			return json.Unmarshal(data, out)
		}
		c, err := release.ResolveCarrier(tag, strings.Repeat("b", 40), "title", "body", 99, api)
		if wrong && err == nil {
			t.Fatal("foreign source draft admitted")
		}
		if !wrong && (err != nil || len(c.Assets) != 1 || calls != 1) {
			t.Fatal("same-source draft could not resume unchanged")
		}
	}
}

func TestDuplicateSourceCarriersFailWithoutMutation(t *testing.T) {
	tag := "ptsip-release-" + strings.Repeat("a", 40)
	api := func(method, endpoint string, payload any, out any) error {
		if method != "GET" {
			t.Fatal("ambiguous carrier caused mutation")
		}
		data, _ := json.Marshal([]release.Carrier{{ID: 1, Tag: tag}, {ID: 2, Tag: tag}})
		return json.Unmarshal(data, out)
	}
	if _, err := release.ResolveCarrier(tag, strings.Repeat("a", 40), "title", "body", 0, api); err == nil {
		t.Fatal("ambiguous source carrier admitted")
	}
}

func TestEngineReuseSeparatesArtifactProducerFromPublicationRouting(t *testing.T) {
	root := t.TempDir()
	contract, err := os.ReadFile(filepath.Join(repoRoot(t), release.ContractPath))
	if err != nil {
		t.Fatal(err)
	}
	write := func(ref, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(ref))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(release.ContractPath, string(contract))
	producer := ".github/checks/release/engine-verification/wheel.go"
	publisher := ".github/checks/release/engine-verification/publication_bundle.go"
	write(producer, "producer-v1")
	write(publisher, "policy-owner-1")
	run := func(args ...string) string {
		t.Helper()
		data, err := release.Run(root, args...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(data))
	}
	run("git", "init", "--quiet")
	run("git", "config", "user.name", "fixture")
	run("git", "config", "user.email", "fixture@example.invalid")
	run("git", "add", ".")
	run("git", "commit", "--quiet", "-m", "fixture")
	origin := run("git", "rev-parse", "HEAD")
	write(publisher, "policy-owner-5")
	run("git", "add", ".")
	run("git", "commit", "--quiet", "-m", "route")
	delta, err := release.EngineSourceChanges(root, origin, "HEAD", "context")
	if err != nil || delta != "" {
		t.Fatalf("publication routing changed engine identity: %s %v", delta, err)
	}
	write(producer, "producer-v2")
	run("git", "add", ".")
	run("git", "commit", "--quiet", "-m", "producer")
	delta, err = release.EngineSourceChanges(root, origin, "HEAD", "context")
	if err != nil || !strings.Contains(delta, "wheel.go") {
		t.Fatalf("artifact producer change was ignored: %s %v", delta, err)
	}
}

func TestDistinctPendingPublisherEnvironmentsPreserveSingleWorkflow(t *testing.T) {
	root := repoRoot(t)
	seen := map[string]bool{}
	for _, c := range []release.Component{{ID: "tool", Kind: "tool"}, {ID: "context", Kind: "engine"}, {ID: "storage", Kind: "engine"}, {ID: "validation", Kind: "engine"}, {ID: "discovery", Kind: "engine"}, {ID: "planning", Kind: "engine"}, {ID: "execution", Kind: "engine"}, {ID: "projection", Kind: "engine"}} {
		environment, err := release.PublisherEnvironment(root, c)
		if err != nil || environment == "" || seen[environment] {
			t.Fatalf("ambiguous pending publisher identity: %s %v", environment, err)
		}
		seen[environment] = true
	}
}
