package main

import (
    "errors"
    "fmt"
    "os"
    "path"
    "path/filepath"
    "strings"
)

// Go verification does not install Argos or mutate translated documents.
// Strict verification additionally requires a source hash for every
// non-title block, in the same order as the canonical README.
func verifyTranslations(root string, r translationRegistry, strict bool) error {
    source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(r.Canonical.Path)))
    if err != nil { return err }
    for _, target := range r.Translations {
        filename, err := safeTargetFile(root, target.Path)
        if err != nil { return err }
        body, err := os.ReadFile(filename)
        if errors.Is(err, os.ErrNotExist) { return fmt.Errorf("missing registered translation: %s", target.Path) }
        if err != nil { return err }
        if err := validate(string(source), string(body), r, target); err != nil {
            return fmt.Errorf("%s: %w", target.Path, err)
        }
        if err := verifySourceMarkers(string(source), string(body), strict); err != nil {
            return fmt.Errorf("%s: %w", target.Path, err)
        }
    }
    mode := "structural"
    if strict { mode = "source-hash-strict" }
    fmt.Printf("README verification PASS (%s): %d registered translation(s)\n", mode, len(r.Translations))
    return nil
}

// Legacy translations lacking generation markers can pass structure checking
// only; this does NOT establish that their prose matches the current source.
func verifySourceMarkers(source, body string, strict bool) error {
    blocks := blockSplit(stripNav(source))
    if len(blocks) == 0 { return errors.New("canonical README has no blocks") }
    hashes := []string{}
    for _, b := range blocks[1:] { hashes = append(hashes, hashBlock(b)) }
    markers := []string{}
    for _, b := range blockSplit(body) {
        if match := shaMarker.FindStringSubmatch(b); match != nil {
            markers = append(markers, match[1])
        }
    }
    if len(markers) == 0 {
        if strict { return errors.New("source freshness UNRESOLVED: translation has no source-hash markers") }
        fmt.Fprintln(os.Stderr, "README verification: legacy translation freshness UNRESOLVED (no source-hash markers)")
        return nil
    }
    if len(markers) != len(hashes) {
        return fmt.Errorf("source-hash marker count mismatch: %d current blocks, %d translation markers", len(hashes), len(markers))
    }
    for i, marker := range markers {
        if marker != hashes[i] {
            return fmt.Errorf("outdated or reordered translation at block %d", i+1)
        }
    }
    return nil
}

// Registry paths are relative, but path validation must also reject symlink
// ancestors and targets before reading or changing registered Markdown.
func safeTargetFile(root, target string) (string, error) {
    if err := safePath(target); err != nil { return "", err }
    rootAbs, err := filepath.Abs(root)
    if err != nil { return "", err }
    filename := filepath.Join(rootAbs, filepath.FromSlash(target))
    rel, err := filepath.Rel(rootAbs, filename)
    if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
        return "", fmt.Errorf("translation target escapes repository: %s", target)
    }
    for dir := filename; dir != rootAbs; dir = filepath.Dir(dir) {
        if dir == filepath.Dir(dir) { return "", fmt.Errorf("unsafe path ancestor: %s", target) }
        info, err := os.Lstat(dir)
        if err != nil {
            if errors.Is(err, os.ErrNotExist) { continue }
            return "", err
        }
        if info.Mode()&os.ModeSymlink != 0 { return "", fmt.Errorf("symlink target forbidden: %s", target) }
    }
    if path.Ext(target) != ".md" { return "", fmt.Errorf("unexpected registered target type: %s", target) }
    return filename, nil
}
