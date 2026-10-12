package release

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"

    "github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

func readVersionJSON(repo *machine.Repository, path string) (map[string]any, error) {
    full, err := repo.Path(path)
    if err != nil { return nil, err }
    data, err := os.ReadFile(full)
    if err != nil { return nil, err }
    var document map[string]any
    if err := json.Unmarshal(data, &document); err != nil { return nil, err }
    if document == nil { return nil, fmt.Errorf("empty version projection JSON: %s", path) }
    return document, nil
}

func stageVersionJSON(changes map[string][]byte, path string, document map[string]any) error {
    data, err := json.MarshalIndent(document, "", "  ")
    if err != nil { return err }
    changes[path] = append(data, '\n')
    return nil
}

func rewriteCargoPackage(content, target string) (string, error) {
    header := strings.SplitN(content, "[dependencies]", 2)[0]
    re := regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"\s*$`)
    match := re.FindStringSubmatch(header)
    if len(match) != 2 { return "", fmt.Errorf("Cargo.toml package version not found") }
    before := `version = "` + match[1] + `"`
    if strings.Count(header, before) != 1 { return "", fmt.Errorf("ambiguous Cargo.toml package version") }
    return strings.Replace(content, before, `version = "` + target + `"`, 1), nil
}

func rewriteCargoLock(content, id, target string) (string, error) {
    previous, err := cargoLockVersion(content, id)
    if err != nil { return "", err }
    prior := "[[package]]\nname = \"ptsip-engine-" + id + "\"\nversion = \"" + previous + "\""
    if strings.Count(content, prior) != 1 { return "", fmt.Errorf("ambiguous Cargo.lock entry: %s", id) }
    next := "[[package]]\nname = \"ptsip-engine-" + id + "\"\nversion = \"" + target + "\""
    return strings.Replace(content, prior, next, 1), nil
}

func applyVersionProjections(repo *machine.Repository, changes map[string][]byte) (int, error) {
    names := make([]string, 0, len(changes))
    for name := range changes { names = append(names, name) }
    sort.Strings(names)
    type stagedFile struct { path, temp string; old []byte; mode os.FileMode }
    staged := []stagedFile{}
    cleanup := func() { for _, s := range staged { if s.temp != "" { _ = os.Remove(s.temp) } } }
    defer cleanup()
    for _, name := range names {
        path, err := repo.Path(name)
        if err != nil { return 0, err }
        old, err := os.ReadFile(path)
        if err != nil { return 0, err }
        if bytes.Equal(old, changes[name]) { continue }
        stat, err := os.Stat(path)
        if err != nil { return 0, err }
        temp, err := os.CreateTemp(filepath.Dir(path), ".ptsip-version-sync-*")
        if err != nil { return 0, err }
        staged = append(staged, stagedFile{path: path, temp: temp.Name(), old: old, mode: stat.Mode().Perm()})
        if _, err = temp.Write(changes[name]); err != nil { _ = temp.Close(); return 0, err }
        if err = temp.Chmod(stat.Mode().Perm()); err != nil { _ = temp.Close(); return 0, err }
        if err = temp.Close(); err != nil { return 0, err }
    }
    applied := 0
    for _, s := range staged {
        if err := os.Rename(s.temp, s.path); err != nil {
            for i := applied - 1; i >= 0; i-- {
                prior := staged[i]
                if revertErr := os.WriteFile(prior.path, prior.old, prior.mode); revertErr != nil {
                    return applied, fmt.Errorf("version projection rename failed: %v; rollback failed: %v", err, revertErr)
                }
            }
            return 0, err
        }
        applied++
    }
    return applied, nil
}

