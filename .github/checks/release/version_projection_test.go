package release

import (
    "strings"
    "testing"
)

func TestEngineVersionProjectionThreeSegment(t *testing.T) {
    catalog := Catalog{BootstrapEngineVersions: map[string]string{"context": "0.2.0", "storage": "0.3.0"}}
    for id, version := range map[string]string{"context": "0.2.0", "storage": "0.3.0", "validation": "1.2.3"} {
        got, err := projectedCargoVersion(catalog, id, version)
        if err != nil || got != version {
            t.Fatalf("unexpected engine Cargo projection for %s: %q %v", id, got, err)
        }
    }
    for _, invalid := range []string{"0.2", "0.2a1", "0.2.0-alpha.1", "0.3.0rc1"} {
        if _, err := projectedCargoVersion(catalog, "storage", invalid); err == nil {
            t.Fatalf("invalid three-segment engine version admitted: %s", invalid)
        }
    }
}

func TestEngineSyncOnlyChangesCargoPackageIdentity(t *testing.T) {
    manifest := "[package]\nname = \"ptsip-engine-context\"\nversion = \"0.2.0-alpha.1\"\n[dependencies]\nserde = \"1.0\"\n"
    updated, err := rewriteCargoPackage(manifest, "0.2.0")
    if err != nil || !strings.Contains(updated, "version = \"0.2.0\"") || !strings.Contains(updated, "serde = \"1.0\"") {
        t.Fatalf("Cargo package-only replacement failed: %q %v", updated, err)
    }
    actual, err := cargoPackageVersion(updated)
    if err != nil || actual != "0.2.0" {
        t.Fatalf("Cargo version mismatch: %q %v", actual, err)
    }
    lock := "[[package]]\nname = \"ptsip-engine-context\"\nversion = \"0.2.0-alpha.1\"\n"
    revised, err := rewriteCargoLock(lock, "context", "0.2.0")
    if err != nil { t.Fatal(err) }
    actual, err = cargoLockVersion(revised, "context")
    if err != nil || actual != "0.2.0" {
        t.Fatalf("Cargo.lock projection mismatch: %q %v", actual, err)
    }
}
