package release

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    "github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

const engineVersionSchema = "ptsip-engine-version/v1"

func loadEngineVersion(repo *machine.Repository, catalog Catalog, id string) (string, error) {
    path := filepath.ToSlash(filepath.Join("src/ptsip/engines", id, catalog.EngineVersionFile))
    data, err := repo.Read(path)
    if err != nil {
        return "", err
    }
    if len(data) != 3 || machine.Text(data["schema_version"]) != engineVersionSchema ||
        machine.Text(data["engine"]) != id {
        return "", fmt.Errorf("invalid engine-local canonical version contract: %s", path)
    }
    version := machine.Text(data["version"])
    if version == "" {
        return "", fmt.Errorf("empty canonical engine version: %s", id)
    }
    return version, nil
}

func projectedCargoVersion(_ Catalog, _ string, version string) (string, error) {
    return CargoVersion(version)
}

func cargoPackageVersion(content string) (string, error) {
    section := strings.SplitN(content, "[dependencies]", 2)[0]
    re := regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"\s*$`)
    matches := re.FindStringSubmatch(section)
    if len(matches) != 2 {
        return "", fmt.Errorf("Cargo package version missing or malformed")
    }
    return matches[1], nil
}

func cargoLockVersion(content, id string) (string, error) {
    re := regexp.MustCompile(`(?m)\[\[package\]\]\r?\nname = "ptsip-engine-` + regexp.QuoteMeta(id) + `"\r?\nversion = "([^"]+)"`)
    matches := re.FindStringSubmatch(content)
    if len(matches) != 2 {
        return "", fmt.Errorf("Cargo.lock engine entry missing: %s", id)
    }
    return matches[1], nil
}

func verifyEngineVersionProjections(repo *machine.Repository, catalog Catalog, id, engineID, cargo, binding, version string) error {
    bindingData, err := repo.Read(binding)
    if err != nil {
        return err
    }
    entries := machine.List(bindingData["bindings"])
    if len(entries) == 0 {
        return fmt.Errorf("no execution bindings for engine: %s", id)
    }
    for _, raw := range entries {
        b := machine.Map(raw)
        if machine.Text(b["engine_id"]) != engineID || machine.Text(b["engine_version"]) != version {
            return fmt.Errorf("engine execution binding version differs from version.yaml: %s", id)
        }
    }
    expectedCargo, err := projectedCargoVersion(catalog, id, version)
    if err != nil {
        return err
    }
    cargoPath, err := repo.Path(cargo)
    if err != nil {
        return err
    }
    cargoBytes, err := os.ReadFile(cargoPath)
    if err != nil {
        return err
    }
    gotCargo, err := cargoPackageVersion(string(cargoBytes))
    if err != nil {
        return err
    }
    if gotCargo != expectedCargo {
        return fmt.Errorf("Cargo.toml version projection mismatch: %s (%s != %s)", id, gotCargo, expectedCargo)
    }
    lockPath, err := repo.Path(filepath.ToSlash(filepath.Join(filepath.Dir(cargo), "Cargo.lock")))
    if err != nil {
        return err
    }
    lockBytes, err := os.ReadFile(lockPath)
    if err != nil {
        return err
    }
    gotLock, err := cargoLockVersion(string(lockBytes), id)
    if err != nil {
        return err
    }
    if gotLock != expectedCargo {
        return fmt.Errorf("Cargo.lock version projection mismatch: %s (%s != %s)", id, gotLock, expectedCargo)
    }
    return nil
}
