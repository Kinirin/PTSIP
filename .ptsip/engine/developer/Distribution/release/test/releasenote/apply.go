package main

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "sort"
)

// The change executor prepares every language before mutating any output.
// Its authority is limited to the translation targets in the registry.
type preparedTranslation struct {
    filename string
    old []byte
    existed bool
    newText string
}

func applyTranslations(root string, r translationRegistry) error {
    source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(r.Canonical.Path)))
    if err != nil { return err }
    changes := []preparedTranslation{}
    for _, target := range r.Translations {
        filename, err := safeTargetFile(root, target.Path)
        if err != nil { return err }
        existing, err := os.ReadFile(filename)
        existed := err == nil
        if err != nil && !errors.Is(err, os.ErrNotExist) { return err }
        rendered, err := render(string(source), string(existing), r, target)
        if err != nil { return fmt.Errorf("%s: %w", target.Path, err) }
        if err := verifySourceMarkers(string(source), rendered, true); err != nil {
            return fmt.Errorf("generated %s: %w", target.Path, err)
        }
        if rendered != string(existing) {
            changes = append(changes, preparedTranslation{filename:filename, old:existing, existed:existed, newText:rendered})
        }
    }
    sort.Slice(changes, func(i,j int) bool { return changes[i].filename < changes[j].filename })
    staged := make(map[string]string, len(changes))
    defer func(){ for _, filename := range staged { _ = os.Remove(filename) } }()
    // Stage every output as a temporary file; never publish partial translated
    // content while a different registered language still needs translation.
    for _, change := range changes {
        if err := os.MkdirAll(filepath.Dir(change.filename), 0755); err != nil { return err }
        tmp, err := os.CreateTemp(filepath.Dir(change.filename), ".readme-apply-*")
        if err != nil { return err }
        staged[change.filename] = tmp.Name()
        if _, err = tmp.WriteString(change.newText); err != nil { _ = tmp.Close(); return err }
        if err = tmp.Chmod(0644); err != nil { _ = tmp.Close(); return err }
        if err = tmp.Close(); err != nil { return err }
    }
    applied := []preparedTranslation{}
    for _, change := range changes {
        if err := os.Rename(staged[change.filename], change.filename); err != nil {
            // A disk error is never PASS. Roll back previously written targets;
            // if recovery also fails the workflow still terminates in FAIL.
            failures := []error{}
            for i := len(applied)-1; i >= 0; i-- {
                previous := applied[i]
                if previous.existed {
                    if rollbackErr := os.WriteFile(previous.filename, previous.old, 0644); rollbackErr != nil {
                        failures = append(failures, rollbackErr)
                    }
                } else if rollbackErr := os.Remove(previous.filename); rollbackErr != nil {
                    failures = append(failures, rollbackErr)
                }
            }
            return errors.Join(fmt.Errorf("README apply failed at %s: %w", change.filename, err), errors.Join(failures...))
        }
        delete(staged, change.filename)
        applied = append(applied, change)
    }
    if err := verifyTranslations(root, r, true); err != nil {
        return fmt.Errorf("post-apply strict verification failed: %w", err)
    }
    fmt.Printf("README apply PASS: %d registered, %d updated\n", len(r.Translations), len(changes))
    return nil
}
