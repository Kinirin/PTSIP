package release

import "testing"

func TestEngineVersionProjectionAlpha(t *testing.T) {
    got, err := CargoVersion("0.2a1")
    if err != nil || got != "0.2.0-alpha.1" {
        t.Fatalf("unexpected Cargo version: %s, %v", got, err)
    }
    catalog := Catalog{BootstrapEngineVersions: map[string]string{"storage": "0.3.0"}}
    historical, err := projectedCargoVersion(catalog, "storage", "0.3.0")
    if err != nil || historical != "0.3.0" {
        t.Fatalf("bootstrap version was changed: %s, %v", historical, err)
    }
}
