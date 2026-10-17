package releasetest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func TestSingleEngineUploadStageContainsAllAndOnlyPendingAdmittedWheels(t *testing.T) {
	root := t.TempDir()
	write := func(ref string, b []byte) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(ref))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	route := ".github/checks/release/pypi-publishers.json"
	b, err := os.ReadFile(filepath.Join(repoRoot(t), route))
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the preserved legacy helper under a local fixture route. The
	// production publisher admits only PTSIP and never calls this helper.
	var legacy map[string]any
	if err := json.Unmarshal(b, &legacy); err != nil {
		t.Fatal(err)
	}
	scope := []string{"PTSIP"}
	for _, id := range []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"} {
		scope = append(scope, "ptsip-engine-"+id)
	}
	legacy["authentication"].(map[string]any)["declared_project_scope"] = scope
	b, err = json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	write(route, b)
	m := release.ShippingManifest{}
	for _, id := range []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"} {
		version := "0.3.1"
		if id == "context" || id == "projection" {
			version = "0.2.1"
		}
		m.Selection.Components = append(m.Selection.Components, release.Component{ID: id, Kind: "engine", Version: version})
		c := release.PublishedComponent{ID: id, Kind: "engine", Version: version}
		for _, host := range []string{"windows", "linux"} {
			name := "ptsip_engine_" + id + "-" + version + "-" + host + ".whl"
			data := []byte("admitted:" + name)
			c.Files = append(c.Files, release.File{Name: name, SHA256: release.Digest(data)})
			write("shipping/pypi/"+id+"/"+name, data)
		}
		m.Snapshot.Components = append(m.Snapshot.Components, c)
	}
	write("shipping/pypi/package/ptsip.whl", []byte("package must wait"))
	write("shipping/pypi/tool/ptsip_tool.whl", []byte("standalone tool is not an engine"))
	if count, err := release.StageEngineUploads(root, m, "shipping"); err != nil || count != 14 {
		t.Fatalf("single upload set: %d %v", count, err)
	}
	// A successful earlier upload is pruned by PyPIStage on rerun; it must not
	// survive through a stale copy in the aggregate directory.
	first := m.Snapshot.Components[0].Files[0].Name
	if err := os.Remove(filepath.Join(root, "shipping/pypi/context", first)); err != nil {
		t.Fatal(err)
	}
	if count, err := release.StageEngineUploads(root, m, "shipping"); err != nil || count != 13 {
		t.Fatalf("already public file remained in upload: %d %v", count, err)
	}
	if _, err := os.Stat(filepath.Join(root, "shipping/pypi/engines", first)); !os.IsNotExist(err) {
		t.Fatal("stale aggregate wheel retained")
	}
	for _, bad := range []string{"receipt.json", "foreign.whl"} {
		write("shipping/pypi/context/"+bad, []byte("not admitted"))
		if _, err := release.StageEngineUploads(root, m, "shipping"); err == nil {
			t.Fatal("unadmitted file included in engine upload")
		}
		if err := os.Remove(filepath.Join(root, "shipping/pypi/context", bad)); err != nil {
			t.Fatal(err)
		}
	}
	other := m.Snapshot.Components[0].Files[1].Name
	write("shipping/pypi/context/"+other, []byte("modified"))
	if _, err := release.StageEngineUploads(root, m, "shipping"); err == nil || !strings.Contains(err.Error(), "digest differs") {
		t.Fatal("changed wheel digest admitted")
	}
}
