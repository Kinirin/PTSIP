package machine

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var AgentLevel1 = []string{"APPLICABILITY", "RULE", "ACTION", "EVIDENCE", "OTHER"}
var AgentLevel2 = map[string][]string{
	"APPLICABILITY": {"PATH", "TASK", "ENVIRONMENT", "VERSION", "VCS_CONTEXT"},
	"RULE":          {"MUTATION", "DEPENDENCY", "COMPATIBILITY", "STYLE", "SECURITY", "RESOURCE"},
	"ACTION":        {"COMMAND", "READ", "WRITE", "GENERATE", "BUILD", "INSTALL", "PUBLISH"},
	"EVIDENCE":      {"TEST", "BUILD_RESULT", "STATIC_CHECK", "ARTIFACT", "REVIEW", "STATUS"},
	"OTHER":         {"CONTEXT", "GOAL"},
}

type InstructionAtom struct {
	AtomID          string              `json:"atom_id"`
	Kind            string              `json:"kind"`
	Text            string              `json:"text"`
	LineStart       int                 `json:"line_start"`
	LineEnd         int                 `json:"line_end"`
	HeadingPath     []string            `json:"heading_path"`
	ParentAtomID    *string             `json:"parent_atom_id"`
	DirectLevel1    []string            `json:"direct_level1"`
	InheritedLevel1 []string            `json:"inherited_level1"`
	Level1          []string            `json:"level1"`
	Level2          map[string][]string `json:"level2"`
	Unresolved      bool                `json:"unresolved"`
}

func agentRX(pattern string) *regexp.Regexp { return regexp.MustCompile("(?i)" + pattern) }

var agentHeading = regexp.MustCompile(`^(#{1,6})\s+(.*\S)\s*$`)
var agentList = regexp.MustCompile(`^([ \t]*)(?:[-+*]|\d+[.)])\s+(\S.*)$`)
var agentCondition = agentRX(`^(?:before|after|when|whenever|if|unless|while|during|for\b|on\b)|\b(?:only when|appl(?:y|ies) to|working on|changes? to)\b`)
var agentPathScope = agentRX("\\b(?:under|within|inside|outside|across|in)\\s+`?[^\\s`]+(?:/|\\\\)[^\\s`]*`?")
var agentRule = agentRX(`\b(?:must(?:\s+not)?|shall(?:\s+not)?|should(?:\s+not)?|do not|don't|never|always|only|cannot|can't|may not|required|forbidden|prohibited|avoid|prefer|ensure)\b`)
var agentAction = agentRX(`(?:^|[,;:]\s+)(?:run|use|read|load|resolve|check|prepare|select|inspect|create|follow|record|evaluate|compare|apply|verify|build|publish|deploy|invoke|write|edit|generate|remove|update|add|install|format|lint|test|execute|commit|push|open|review|confirm|report|stop|keep|preserve|reject)\b`)
var agentCommand = agentRX("(?:^|[`\\s])(?:python(?:\\s+-m)?|pytest|git|pip|uv|npm|pnpm|yarn|cargo|go\\s+(?:test|build)|mvn|gradle|make|cmake|ruff|mypy|pyright|twine|ptsip)(?:\\s|`|$)")
var agentEvidence = agentRX(`\b(?:verify|verification|validate|validation|evidence|test|tests|pytest|lint|typecheck|type check|artifact|review|status|pass|passed|fail|failed|success|exact[- ]sha|clean status)\b`)
var agentStatusRX = agentRX(`\b(?:status|pass|passed|fail|failed|success|successful|exact[- ]sha|commit status|clean status)\b`)
var agentGoal = agentRX(`\b(?:the\s+)?(?:goal|objective|purpose|intent)\s+(?:of\s+[^.]{1,80}\s+)?is\s+|\b(?:this|the\s+project|the\s+system|the\s+component)\s+(?:exists|is\s+designed|is\s+intended)\s+to\s+|\bwe\s+aim\s+to\s+`)
var agentContext = agentRX(`\b(?:is|are|means|refers to|consists of|contains|includes|uses|belongs to|lives under|is based on|is built with)\b`)
var agentOther = []*regexp.Regexp{agentGoal, agentContext, agentRX(`^[^:]{1,80}:\s+\S`), agentRX(`\s(?:--|—|->|→)\s`), agentRX(`^(?:Python|Node(?:\.js)?|Java|Go|Rust|R)\s+\d+(?:\.\d+){0,3}\+?$`), agentRX(`^https?://\S+$`)}

