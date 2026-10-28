package releasenote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	version "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/version"
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

type Edit struct {
	Path       string
	Content    []byte
	CreateOnly bool
	Original   []byte
}
type Detail struct {
	Summary       string `json:"summary"`
	Details       string `json:"technical_details"`
	Documentation []Link `json:"documentation"`
}
type Input struct {
	Notes map[string]Detail `json:"notes"`
}

func destination(root, relative string) (string, error) {
	if relative == "" || path.Clean(relative) != relative || path.IsAbs(relative) || strings.ContainsAny(relative, "\\:\x00") {
		return "", fmt.Errorf("INVALID_DOCUMENTATION_OUTPUT_PATH: %s", relative)
	}
	current := root
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("INVALID_DOCUMENTATION_OUTPUT_PATH")
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("DOCUMENTATION_OUTPUT_SYMLINK: %s", relative)
		}
	}
	return current, nil
}

func (c *Catalog) Prepare(base, head string, input Input, validate authoring.Validator) ([]Edit, []version.DeclaredVersionChange, error) {
	if err := c.CheckCurrent(validate); err != nil {
		return nil, nil, err
	}
	report, changes, err := version.DeclaredChanges(c.Root, base, head)
	if err != nil {
		return nil, nil, err
	}
	selected := map[string]bool{}
	for _, change := range changes {
		selected[change.Component] = true
	}
	for component := range input.Notes {
		if !selected[component] {
			return nil, nil, fmt.Errorf("DETAIL_HAS_NO_DECLARED_VERSION_CHANGE: %s", component)
		}
	}
	if len(changes) == 0 {
		return []Edit{}, changes, nil
	}
	edits := []Edit{}
	notePaths := []string{}
	for _, change := range changes {
		kind := kindOf(change.Component)
		route := c.Workflow.Archive.Types[kind]
		documentation := strings.ReplaceAll(route.Documentation, "{engine}", strings.TrimPrefix(change.Component, "engine."))
		details := "Changed source paths:\n\n"
		for _, file := range change.Files {
			details += "- `" + file + "`\n"
		}
		details += "\nThis mechanical record does not infer functional verification from source differences."
		note := Note{Schema: "ptsip-release-note/v1", Language: "en", Kind: kind, Change: change, Base: base, Head: head, Digest: report.VersionDigest,
			Summary: fmt.Sprintf("Declared %s version changed from `%s` to `%s`.", change.Component, change.Previous, change.Version), Details: details,
			Documentation: []Link{{Title: "Detailed technical contract or documentation", Path: documentation}}}
		if detail, present := input.Notes[change.Component]; present {
			if detail.Summary != "" {
				note.Summary = detail.Summary
			}
			if detail.Details != "" {
				note.Details = detail.Details
			}
			if detail.Documentation != nil {
				note.Documentation = detail.Documentation
			}
		}
		if err := c.CheckNote(note, validate); err != nil {
			return nil, nil, err
		}
		relative, err := c.ArchivePath(note)
		if err != nil {
			return nil, nil, err
		}
		content, err := c.RenderNote(note)
		if err != nil {
			return nil, nil, err
		}
		edits = append(edits, Edit{Path: relative, Content: content, CreateOnly: true})
		notePaths = append(notePaths, relative)
	}
	filename, err := authoring.LocalFile(c.Root, c.Workflow.Readme.Path)
	if err != nil {
		return nil, nil, err
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}
	readme, err := ParseReadme(string(raw))
	if err != nil {
		return nil, nil, err
	}
	versions, err := version.VersionsAtCommit(c.Root, head)
	if err != nil {
		return nil, nil, err
	}
	if err := UpdateDeclaredVersions(&readme, versions, notePaths); err != nil {
		return nil, nil, err
	}
	if err := c.Validate(validate, BasicID, readme); err != nil {
		return nil, nil, err
	}
	edits = append(edits, Edit{Path: c.Workflow.Readme.Path, Content: []byte(RenderReadme(readme)), Original: raw})
	return edits, changes, nil
}

