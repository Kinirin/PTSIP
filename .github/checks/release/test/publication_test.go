package releasetest

import (
	"encoding/json"
	"fmt"
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
