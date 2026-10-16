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

func TestProjectTokenScopeAllowsPackageAndRejectsStandaloneProjects(t *testing.T) {
	root := repoRoot(t)
	packageComponent := release.Component{ID: "package", Kind: "package"}
	if environment, err := release.PublisherEnvironment(root, packageComponent); err != nil || environment != "pypi" {
		t.Fatalf("registered token route unavailable: %s %v", environment, err)
	}
	if err := release.AdmitPublisherScope(root, release.Selection{Components: []release.Component{packageComponent}}, "github-pypi"); err != nil {
		t.Fatal(err)
	}
	if _, err := release.PublisherEnvironment(root, release.Component{ID: "tool", Kind: "tool"}); err == nil {
		t.Fatal("PTSIP-only token claimed the new standalone Tool project")
	}
	for _, id := range []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"} {
		s := release.Selection{Components: []release.Component{{ID: id, Kind: "engine"}}}
		if err := release.AdmitPublisherScope(root, s, "github-pypi"); err == nil {
			t.Fatalf("PTSIP-only token admitted engine project %s", id)
		}
		if err := release.AdmitPublisherScope(root, s, "github"); err != nil {
			t.Fatal("token scope blocked an authorized GitHub-only release")
		}
	}
	if err := release.AdmitPublisherScope(root, release.Selection{}, "unknown"); err == nil {
		t.Fatal("unregistered destination bypassed authentication admission")
	}
}
