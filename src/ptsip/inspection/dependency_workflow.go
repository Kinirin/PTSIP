package inspection

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const workflowAdapter = "github-actions"

// scanWorkflows inspects the literal contents of tracked GitHub Actions run
// steps. Runners, expressions and referenced scripts are never executed.
func scanWorkflows(c dependencyContext) ([]DependencyEdge, []DependencyIssue, []string) {
	var edges []DependencyEdge
	var issues []DependencyIssue
	var adapters []string
	for _, name := range c.paths {
		suffix := strings.ToLower(path.Ext(name))
		if !strings.HasPrefix(name, ".github/workflows/") || (suffix != ".yml" && suffix != ".yaml") {
			continue
		}
		if adapters == nil {
			adapters = []string{workflowAdapter}
		}
		found, err := workflowEdges(c, name)
		if err != nil {
			issues = append(issues, DependencyIssue{Adapter: workflowAdapter, Path: name, Message: err.Error()})
			continue
		}
		edges = append(edges, found...)
	}
	return edges, issues, adapters
}

func workflowDocument(raw []byte) (map[string]any, error) {
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("workflow must be UTF-8 YAML")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var value any
	if err := decoder.Decode(&value); err != nil && err != io.EOF {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("workflow must contain exactly one YAML document")
	}
	if value == nil {
		return nil, nil
	}
	document, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("workflow root is not a mapping")
	}
	return document, nil
}

func workflowEdges(c dependencyContext, name string) ([]DependencyEdge, error) {
	raw, err := c.read(name)
	if err != nil {
		return nil, err
	}
	workflow, err := workflowDocument(raw)
	if err != nil {
		return nil, err
	}
	rawJobs, exists := workflow["jobs"]
	if !exists {
		return nil, nil
	}
	jobs, ok := rawJobs.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("workflow jobs is not a mapping")
	}
	var edges []DependencyEdge
	for _, jobName := range slices.Sorted(maps.Keys(jobs)) {
		job, _ := jobs[jobName].(map[string]any)
		steps, _ := job["steps"].([]any)
		phase := workflowPhase(name, jobName)
		for index, value := range steps {
			step, _ := value.(map[string]any)
			command, ok := step["run"].(string)
			if !ok {
				continue
			}
			cwd := workflowWorkingDirectory(workflow, job, step)
			for _, target := range workflowScriptTargets(command) {
				if workflowAbsolutePath(target) {
					continue
				}
				resolved, found := resolveWorkflowScript(c, cwd, target)
				if !found && !strings.Contains(target, "/") {
					continue
				}
				edge := DependencyEdge{
					EvidenceID: fmt.Sprintf("github-actions:%s:%s:%d:%s", name, jobName, index, target),
					Source:     name, Target: target, EdgeType: "INVOKES", Phase: phase, Provenance: "DECLARED",
					Adapter: workflowAdapter, WorkingDirectory: &cwd,
				}
				if found {
					edge.Resolution, edge.TargetScope, edge.ResolvedPath = "RESOLVED", "PROJECT_COMPONENT", &resolved
				} else {
					note := "Explicit local-script invocation could not be resolved from its effective working-directory"
					edge.Resolution, edge.TargetScope, edge.Note = "UNRESOLVED", "UNRESOLVED_TARGET", &note
				}
				edges = append(edges, edge)
			}
		}
	}
	return edges, nil
}

func workflowPhase(name, jobName string) string {
	label := strings.ToLower(name + " " + jobName)
	if strings.Contains(label, "release") || strings.Contains(label, "publish") || strings.Contains(label, "deploy") {
		return "RELEASE"
	}
	if strings.Contains(label, "test") {
		return "TEST"
	}
	return "BUILD"
}

