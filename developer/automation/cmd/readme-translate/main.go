// Command readme-translate synchronizes only the languages registered in
// docs/translated/index.yaml. Argos is an external translation engine;
// policy, Markdown handling, file selection, and verification remain in Go.
package main

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "flag"
    "fmt"
    "os"
    "os/exec"
    "path"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
    "time"

    "go.yaml.in/yaml/v3"
)

type languageEntry struct {
    Language string `yaml:"language"`
    Path string `yaml:"path"`
}
type translationRegistry struct {
    SchemaVersion string `yaml:"schema_version"`
    Canonical languageEntry `yaml:"canonical"`
    Translations []languageEntry `yaml:"translations"`
}
var languageID = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]+)*$`)
var shaMarker = regexp.MustCompile(`^<!-- ptsip-readme-i18n: ([0-9a-f]{64}) -->$`)
var fenceStart = regexp.MustCompile("^[ \t]*("+ "\x60\x60\x60"+ "+|~~~+)")
var heading = regexp.MustCompile(`^(#{1,6})[ \t]+(.*)$`)
var protected = regexp.MustCompile("(\x60+[^\x60\\n]*\x60+|\\[[^\\]\\n]*\\]\\([^)]+\\)|https?://[^\\s<>)]+|</?[^>\\n]+>|\\*\\*|__|~~|\\*|\\b(?:MUST NOT|SHOULD NOT|MUST|SHOULD|MAY|PTSIP|VPMS)\\b)")
var inlineCode = regexp.MustCompile("\x60+[^\x60\\n]*\x60+")
var mdLink = regexp.MustCompile(`(!?)\[([^\]\n]*)\]\(([^)]+)\)`)
var mdLinkTarget = regexp.MustCompile(`\]\(([^)]+)\)`)
var navLine = regexp.MustCompile(`(?s)^\s*<p\s+align="right">.*?</p>\s*`)
var checksum = regexp.MustCompile(`^[0-9a-f]{64}$`)

func safePath(s string) error {
    if s == "" || strings.Contains(s, "\\") || strings.Contains(s, ":") ||
        strings.HasPrefix(s, "/") || path.Clean(s) != s {
        return fmt.Errorf("unsafe registry path %q", s)
    }
    for _, p := range strings.Split(s, "/") {
        if p == "." || p == ".." || p == "" {
            return fmt.Errorf("unsafe registry path %q", s)
        }
    }
    return nil
}
func loadRegistry(root, filename string) (translationRegistry, error) {
    var r translationRegistry
    if err := safePath(filename); err != nil { return r, err }
    b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(filename)))
    if err != nil { return r, err }
    decoder := yaml.NewDecoder(bytes.NewReader(b))
    decoder.KnownFields(true)
    if err = decoder.Decode(&r); err != nil { return r, err }
    if r.SchemaVersion != "ptsip-readme-translation-registry/v1" {
        return r, errors.New("unknown README translation registry version")
    }
    if len(r.Translations) == 0 {
        return r, errors.New("translation registry must declare at least one target")
    }
    entries := append([]languageEntry{r.Canonical}, r.Translations...)
    seenLanguage, seenPath := map[string]bool{}, map[string]bool{}
    for i, e := range entries {
        if !languageID.MatchString(e.Language) { return r, fmt.Errorf("invalid language code %q", e.Language) }
        if err := safePath(e.Path); err != nil { return r, err }
        if i > 0 && !strings.HasPrefix(e.Path, "docs/translated/") {
            return r, fmt.Errorf("translation target outside docs/translated/: %q", e.Path)
        }
        if seenLanguage[e.Language] || seenPath[e.Path] {
            return r, fmt.Errorf("duplicate language/path %q / %q", e.Language, e.Path)
        }
        seenLanguage[e.Language], seenPath[e.Path] = true, true
    }
    return r, nil
}
func stripNav(s string) string {
    return navLine.ReplaceAllString(strings.TrimPrefix(s, "\ufeff"), "")
}
func blockSplit(s string) []string {
    s = strings.ReplaceAll(s, "\r\n", "\n")
    var out []string
    var lines []string
    inFence := byte(0)
    for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
        trimmed := strings.TrimLeft(line, " \t")
        if strings.HasPrefix(trimmed, "\x60\x60\x60") || strings.HasPrefix(trimmed, "~~~") {
            symbol := trimmed[0]
            if inFence == 0 { inFence = symbol } else if inFence == symbol { inFence = 0 }
        }
        if strings.TrimSpace(line) == "" && inFence == 0 {
            if len(lines) > 0 { out = append(out, strings.Join(lines, "\n")); lines = nil }
            continue
        }
        lines = append(lines, line)
    }
    if len(lines) > 0 { out = append(out, strings.Join(lines, "\n")) }
    return out
}
func hashBlock(s string) string {
    v := sha256.Sum256([]byte(s))
    return hex.EncodeToString(v[:])
}
func oldMemory(body string) map[string]string {
    blocks := blockSplit(body)
    memory := make(map[string]string)
    for i := 0; i+1 < len(blocks); i++ {
        if m := shaMarker.FindStringSubmatch(blocks[i]); m != nil && !shaMarker.MatchString(blocks[i+1]) {
            memory[m[1]] = blocks[i+1]
            i++
        }
    }
    return memory
}
func rebaseLinks(text, source, target string) string {
    from := path.Dir(source)
    to := path.Dir(target)
    return mdLinkTarget.ReplaceAllStringFunc(text, func(full string) string {
        value := full[2:len(full)-1]
        if value == "" || strings.HasPrefix(value, "#") || strings.HasPrefix(value, "/") ||
            strings.HasPrefix(value, "../") || strings.Contains(value, "://") || strings.HasPrefix(value, "mailto:") {
            return full
        }
        ref := value
        suffix := ""
        if i := strings.IndexAny(ref, "?#"); i >= 0 { suffix, ref = ref[i:], ref[:i] }
        if ref == "" { return full }
        repoPath := path.Clean(path.Join(from, ref))
        // path.Rel is unavailable; filepath.Rel uses the host separator.
        rel, err := filepath.Rel(filepath.FromSlash(to), filepath.FromSlash(repoPath))
        if err != nil { return full }
        rel = filepath.ToSlash(rel)
        if strings.HasSuffix(ref, "/") { rel += "/" }
        return "](" + rel + suffix + ")"
    })
}
func normalizeLink(ref, owner string) string {
    if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "/") ||
        strings.Contains(ref, "://") || strings.HasPrefix(ref, "mailto:") {
        return ref
    }
    suffix := ""
    if i := strings.IndexAny(ref, "?#"); i >= 0 {
        suffix, ref = ref[i:], ref[:i]
    }
    if ref == "" { return suffix }
    return path.Clean(path.Join(path.Dir(owner), ref)) + suffix
}
func codeTokens(s string) []string { return inlineCode.FindAllString(s, -1) }
func fenced(s string) []string {
    out := []string{}
    for _, b := range blockSplit(s) {
        if strings.HasPrefix(strings.TrimLeft(b, " \t"), "\x60\x60\x60") || strings.HasPrefix(strings.TrimLeft(b, " \t"), "~~~") {
            out = append(out, b)
        }
    }
    return out
}
func headingLevels(s string) []int {
    out := []int{}
    for _, line := range strings.Split(s, "\n") {
        if m := heading.FindStringSubmatch(line); m != nil { out = append(out, len(m[1])) }
    }
    return out
}
func equalStrings(a, b []string) bool {
    if len(a) != len(b) { return false }
    for i := range a { if a[i] != b[i] { return false } }
    return true
}
func equalInts(a, b []int) bool {
    if len(a) != len(b) { return false }
    for i := range a { if a[i] != b[i] { return false } }
    return true
}
func validate(source, localized string, r translationRegistry, target languageEntry) error {
    src := stripNav(source)
    if !equalInts(headingLevels(src), headingLevels(localized)) { return errors.New("heading structure mismatch") }
    if !equalStrings(fenced(src), fenced(localized)) { return errors.New("fenced code blocks mismatch") }
    counts := map[string]int{}
    for _, v := range codeTokens(localized) { counts[v]++ }
    for _, v := range codeTokens(src) {
        if counts[v] == 0 { return fmt.Errorf("missing inline code token %q", v) }
        counts[v]--
    }
    targetLinks := mdLinkTarget.FindAllStringSubmatch(localized, -1)
    for _, m := range mdLinkTarget.FindAllStringSubmatch(src, -1) {
        expected := normalizeLink(m[1], r.Canonical.Path)
        found := false
        for _, candidate := range targetLinks {
            if normalizeLink(candidate[1], target.Path) == expected {
                found = true
                break
            }
        }
        if !found { return fmt.Errorf("missing canonical link target %q", expected) }
    }
    expectedBack, err := filepath.Rel(filepath.Dir(filepath.FromSlash(target.Path)), filepath.FromSlash(r.Canonical.Path))
    if err != nil { return err }
    if !strings.Contains(localized, "href=\""+filepath.ToSlash(expectedBack)+"\"") {
        return errors.New("missing canonical README backlink")
    }
    for _, keyword := range []string{"MUST NOT", "SHOULD NOT", "MUST", "SHOULD", "MAY"} {
        if strings.Contains(src, keyword) && !strings.Contains(localized, keyword) {
            return fmt.Errorf("normative keyword disappeared: %s", keyword)
        }
    }
    return nil
}
type argos struct {
    from, to string
    cache map[string]string
}
func (a *argos) translate(s string) (string, error) {
    if strings.TrimSpace(s) == "" || !regexp.MustCompile(`[A-Za-z]`).MatchString(s) { return s, nil }
    if translated, ok := a.cache[s]; ok { return translated, nil }
    leading := s[:len(s)-len(strings.TrimLeft(s, " \t"))]
    trailing := s[len(strings.TrimRight(s, " \t")):]
    core := strings.TrimSpace(s)
    if core == "" { return s, nil }
    ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "argos-translate", "--from-lang", a.from, "--to-lang", a.to)
    cmd.Stdin = strings.NewReader(core)
    var errBuf bytes.Buffer
    cmd.Stderr = &errBuf
    raw, err := cmd.Output()
    if err != nil { return "", fmt.Errorf("Argos %s -> %s: %w: %s", a.from, a.to, err, errBuf.String()) }
    value := strings.TrimSpace(string(raw))
    if value == "" { return "", errors.New("Argos returned empty translation") }
    value = leading + value + trailing
    a.cache[s] = value
    return value, nil
}
func translateRich(s string, a *argos) (string, error) {
    positions := protected.FindAllStringIndex(s, -1)
    var b strings.Builder
    start := 0
    for _, p := range positions {
        if p[0] < start { continue }
        plain, err := a.translate(s[start:p[0]])
        if err != nil { return "", err }
        b.WriteString(plain)
        token := s[p[0]:p[1]]
        if m := mdLink.FindStringSubmatch(token); m != nil {
            label, err := translateRich(m[2], a)
            if err != nil { return "", err }
            token = m[1]+"["+label+"]("+m[3]+")"
        }
        b.WriteString(token)
        start = p[1]
    }
    tail, err := a.translate(s[start:])
    if err != nil { return "", err }
    b.WriteString(tail)
    return b.String(), nil
}
func translateLine(line string, a *argos) (string, error) {
    if line == "" { return line, nil }
    if m := heading.FindStringSubmatch(line); m != nil {
        if len(m[1]) == 1 { return line, nil }
        v, err := translateRich(m[2], a)
        return m[1]+" "+v, err
    }
    left := ""
    if strings.HasPrefix(line, "> ") { left, line = "> ", strings.TrimPrefix(line, "> ") }
    if strings.HasPrefix(line, "- ") { left, line = left+"- ", strings.TrimPrefix(line, "- ") }
    if strings.HasPrefix(line, "|") && strings.HasSuffix(strings.TrimSpace(line), "|") {
        cells := strings.Split(line, "|")
        for i, c := range cells {
            if strings.Trim(c, " :-") == "" { continue }
            prefix := c[:len(c)-len(strings.TrimLeft(c, " \t"))]
            suffix := c[len(strings.TrimRight(c, " \t")):]
            v, err := translateRich(strings.TrimSpace(c), a)
            if err != nil { return "", err }
            cells[i] = prefix+v+suffix
        }
        return left+strings.Join(cells, "|"), nil
    }
    v, err := translateRich(line, a)
    return left+v, err
}
func translateBlock(block string, a *argos) (string, error) {
    if strings.HasPrefix(block, "# ") { return block, nil }
    trim := strings.TrimLeft(block, " \t")
    if strings.HasPrefix(trim, "\x60\x60\x60") || strings.HasPrefix(trim, "~~~") { return block, nil }
    lines := strings.Split(block, "\n")
    for i := range lines {
        v, err := translateLine(lines[i], a)
        if err != nil { return "", err }
        lines[i] = v
    }
    return strings.Join(lines, "\n"), nil
}
func render(source, old string, r translationRegistry, target languageEntry) (string, error) {
    src := stripNav(source)
    blocks := blockSplit(src)
    if len(blocks) == 0 || !strings.HasPrefix(blocks[0], "# ") { return "", errors.New("README lacks first-level title") }
    memory := oldMemory(old)
    a := &argos{from:r.Canonical.Language, to:target.Language, cache:map[string]string{}}
    parts := make([]string, 0, len(blocks)*2)
    parts = append(parts, blocks[0])
    for _, block := range blocks[1:] {
        hash := hashBlock(block)
        localized, ok := memory[hash]
        if !ok {
            var err error
            localized, err = translateBlock(block, a)
            if err != nil { return "", fmt.Errorf("block %s: %w", hash[:12], err) }
            localized = rebaseLinks(localized, r.Canonical.Path, target.Path)
        }
        if !checksum.MatchString(hash) { return "", errors.New("invalid source checksum") }
        parts = append(parts, "<!-- ptsip-readme-i18n: "+hash+" -->", localized)
    }
    backlink, err := filepath.Rel(filepath.Dir(filepath.FromSlash(target.Path)), filepath.FromSlash(r.Canonical.Path))
    if err != nil { return "", err }
    backlink = filepath.ToSlash(backlink)
    nav := fmt.Sprintf("<p align=\"right\">\n  <a href=\"%s\">%s</a> | %s\n</p>", backlink, r.Canonical.Language, target.Language)
    output := "<!-- AUTO-GENERATED: registry-driven Go projection; edit canonical README.md instead. -->\n"+
        nav+"\n\n"+parts[0]+"\n\n"+strings.Join(parts[1:], "\n\n")+"\n"
    if err := validate(source, output, r, target); err != nil { return "", err }
    return output, nil
}
func runExternal(binary string, args ...string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
    defer cancel()
    cmd := exec.CommandContext(ctx, binary, args...)
    cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
    if err := cmd.Run(); err != nil { return fmt.Errorf("%s %v: %w", binary, args, err) }
    return nil
}
func run(root, action, registryFile string) error {
    r, err := loadRegistry(root, registryFile)
    if err != nil { return err }
    switch action {
    case "list-targets":
        for _, e := range r.Translations { fmt.Println(e.Path) }
        return nil
    case "models":
        if err := runExternal("argospm", "update"); err != nil { return err }
        for _, e := range r.Translations {
            pkg := "translate-"+r.Canonical.Language+"_"+e.Language
            if err := runExternal("argospm", "install", pkg); err != nil { return err }
        }
        return nil
    case "validate", "sync":
    default:
        return fmt.Errorf("unsupported action %q", action)
    }
    source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(r.Canonical.Path)))
    if err != nil { return err }
    pending := make(map[string]string)
    for _, target := range r.Translations {
        dst := filepath.Join(root, filepath.FromSlash(target.Path))
        current, readErr := os.ReadFile(dst)
        if readErr != nil && !os.IsNotExist(readErr) { return readErr }
        if action == "validate" {
            if os.IsNotExist(readErr) { return fmt.Errorf("missing registered target: %s", target.Path) }
            if err := validate(string(source), string(current), r, target); err != nil {
                return fmt.Errorf("%s: %w", target.Path, err)
            }
            continue
        }
        rendered, err := render(string(source), string(current), r, target)
        if err != nil { return fmt.Errorf("%s: %w", target.Path, err) }
        if rendered != string(current) { pending[dst] = rendered }
    }
    if action == "validate" {
        fmt.Printf("Validated %d registered translation(s)\n", len(r.Translations))
        return nil
    }
    paths := make([]string, 0, len(pending))
    for p := range pending { paths = append(paths, p) }
    sort.Strings(paths)
    for _, filename := range paths {
        if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil { return err }
        tmp, err := os.CreateTemp(filepath.Dir(filename), ".readme-translate-*")
        if err != nil { return err }
        if _, err = tmp.WriteString(pending[filename]); err == nil { err = tmp.Close() } else { _ = tmp.Close() }
        if err != nil { _ = os.Remove(tmp.Name()); return err }
        if err = os.Rename(tmp.Name(), filename); err != nil { _ = os.Remove(tmp.Name()); return err }
        fmt.Printf("Synchronized %s\n", filepath.ToSlash(filename))
    }
    fmt.Printf("README translations: %d registered, %d updated\n", len(r.Translations), len(paths))
    return nil
}
func main() {
    root := flag.String("root", "../..", "repository root relative to Go module")
    registry := flag.String("registry", "docs/translated/index.yaml", "registered translation source")
    action := flag.String("action", "sync", "list-targets | models | validate | sync")
    flag.Parse()
    if err := run(*root, *action, *registry); err != nil {
        fmt.Fprintln(os.Stderr, "readme-translate:", err)
        os.Exit(2)
    }
}