// Apply stages every result before mutation and rolls back completed writes on
// failure. Archives are create-only; identical replay is idempotent.
func (c *Catalog) Apply(edits []Edit) ([]string, error) {
	type prepared struct {
		edit           Edit
		filename, temp string
		old            []byte
		existed        bool
	}
	items := []prepared{}
	seen := map[string]bool{}
	for _, edit := range edits {
		if seen[edit.Path] || (edit.Path != c.Workflow.Readme.Path && !strings.HasPrefix(edit.Path, c.Workflow.Archive.Root+"/")) {
			return nil, fmt.Errorf("UNREGISTERED_DOCUMENTATION_OUTPUT")
		}
		seen[edit.Path] = true
		if edit.Path != c.Workflow.Readme.Path && !edit.CreateOnly {
			return nil, fmt.Errorf("ARCHIVE_REQUIRES_CREATE_ONLY")
		}
		filename, err := destination(c.Root, edit.Path)
		if err != nil {
			return nil, err
		}
		old, err := os.ReadFile(filename)
		existed := err == nil
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if edit.Path == c.Workflow.Readme.Path && (edit.Original == nil || !bytes.Equal(old, edit.Original)) {
			return nil, fmt.Errorf("README_CHANGED_AFTER_PREPARATION")
		}
		if edit.CreateOnly {
			line, _, _ := bytes.Cut(edit.Content, []byte{'\n'})
			if !bytes.HasPrefix(line, []byte(NotePrefix)) || !bytes.HasSuffix(line, []byte(" -->")) {
				return nil, fmt.Errorf("ARCHIVE_REQUIRES_ENGLISH_NOTE_METADATA")
			}
			var note Note
			if err := json.Unmarshal(line[len(NotePrefix):len(line)-4], &note); err != nil {
				return nil, err
			}
			path, err := c.ArchivePath(note)
			if err != nil || path != edit.Path || note.Language != "en" {
				return nil, fmt.Errorf("ARCHIVE_REQUIRES_ORIGINAL_ENGLISH_NOTE")
			}
			content, err := c.RenderNote(note)
			if err != nil || !bytes.Equal(content, edit.Content) {
				return nil, fmt.Errorf("ARCHIVE_TEMPLATE_MISMATCH")
			}
		}
		if existed && edit.CreateOnly && !bytes.Equal(bytes.ReplaceAll(old, []byte("\r\n"), []byte("\n")), edit.Content) {
			return nil, fmt.Errorf("ARCHIVE_CONTENT_CONFLICT: %s", edit.Path)
		}
		if existed && bytes.Equal(bytes.ReplaceAll(old, []byte("\r\n"), []byte("\n")), edit.Content) {
			continue
		}
		items = append(items, prepared{edit: edit, filename: filename, old: old, existed: existed})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].edit.Path < items[j].edit.Path })
	defer func() {
		for _, item := range items {
			if item.temp != "" {
				_ = os.Remove(item.temp)
			}
		}
	}()
	for i := range items {
		if err := os.MkdirAll(filepath.Dir(items[i].filename), 0755); err != nil {
			return nil, err
		}
		temp, err := os.CreateTemp(filepath.Dir(items[i].filename), ".documentation-*")
		if err != nil {
			return nil, err
		}
		items[i].temp = temp.Name()
		if _, err := temp.Write(items[i].edit.Content); err != nil {
			temp.Close()
			return nil, err
		}
		if err := temp.Chmod(0644); err != nil {
			temp.Close()
			return nil, err
		}
		if err := temp.Close(); err != nil {
			return nil, err
		}
	}
	paths := []string{}
	for i, item := range items {
		var applyErr error
		if item.edit.CreateOnly {
			applyErr = os.Link(item.temp, item.filename)
		} else {
			applyErr = os.Rename(item.temp, item.filename)
		}
		if err := applyErr; err != nil {
			failures := []error{err}
			for j := i - 1; j >= 0; j-- {
				previous := items[j]
				var recovery error
				if previous.existed {
					recovery = os.WriteFile(previous.filename, previous.old, 0644)
				} else {
					recovery = os.Remove(previous.filename)
				}
				if recovery != nil {
					failures = append(failures, recovery)
				}
			}
			return nil, errors.Join(failures...)
		}
		paths = append(paths, item.edit.Path)
	}
	return paths, nil
}

func (c *Catalog) CheckCurrent(validate authoring.Validator) error {
	filename, err := authoring.LocalFile(c.Root, c.Workflow.Readme.Path)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	document, err := ParseReadme(string(raw))
	if err != nil {
		return err
	}
	if err := c.Validate(validate, BasicID, document); err != nil {
		return err
	}
	registry, err := authoring.ReadDocument(c.Root, c.Workflow.Translation.Registry)
	if err != nil {
		return err
	}
	if err := ValidateRegistry(c.Root, registry, validate); err != nil {
		return err
	}
	var contract any
	if err := ReadJSON(c.Root, c.Workflow.VersionContract, &contract); err != nil {
		return err
	}
	if err := c.Validate(validate, "urn:ptsip:release:versions-different-contract:v1", contract); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(c.Root, "docs/releasenote/engine/README.md")); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("WITHDRAWN_ENGINE_README_PRESENT_OR_UNREADABLE: %v", err)
	}
	_, err = c.CheckArchive(validate)
	return err
}

func Run(root, operation, base, head, input string, validate authoring.Validator) (any, int) {
	report := map[string]any{"status": "FAIL", "scope": "DOCUMENTATION_FORMAT_AND_DECLARED_VERSION_CHANGES", "publication_authorized": false, "execution_authorized": false}
	fail := func(err error) (any, int) { report["problem"] = err.Error(); return report, 2 }
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fail(err)
	}
	catalog, err := Load(absolute)
	if err != nil {
		return fail(err)
	}
	if operation == "check" {
		if err := catalog.CheckCurrent(validate); err != nil {
			return fail(err)
		}
		report["status"] = "VALID"
		return report, 0
	}
	if operation != "generate" && operation != "preview" {
		return fail(fmt.Errorf("UNREGISTERED_DOCUMENTATION_OPERATION"))
	}
	details := Input{}
	if input != "" {
		doc, err := authoring.ReadDocument(absolute, input)
		if err != nil {
			return fail(err)
		}
		raw, err := json.Marshal(doc)
		if err != nil {
			return fail(err)
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&details); err != nil {
			return fail(err)
		}
	}
	edits, changes, err := catalog.Prepare(base, head, details, validate)
	if err != nil {
		return fail(err)
	}
	paths := []string{}
	for _, edit := range edits {
		paths = append(paths, edit.Path)
	}
	status := "PREPARED"
	if operation == "generate" {
		paths, err = catalog.Apply(edits)
		if err != nil {
			return fail(err)
		}
		status = "GENERATED"
	}
	if len(changes) == 0 {
		status = "NO_VERSION_CHANGES"
	}
	report["status"], report["changes"], report["output_paths"] = status, changes, paths
	report["translation"] = "CURRENT_REGISTERED_TARGETS_ONLY_NO_ARCHIVE"
	return report, 0
}