func workflowWorkingDirectory(workflow, job, step map[string]any) string {
	if value, _ := step["working-directory"].(string); strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	for _, owner := range []map[string]any{job, workflow} {
		defaults, _ := owner["defaults"].(map[string]any)
		run, _ := defaults["run"].(map[string]any)
		if value, _ := run["working-directory"].(string); strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "."
}

func workflowAbsolutePath(value string) bool {
	return strings.HasPrefix(value, "/") || (len(value) >= 2 && value[1] == ':')
}

func resolveWorkflowScript(c dependencyContext, cwd, target string) (string, bool) {
	cwd = strings.ReplaceAll(cwd, "\\", "/")
	if workflowAbsolutePath(cwd) || strings.ContainsAny(cwd+target, "\x00\r\n") {
		return "", false
	}
	cwd = path.Clean(cwd)
	if cwd == ".." || strings.HasPrefix(cwd, "../") {
		return "", false
	}
	resolved := path.Join(cwd, target)
	if !c.exists(resolved) {
		return "", false
	}
	return resolved, true
}

func workflowScriptSuffix(token string) string {
	base := path.Base(strings.ReplaceAll(token, "\\", "/"))
	suffix := strings.ToLower(path.Ext(base))
	if len(suffix) == len(base) {
		return ""
	}
	return suffix
}

func isWorkflowScript(token string) bool {
	switch workflowScriptSuffix(token) {
	case ".py", ".ps1", ".sh", ".bash", ".bat", ".cmd":
		return true
	default:
		return false
	}
}

// workflowCommands tokenizes a run line conservatively. Quoted path spaces and
// Windows separators are preserved. This is not a shell interpreter.
func workflowCommands(line string) [][]string {
	var commands [][]string
	var words []string
	var word strings.Builder
	var quote rune
	flushWord := func() {
		if word.Len() != 0 {
			words = append(words, word.String())
			word.Reset()
		}
	}
	flushCommand := func() {
		flushWord()
		if len(words) != 0 {
			commands = append(commands, words)
			words = nil
		}
	}
	chars := []rune(line)
	for i := 0; i < len(chars); i++ {
		char := chars[i]
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				word.WriteRune(char)
			}
			continue
		}
		switch {
		case char == '\'' || char == '"':
			quote = char
		case char == '#' && word.Len() == 0:
			flushCommand()
			return commands
		case unicode.IsSpace(char):
			flushWord()
		case char == ';':
			flushCommand()
		case (char == '&' || char == '|') && i+1 < len(chars) && chars[i+1] == char:
			flushCommand()
			i++
		default:
			word.WriteRune(char)
		}
	}
	flushCommand()
	return commands
}

func workflowScriptTargets(command string) []string {
	var targets []string
	for _, line := range strings.Split(command, "\n") {
		for _, words := range workflowCommands(line) {
			for len(words) != 0 && strings.Contains(words[0], "=") && !isWorkflowScript(words[0]) {
				words = words[1:]
			}
			if len(words) == 0 {
				continue
			}
			target := strings.ReplaceAll(workflowRunnerTarget(words), "\\", "/")
			if target != "" && !slices.Contains(targets, target) {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

func workflowRunnerTarget(words []string) string {
	if isWorkflowScript(words[0]) {
		return words[0]
	}
	runner := strings.ToLower(path.Base(strings.ReplaceAll(words[0], "\\", "/")))
	switch runner {
	case "python", "python3", "python.exe", "python3.exe", "py", "py.exe":
		for _, word := range words[1:] {
			if word == "-c" || word == "-m" {
				return ""
			}
			if strings.HasPrefix(word, "-") {
				continue
			}
			if workflowScriptSuffix(word) == ".py" {
				return word
			}
			break
		}
	case "pwsh", "pwsh.exe", "powershell", "powershell.exe":
		for i, word := range words[1:] {
			if (strings.EqualFold(word, "-file") || strings.EqualFold(word, "/file")) && i+2 < len(words) {
				if workflowScriptSuffix(words[i+2]) == ".ps1" {
					return words[i+2]
				}
				break
			}
			if workflowScriptSuffix(word) == ".ps1" {
				return word
			}
		}
	case "bash", "sh":
		for _, word := range words[1:] {
			if strings.HasPrefix(word, "-") {
				continue
			}
			if suffix := workflowScriptSuffix(word); suffix == ".sh" || suffix == ".bash" {
				return word
			}
			break
		}
	case "cmd", "cmd.exe":
		for _, word := range words[1:] {
			if strings.EqualFold(word, "/c") || strings.EqualFold(word, "/k") {
				continue
			}
			if suffix := workflowScriptSuffix(word); suffix == ".bat" || suffix == ".cmd" {
				return word
			}
			break
		}
	}
	return ""
}