// SyncEngineVersions updates only derived records; version.yaml is never modified.
func SyncEngineVersions(repo *machine.Repository, targets string) (machine.Object, error) {
    catalog, err := loadCatalog(repo)
    if err != nil { return nil, err }
    allowed := map[string]bool{}
    for _, id := range catalog.EngineIDs { allowed[id] = true }
    selected, err := selectionSet(targets, allowed)
    if err != nil { return nil, err }
    index, err := readVersionJSON(repo, catalog.EngineIndex)
    if err != nil { return nil, err }
    entries, ok := index["official_engine_entries"].([]any)
    if !ok || len(entries) != len(catalog.EngineIDs) { return nil, fmt.Errorf("invalid official engine registry") }
    changes := map[string][]byte{}
    updated := []string{}
    for _, raw := range entries {
        engine, ok := raw.(map[string]any)
        if !ok { return nil, fmt.Errorf("invalid official engine entry") }
        id := strings.ToLower(fmt.Sprint(engine["kind"]))
        if !allowed[id] { return nil, fmt.Errorf("unknown official engine: %s", id) }
        if !selected[id] { continue }
        version, err := loadEngineVersion(repo, catalog, id)
        if err != nil { return nil, err }
        if !regexp.MustCompile(catalog.VersionRules["engine"]).MatchString(version) &&
            version != catalog.BootstrapEngineVersions[id] {
            return nil, fmt.Errorf("invalid canonical public engine version: %s %s", id, version)
        }
        cargoVersion, err := projectedCargoVersion(catalog, id, version)
        if err != nil { return nil, err }
        engine["version"] = version
        manifestPath := fmt.Sprint(engine["manifest_ref"])
        manifest, err := readVersionJSON(repo, manifestPath)
        if err != nil { return nil, err }
        if manifest["engine_id"] != engine["engine_id"] { return nil, fmt.Errorf("engine identity mismatch: %s", id) }
        manifest["implementation_version"] = version
        if err := stageVersionJSON(changes, manifestPath, manifest); err != nil { return nil, err }
        bindingPath := fmt.Sprint(engine["execution_binding_ref"])
        binding, err := readVersionJSON(repo, bindingPath)
        if err != nil { return nil, err }
        bound, ok := binding["bindings"].([]any)
        if !ok || len(bound) == 0 { return nil, fmt.Errorf("missing execution binding: %s", id) }
        for _, b := range bound {
            row, ok := b.(map[string]any)
            if !ok || row["engine_id"] != engine["engine_id"] { return nil, fmt.Errorf("execution binding identity mismatch: %s", id) }
            row["engine_version"] = version
        }
        if err := stageVersionJSON(changes, bindingPath, binding); err != nil { return nil, err }
        cargo := fmt.Sprint(engine["cargo_manifest_ref"])
        cargoPath, err := repo.Path(cargo)
        if err != nil { return nil, err }
        cargoContent, err := os.ReadFile(cargoPath)
        if err != nil { return nil, err }
        cargoNext, err := rewriteCargoPackage(string(cargoContent), cargoVersion)
        if err != nil { return nil, err }
        changes[cargo] = []byte(cargoNext)
        lock := filepath.ToSlash(filepath.Join(filepath.Dir(cargo), "Cargo.lock"))
        lockPath, err := repo.Path(lock)
        if err != nil { return nil, err }
        lockContent, err := os.ReadFile(lockPath)
        if err != nil { return nil, err }
        lockNext, err := rewriteCargoLock(string(lockContent), id, cargoVersion)
        if err != nil { return nil, err }
        changes[lock] = []byte(lockNext)
        updated = append(updated, id)
    }
    if len(updated) != len(selected) { return nil, fmt.Errorf("registered engine selection incomplete") }
    if err := stageVersionJSON(changes, catalog.EngineIndex, index); err != nil { return nil, err }
    written, err := applyVersionProjections(repo, changes)
    if err != nil { return nil, err }
    sort.Strings(updated)
    return machine.Object{"status": "PASS", "operation": "SYNC_DERIVED_ENGINE_VERSIONS", "engines": updated, "files_changed": written}, nil
}
