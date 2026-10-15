package releasecontroller

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const PresentationPath = ".github/checks/release/release-presentation.json"

type Presentation struct {
	Schema        string   `json:"schema_version"`
	Title         string   `json:"title_pattern"`
	Sections      []string `json:"section_order"`
	IdentityOrder []string `json:"identity_order"`
}

func ReleasePresentation(root string, s Selection, snapshot Snapshot) (string, string, error) {
	var contract Presentation
	if err := ReadJSON(root, PresentationPath, &contract); err != nil {
		return "", "", err
	}
	if contract.Schema != "ptsip-release-presentation/v1" || contract.Title != "PTSIP_Package {package_version}" || strings.Join(contract.Sections, ",") != "PTSIP Release,PTSIP Tool,Project Profile,PTSIP Engine" || strings.Join(contract.IdentityOrder, ",") != "PTSIP_Package,PTSIP_Tool,Project Profile,Engine" {
		return "", "", fmt.Errorf("release presentation contract unresolved")
	}
	body := "# " + contract.Sections[0] + "\n\n```yaml\n" + IdentityText(snapshot, "", "") + "```\n\n"
	composition, err := ReadComposition(root)
	if err != nil {
		return "", "", err
	}
	if composition.PackagePyPI != "" {
		body += "PyPI Package: `PTSIP==" + composition.PackagePyPI + "`. This installs Tool, Profiles and every registered engine.\n\n"
	}
	selected := map[string]Component{}
	for _, c := range s.Components {
		selected[c.ID] = c
	}
	for _, kind := range []string{"tool", "profiles"} {
		heading, id := contract.Sections[1], "tool"
		if kind == "profiles" {
			heading, id = contract.Sections[2], "profiles"
			for _, c := range snapshot.Components {
				if c.ID == id {
					heading += " " + c.Version
				}
			}
		}
		body += "# " + heading + "\n\n"
		c, ok := selected[id]
		if !ok {
			body += "This component is unchanged by this release.\n\n"
			continue
		}
		p, err := SafePath(root, c.Note)
		if err != nil {
			return "", "", err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", "", err
		}
		lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
		if len(lines) == 0 || !strings.HasPrefix(lines[0], "# ") {
			return "", "", fmt.Errorf("component note needs its title: %s", c.Note)
		}
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "#") {
				line = "#" + line
			}
			body += line + "\n"
		}
		body += "\n"
	}
	body += "# " + contract.Sections[3] + "\n\n"
	var index struct {
		Entries []struct {
			Kind       string
			ID         string `json:"engine_id"`
			Capability string `json:"capability_id"`
		} `json:"official_engine_entries"`
	}
	if err := ReadJSON(root, "src/ptsip/engines/index.json", &index); err != nil {
		return "", "", err
	}
	engines := []Component{}
	for _, c := range s.Components {
		if c.Kind == "engine" {
			engines = append(engines, c)
		}
	}
	sort.Slice(engines, func(i, j int) bool { return engines[i].ID < engines[j].ID })
	for _, c := range engines {
		p, err := SafePath(root, c.Note)
		if err != nil {
			return "", "", err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", "", err
		}
		note := strings.ReplaceAll(string(b), "\r\n", "\n")
		note = strings.TrimSpace(strings.TrimPrefix(note, strings.SplitN(note, "\n", 2)[0]))
		valid := false
		for _, entry := range index.Entries {
			if strings.EqualFold(entry.Kind, c.ID) {
				valid = strings.Contains(note, "Engine ID: `"+entry.ID+"`") && strings.Contains(note, "Capability: `"+entry.Capability+"`")
			}
		}
		if !valid || !strings.Contains(note, "### Purpose\n") || !strings.Contains(note, "### Contract changes\n") || strings.Contains(note, "\n# ") || strings.Contains(note, "\n## ") {
			return "", "", fmt.Errorf("engine note identity or section contract differs: %s", c.Note)
		}
		body += "## " + strings.ToUpper(c.ID[:1]) + c.ID[1:] + " " + c.Version + "\n\n" + note + "\n\n"
	}
	if len(engines) == 0 {
		body += "No engine changes are included in this release.\n"
	}
	title := strings.ReplaceAll(contract.Title, "{package_version}", snapshot.Version)
	if composition.PackagePyPI != "" {
		title = "PTSIP_Package " + composition.PackagePyPI + " (" + snapshot.Version + ")"
	}
	return title, strings.TrimSpace(body) + "\n", nil
}
