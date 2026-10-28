package releasenote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	version "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/version"
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

const NotePrefix = "<!-- ptsip-release-note:v1 "

type Link struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}
type Note struct {
	Schema        string                        `json:"schema_version"`
	Language      string                        `json:"language"`
	Kind          string                        `json:"kind"`
	Change        version.DeclaredVersionChange `json:"change"`
	Base          string                        `json:"base_sha"`
	Head          string                        `json:"head_sha"`
	Digest        string                        `json:"version_sha256"`
	Summary       string                        `json:"summary"`
	Details       string                        `json:"technical_details"`
	Documentation []Link                        `json:"documentation"`
}

func kindOf(component string) string {
	if strings.HasPrefix(component, "engine.") {
		return "engine"
	}
	return component
}

func (c *Catalog) ArchivePath(note Note) (string, error) {
	if note.Kind != kindOf(note.Change.Component) {
		return "", fmt.Errorf("NOTE_COMPONENT_KIND_MISMATCH")
	}
	route, present := c.Workflow.Archive.Types[note.Kind]
	if !present {
		return "", fmt.Errorf("UNKNOWN_NOTE_KIND")
	}
	filenameVersion := note.Change.Version
	if note.Kind == "tool" {
		parts := regexp.MustCompile(`^(T\.[0-9]+\.[0-9]+\.[0-9]+)(?:\.((?:a|b|rc)[1-9][0-9]*))?$`).FindStringSubmatch(note.Change.Version)
		if parts == nil {
			return "", fmt.Errorf("INVALID_TOOL_NOTE_VERSION")
		}
		filenameVersion = parts[1]
		if parts[2] != "" {
			filenameVersion += "-" + parts[2]
		}
	}
	path := strings.NewReplacer("{engine}", strings.TrimPrefix(note.Change.Component, "engine."), "{version}", note.Change.Version, "{filename_version}", filenameVersion).Replace(route.Path)
	if strings.ContainsAny(path, "{}\\:\r\n") || filepath.ToSlash(filepath.Clean(path)) != path || !strings.HasPrefix(path, c.Workflow.Archive.Root+"/") {
		return "", fmt.Errorf("INVALID_NOTE_PATH")
	}
	return path, nil
}

func (c *Catalog) CheckNote(note Note, validate authoring.Validator) error {
	id := "urn:ptsip:docs:format:" + note.Kind + "-readme:v1"
	if note.Kind == "package" {
		id = BasicID + "#/$defs/ReleaseNote"
	}
	if _, err := c.ArchivePath(note); err != nil {
		return err
	}
	if note.Change.Previous == note.Change.Version {
		return fmt.Errorf("NOTE_REQUIRES_ACTUAL_VERSION_CHANGE")
	}
	if err := c.Validate(validate, id, note); err != nil {
		return err
	}
	for _, body := range []string{note.Summary, note.Details} {
		for _, level := range HeadingLines(body) {
			if level < 3 {
				return fmt.Errorf("NOTE_BODY_CANNOT_REPLACE_TEMPLATE_SECTIONS")
			}
		}
	}
	for _, link := range note.Documentation {
		if strings.HasPrefix(link.Path, "docs/translated/") {
			return fmt.Errorf("ARCHIVE_DOCUMENTATION_MUST_REFERENCE_ORIGINAL")
		}
		if _, err := authoring.LocalFile(c.Root, link.Path); err != nil {
			return err
		}
		if strings.ContainsAny(link.Title, "\r\n[]") || strings.ContainsAny(link.Path, "()") {
			return fmt.Errorf("INVALID_DOCUMENTATION_LINK")
		}
	}
	return nil
}

func (c *Catalog) RenderNote(note Note) ([]byte, error) {
	path, err := c.ArchivePath(note)
	if err != nil {
		return nil, err
	}
	metadata, err := json.Marshal(note)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString(NotePrefix + string(metadata) + " -->\n\n")
	fmt.Fprintf(&out, "# PTSIP %s %s\n\nOriginal language: English. This record describes declared version changes.\n\n", note.Kind, note.Change.Version)
	fmt.Fprintf(&out, "## Summary\n\n%s\n\n## Version changes\n\n| Component | Previous | Current |\n| --- | --- | --- |\n| `%s` | `%s` | `%s` |\n\n", note.Summary, note.Change.Component, note.Change.Previous, note.Change.Version)
	fmt.Fprintf(&out, "Base source: `%s`  \nCurrent source: `%s`  \nVERSION SHA256: `%s`\n\n", note.Base, note.Head, note.Digest)
	fmt.Fprintf(&out, "## Technical details\n\n%s\n\n## Documentation\n\n", note.Details)
	for _, link := range note.Documentation {
		relative, err := filepath.Rel(filepath.Dir(filepath.FromSlash(path)), filepath.FromSlash(link.Path))
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, "- [%s](%s)\n", link.Title, filepath.ToSlash(relative))
	}
	return []byte(out.String()), nil
}

func (c *Catalog) CheckArchive(validate authoring.Validator) (int, error) {
	count := 0
	root := filepath.Join(c.Root, filepath.FromSlash(c.Workflow.Archive.Root))
	if _, err := os.Stat(root); err != nil {
		return count, err
	}
	err := filepath.WalkDir(root, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("ARCHIVE_SYMLINK_FORBIDDEN: %s", filename)
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		raw, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(raw, []byte(NotePrefix)) {
			return nil
		} // Preserved historical formats.
		line, _, _ := bytes.Cut(raw, []byte{'\n'})
		if !bytes.HasSuffix(line, []byte(" -->")) {
			return fmt.Errorf("NOTE_METADATA_INVALID")
		}
		var note Note
		if err := json.Unmarshal(line[len(NotePrefix):len(line)-4], &note); err != nil {
			return err
		}
		if err := c.CheckNote(note, validate); err != nil {
			return err
		}
		path, err := c.ArchivePath(note)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(c.Root, filename)
		if err != nil || filepath.ToSlash(relative) != path {
			return fmt.Errorf("NOTE_CLASSIFICATION_PATH_MISMATCH")
		}
		expected, err := c.RenderNote(note)
		if err != nil {
			return err
		}
		if !bytes.Equal(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n")), expected) {
			return fmt.Errorf("NOTE_TEMPLATE_CONTENT_MISMATCH: %s", path)
		}
		count++
		return nil
	})
	return count, err
}
