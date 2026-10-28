package releasenote

import (
	"fmt"
	"sort"
	"strings"
)

type Section struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}
type Readme struct {
	Schema       string    `json:"schema_version"`
	Language     string    `json:"language"`
	Title        string    `json:"title"`
	Introduction string    `json:"introduction"`
	Navigation   string    `json:"navigation,omitempty"`
	Sections     []Section `json:"sections"`
}

// HeadingLines ignores fenced examples, including examples containing titles.
func HeadingLines(text string) map[int]int {
	result := map[int]int{}
	fence := byte(0)
	width := 0
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
			n := 0
			for n < len(trimmed) && trimmed[n] == trimmed[0] {
				n++
			}
			if n >= 3 {
				if fence == 0 {
					fence, width = trimmed[0], n
					continue
				}
				if fence == trimmed[0] && n >= width && strings.TrimSpace(trimmed[n:]) == "" {
					fence, width = 0, 0
					continue
				}
			}
		}
		if fence != 0 {
			continue
		}
		for level := 1; level <= 6; level++ {
			if strings.HasPrefix(line, strings.Repeat("#", level)+" ") {
				result[i] = level
				break
			}
		}
	}
	return result
}

func ParseReadme(raw string) (Readme, error) {
	text := strings.ReplaceAll(strings.TrimPrefix(raw, "\ufeff"), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	headings := HeadingLines(text)
	title := -1
	positions := []int{}
	for i, level := range headings {
		if level == 1 {
			if title != -1 {
				return Readme{}, fmt.Errorf("README_SINGLE_TITLE_REQUIRED")
			}
			title = i
		}
		if level == 2 {
			positions = append(positions, i)
		}
	}
	sort.Ints(positions)
	if title < 0 || len(positions) == 0 || positions[0] <= title {
		return Readme{}, fmt.Errorf("README_TITLE_AND_SECTIONS_REQUIRED")
	}
	doc := Readme{Schema: "ptsip-readme/v1", Language: "en", Title: strings.TrimPrefix(lines[title], "# "), Navigation: strings.TrimSpace(strings.Join(lines[:title], "\n")), Introduction: strings.TrimSpace(strings.Join(lines[title+1:positions[0]], "\n")), Sections: []Section{}}
	for i, start := range positions {
		end := len(lines)
		if i+1 < len(positions) {
			end = positions[i+1]
		}
		doc.Sections = append(doc.Sections, Section{Heading: strings.TrimPrefix(lines[start], "## "), Body: strings.TrimSpace(strings.Join(lines[start+1:end], "\n"))})
	}
	return doc, nil
}

func RenderReadme(doc Readme) string {
	var out strings.Builder
	if doc.Navigation != "" {
		out.WriteString(doc.Navigation + "\n\n")
	}
	out.WriteString("# " + doc.Title + "\n\n" + doc.Introduction + "\n\n")
	for _, section := range doc.Sections {
		out.WriteString("## " + section.Heading + "\n\n" + section.Body + "\n\n")
	}
	return strings.TrimSpace(out.String()) + "\n"
}

func UpdateDeclaredVersions(doc *Readme, versions map[string]string, notes []string) error {
	const start = "<!-- ptsip:declared-versions:start -->"
	const end = "<!-- ptsip:declared-versions:end -->"
	if len(doc.Sections) == 0 || doc.Sections[0].Heading != "Published release and component versions" {
		return fmt.Errorf("README_VERSION_SECTION_REQUIRED")
	}
	body := doc.Sections[0].Body
	a, b := strings.Index(body, start), strings.Index(body, end)
	if (a < 0) != (b < 0) || (a >= 0 && b <= a) || strings.Count(body, start) > 1 || strings.Count(body, end) > 1 {
		return fmt.Errorf("README_DECLARED_VERSION_MARKERS_INVALID")
	}
	var block strings.Builder
	block.WriteString(start + "\n### Current source declarations\n\nThese are source declarations; publication evidence is recorded separately above.\n\n| Component | Declared version |\n| --- | --- |\n")
	keys := make([]string, 0, len(versions))
	for key := range versions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&block, "| `%s` | `%s` |\n", key, versions[key])
	}
	block.WriteString("\nEnglish change records:\n\n")
	for _, note := range notes {
		fmt.Fprintf(&block, "- [%s](%s)\n", note, note)
	}
	block.WriteString(end)
	if a < 0 {
		body = strings.TrimSpace(body) + "\n\n" + block.String()
	} else {
		body = body[:a] + block.String() + body[b+len(end):]
	}
	doc.Sections[0].Body = body
	return nil
}