func agentHas(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
func agentOrdered(vocabulary []string, hits map[string]bool) []string {
	out := []string{}
	for _, label := range vocabulary {
		if hits[label] {
			out = append(out, label)
		}
	}
	return out
}
func agentSpace(value string) string { return strings.Join(strings.Fields(value), " ") }

// ParseAgentMarkdown preserves source ranges and introductory ancestry before classifying.
func ParseAgentMarkdown(text string) []InstructionAtom {
	atoms := []InstructionAtom{}
	headings := []string{}
	paragraph := []string{}
	paragraphStart := 0
	type listEntry struct{ indent, index int }
	stack := []listEntry{}
	pending, active, codeParent := -1, -1, -1
	code := []string{}
	codeStart := 0
	inCode := false
	appendAtom := func(kind, text string, start, end, parent int) int {
		text = agentSpace(text)
		if text == "" {
			return -1
		}
		var parentID *string
		if parent >= 0 {
			id := atoms[parent].AtomID
			parentID = &id
		}
		atoms = append(atoms, InstructionAtom{AtomID: fmt.Sprintf("A%04d", len(atoms)+1), Kind: kind, Text: text, LineStart: start, LineEnd: end, HeadingPath: append([]string{}, headings...), ParentAtomID: parentID})
		return len(atoms) - 1
	}
	flush := func(end int) {
		if len(paragraph) == 0 {
			return
		}
		idx := appendAtom("paragraph", strings.Join(paragraph, " "), paragraphStart, end, -1)
		paragraph = []string{}
		paragraphStart = 0
		pending = -1
		if idx >= 0 && strings.HasSuffix(atoms[idx].Text, ":") {
			pending = idx
		}
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, raw := range lines {
		no := i + 1
		stripped := strings.TrimSpace(raw)
		if strings.HasPrefix(stripped, "```") {
			if inCode {
				end := no - 1
				if end < codeStart {
					end = codeStart
				}
				appendAtom("code_block", strings.Join(code, "\n"), codeStart, end, codeParent)
				code = []string{}
				codeParent = -1
				inCode = false
			} else {
				flush(no - 1)
				inCode = true
				codeStart = no + 1
				codeParent = pending
				pending = -1
				active = -1
				stack = nil
			}
			continue
		}
		if inCode {
			if stripped != "" {
				code = append(code, stripped)
			}
			continue
		}
		if h := agentHeading.FindStringSubmatch(stripped); h != nil {
			flush(no - 1)
			level := len(h[1]) - 1
			if level < len(headings) {
				headings = headings[:level]
			}
			headings = append(headings, strings.TrimSpace(h[2]))
			stack = nil
			pending = -1
			active = -1
			continue
		}
		if stripped == "" {
			flush(no - 1)
			if len(stack) > 0 {
				stack = nil
				active = -1
			}
			continue
		}
		if item := agentList.FindStringSubmatch(raw); item != nil {
			flush(no - 1)
			indent := 0
			for _, r := range item[1] {
				if r == '\t' {
					indent += 4 - indent%4
				} else {
					indent++
				}
			}
			if len(stack) == 0 {
				active = pending
				pending = -1
			}
			for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
				stack = stack[:len(stack)-1]
			}
			parent := active
			if len(stack) > 0 {
				parent = stack[len(stack)-1].index
			}
			idx := appendAtom("list_item", item[2], no, no, parent)
			if idx >= 0 {
				stack = append(stack, listEntry{indent, idx})
			}
			continue
		}
		if len(paragraph) == 0 {
			pending = -1
			active = -1
			stack = nil
			paragraphStart = no
		}
		paragraph = append(paragraph, stripped)
	}
	if inCode && len(code) > 0 {
		appendAtom("code_block", strings.Join(code, "\n"), codeStart, len(lines), codeParent)
	}
	flush(len(lines))
	return atoms
}

func agentDirect(atom InstructionAtom) []string {
	hits := map[string]bool{}
	t := atom.Text
	if agentCondition.MatchString(t) || agentPathScope.MatchString(t) {
		hits["APPLICABILITY"] = true
	}
	if agentRule.MatchString(t) {
		hits["RULE"] = true
	}
	if agentAction.MatchString(t) || (atom.Kind == "code_block" && agentCommand.MatchString(t)) {
		hits["ACTION"] = true
	}
	if agentEvidence.MatchString(t) && (hits["ACTION"] || agentStatusRX.MatchString(t) || agentRX(`\b(?:evidence|verification|validation|review)\b`).MatchString(t)) {
		hits["EVIDENCE"] = true
	}
	for _, rx := range agentOther {
		if rx.MatchString(t) {
			hits["OTHER"] = true
			break
		}
	}
	return agentOrdered(AgentLevel1, hits)
}

func agentSecond(atom InstructionAtom, parents []string) map[string][]string {
	text := atom.Text
	out := map[string][]string{}
	hits := func(pattern string) bool { return agentRX(pattern).MatchString(text) }
	clause := func(verbs string) bool { return hits(`(?:^|[,;:]\s+)(?:` + verbs + `)\b`) }
	patterns := map[string]map[string]string{
		"APPLICABILITY": {"PATH": `\b(?:path|file|directory|directories)\b`, "TASK": `\b(?:task|work|workflow|session|implementation|release|migration|review|documentation|verification)\b`, "ENVIRONMENT": `\b(?:windows|linux|macos|CI|runner|shell|powershell|bash|environment|venv)\b`, "VERSION": `\b(?:version|tool|python|node(?:\.js)?|java|go|rust)\s+v?\d+(?:\.\d+){0,3}\b`, "VCS_CONTEXT": "\\b(?:git|branch|HEAD|commit|tag|checkout|merge|rebase|stash|origin/[^\\s`]+)\\b"},
		"RULE":          {"MUTATION": `\b(?:edit|write|change|modify|move|relocate|delete|remove|revert|reset|clean|stash|overwrite|create)\b`, "DEPENDENCY": `\b(?:dependency|dependencies|import|imports|depends on|dependency direction)\b`, "COMPATIBILITY": `\b(?:compatibility|compatible|backward|forward|supported versions?|version support)\b`, "STYLE": `\b(?:style|naming|formatting|format|line length|comments?|docstrings?)\b`, "SECURITY": `\b(?:security|secret|credential|password|token|authentication|authorization|permission|privacy)\b`, "RESOURCE": `\b(?:cpu|memory|disk|network|time budget|token budget|credit cost|resource)\b`},
		"EVIDENCE":      {"TEST": `\b(?:test|tests|testing|pytest|regression|smoke)\b`, "BUILD_RESULT": `\b(?:build result|built distribution|build output|build succeeded|build failed|distribution)\b`, "STATIC_CHECK": `\b(?:lint|static check|type check|typecheck|ruff|mypy|pyright|format check)\b`, "ARTIFACT": `\b(?:artifact|wheel|sdist|package output|generated output|distribution file)\b`, "REVIEW": `\b(?:review|final diff|approval|approved)\b`, "STATUS": agentStatusRX.String()},
	}
	for _, parent := range parents {
		h := map[string]bool{}
		for key, p := range patterns[parent] {
			h[key] = hits(p)
		}
		switch parent {
		case "APPLICABILITY":
			h["PATH"] = h["PATH"] || agentPathScope.MatchString(text)
		case "ACTION":
			h["COMMAND"] = agentCommand.MatchString(text) || atom.Kind == "code_block"
			h["READ"] = clause("read|load|inspect|open")
			h["WRITE"] = clause("write|edit|update|change|modify|remove|delete|move|relocate|create")
			h["GENERATE"] = clause("generate|scaffold|materialize|render")
			h["BUILD"] = clause("build|compile|package")
			h["INSTALL"] = clause("install") || hits(`(?:^|[,;:]\s+)(?:pip|npm|pnpm|yarn)\s+(?:install|add)\b`)
			h["PUBLISH"] = clause("publish|release|deploy|upload|promote")
		case "OTHER":
			h["GOAL"] = agentGoal.MatchString(text)
			for _, rx := range agentOther[1:] {
				if rx.MatchString(text) {
					h["CONTEXT"] = true
					break
				}
			}
		}
		out[parent] = agentOrdered(AgentLevel2[parent], h)
	}
	return out
}

func ClassifyAgentMarkdown(text string) []InstructionAtom {
	atoms := ParseAgentMarkdown(text)
	byID := map[string]InstructionAtom{}
	direct := map[string][]string{}
	for _, a := range atoms {
		byID[a.AtomID] = a
		direct[a.AtomID] = agentDirect(a)
	}
	for i := range atoms {
		a := &atoms[i]
		inherited := map[string]bool{}
		parentID := a.ParentAtomID
		seen := map[string]bool{}
		for parentID != nil && !seen[*parentID] {
			seen[*parentID] = true
			p, ok := byID[*parentID]
			if !ok {
				break
			}
			if strings.HasSuffix(p.Text, ":") {
				for _, label := range direct[p.AtomID] {
					if label != "OTHER" {
						inherited[label] = true
					}
				}
			}
			parentID = p.ParentAtomID
		}
		a.DirectLevel1 = direct[a.AtomID]
		a.InheritedLevel1 = agentOrdered(AgentLevel1, inherited)
		for _, label := range a.DirectLevel1 {
			inherited[label] = true
		}
		a.Level1 = agentOrdered(AgentLevel1, inherited)
		a.Level2 = agentSecond(*a, a.Level1)
		a.Unresolved = len(a.Level1) == 0
	}
	return atoms
}

// agentSection names the admitted current owner, never a reconstructed legacy policy.
func agentSection(r *Repository, id, section string) (any, error) {
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	policy, err := resolver.Policy(id)
	if err != nil {
		return nil, err
	}
	if Map(policy["policy"])["status"] != "ACTIVE" {
		return nil, fmt.Errorf("agent contract owner is not ACTIVE: %s", id)
	}
	value, ok := Map(policy["rules"])[section]
	if !ok {
		return nil, fmt.Errorf("missing exact agent contract section %s#%s", id, section)
	}
	return value, nil
}

func ValidateAgentTaxonomy(r *Repository) error {
	first, err := agentSection(r, "MPD-INFO-0001", "unit_mpd_0010_f1b93fa1851f")
	if err != nil {
		return err
	}
	if !agentEqualStrings(first, AgentLevel1) {
		return fmt.Errorf("Level 1 vocabulary mismatch")
	}
	second, err := agentSection(r, "MPD-INFO-0001", "unit_mpd_0010_a7750d894301")
	if err != nil {
		return err
	}
	for parent, expected := range AgentLevel2 {
		if !agentEqualStrings(Map(second)[parent], expected) {
			return fmt.Errorf("Level 2 vocabulary mismatch for %s", parent)
		}
	}
	unresolved, err := agentSection(r, "MPD-CNTR-0001", "unit_mpd_0010_804417a00130")
	if err != nil {
		return err
	}
	if unresolved != false {
		return fmt.Errorf("UNRESOLVED must not be a namespace")
	}
	return nil
}
func agentEqualStrings(value any, expected []string) bool {
	list := List(value)
	if len(list) != len(expected) {
		return false
	}
	for i, v := range expected {
		if list[i] != v {
			return false
		}
	}
	return true
}

func ClassifyAgentFile(r *Repository, source string) (Object, error) {
	if err := ValidateAgentTaxonomy(r); err != nil {
		return nil, err
	}
	relative, err := r.Scope(source)
	if err != nil {
		return nil, err
	}
	path, err := r.Path(source)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	atoms := ClassifyAgentMarkdown(strings.ReplaceAll(string(data), "\r\n", "\n"))
	return Object{"schema_version": "ptsip-agent-instruction-classification-trial/v1", "policy_ref": "MPD-INFO-0001#unit_mpd_0010_f1b93fa1851f", "source": relative, "level_1_vocabulary": AgentLevel1, "level_2_vocabulary": AgentLevel2, "summary": agentAtomSummary(atoms), "atoms": atoms}, nil
}
func agentAtomSummary(atoms []InstructionAtom) Object {
	unresolved := 0
	counts := Object{}
	for _, label := range AgentLevel1 {
		counts[label] = 0
	}
	for _, a := range atoms {
		if a.Unresolved {
			unresolved++
		}
		for _, label := range a.Level1 {
			counts[label] = counts[label].(int) + 1
		}
	}
	return Object{"atom_count": len(atoms), "classified_count": len(atoms) - unresolved, "unresolved_count": unresolved, "level_1_counts": counts}
}
