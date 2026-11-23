package inspection

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

const jsAdapter = "javascript-typescript"
const npmAdapter = "npm-manifest"

var jsExtensions = []string{".cjs", ".cts", ".js", ".jsx", ".mjs", ".mts", ".ts", ".tsx"}
var npmSections = []string{"dependencies", "devDependencies", "peerDependencies", "optionalDependencies"}

type npmManifest struct {
	name     string
	sections map[string][]string
}

func scanJavaScript(c dependencyContext) ([]DependencyEdge, []DependencyIssue, []string) {
	var edges []DependencyEdge
	var issues []DependencyIssue
	var npmSeen, jsSeen bool
	manifests := map[string]npmManifest{}
	names := map[string][]string{}
	addIssue := func(adapter, file, message string) {
		issues = append(issues, DependencyIssue{Adapter: adapter, Path: file, Message: message})
	}
	for _, file := range c.paths {
		if path.Base(file) != "package.json" {
			continue
		}
		npmSeen = true
		data, err := c.read(file)
		if err != nil {
			addIssue(npmAdapter, file, err.Error())
			continue
		}
		decoded, err := jsondoc.Decode(bytes.TrimPrefix(data, []byte("\ufeff")))
		payload, ok := decoded.(map[string]any)
		if err != nil || !ok {
			message := "package.json root must be an object"
			if err != nil {
				message = err.Error()
			}
			addIssue(npmAdapter, file, message)
			continue
		}
		manifest := npmManifest{sections: map[string][]string{}}
		if raw, found := payload["name"]; found {
			manifest.name, ok = raw.(string)
			if !ok || manifest.name == "" {
				addIssue(npmAdapter, file, "package.json name must be a non-empty string")
			}
		}
		for _, section := range npmSections {
			raw, found := payload[section]
			if !found {
				continue
			}
			entries, ok := raw.(map[string]any)
			if !ok {
				addIssue(npmAdapter, file, section+" must be an object")
				continue
			}
			for name, requirement := range entries {
				if _, ok := requirement.(string); !ok || name == "" {
					addIssue(npmAdapter, file, fmt.Sprintf("dependency %q in %s must have a name and string version", name, section))
					continue
				}
				manifest.sections[section] = append(manifest.sections[section], name)
			}
			sort.Strings(manifest.sections[section])
		}
		manifests[file] = manifest
		if manifest.name != "" {
			names[manifest.name] = append(names[manifest.name], file)
		}
	}
	for name, files := range names {
		if len(files) > 1 {
			for _, file := range files {
				addIssue(npmAdapter, file, fmt.Sprintf("duplicate local npm package name %q; package identity is ambiguous", name))
			}
		}
	}
	for _, file := range c.paths {
		if manifest, present := manifests[file]; present {
			for _, section := range npmSections {
				for _, name := range manifest.sections[section] {
					edges = append(edges, npmManifestEdge(file, section, name, names[name]))
				}
			}
		}
		if !isJavaScriptSource(file) {
			continue
		}
		jsSeen = true
		data, err := c.read(file)
		if err != nil {
			addIssue(jsAdapter, file, err.Error())
			continue
		}
		if !utf8.Valid(data) {
			addIssue(jsAdapter, file, "JavaScript or TypeScript source must be UTF-8")
			continue
		}
		lexer := jsLexer{text: strings.TrimPrefix(string(data), "\ufeff"), line: 1}
		extension := strings.ToLower(path.Ext(file))
		lexer.jsx = extension == ".jsx" || extension == ".tsx"
		lexer.code(false, 0)
		if lexer.problem != "" {
			addIssue(jsAdapter, file, lexer.problem)
		}
		nearest := nearestNPMManifest(file, manifests, c.files)
		for index, token := range lexer.tokens {
			target, kind, dynamic, problem := jsDependency(lexer.tokens, index)
			if problem != "" {
				addIssue(jsAdapter, file, fmt.Sprintf("line %d: %s", token.line, problem))
			}
			if kind == "" {
				continue
			}
			edge := DependencyEdge{Source: file, Target: target, EdgeType: kind, Phase: jsPhase(file),
				Provenance: "OBSERVED", Line: &token.line, Adapter: jsAdapter}
			if dynamic {
				edge.Resolution, edge.TargetScope = "DYNAMIC", "UNRESOLVED_TARGET"
				note := "JavaScript or TypeScript dependency target is not a plain string literal"
				edge.Note = &note
				edge.EvidenceID = fmt.Sprintf("javascript-dynamic:%s:%d:%s", file, token.line, token.text)
			} else {
				resolveJSDependency(&edge, c, names, nearest)
				edge.EvidenceID = fmt.Sprintf("javascript:%s:%d:%s:%s", file, token.line, strings.ToLower(kind), target)
			}
			edges = append(edges, edge)
		}
	}
	var adapters []string
	if jsSeen {
		adapters = append(adapters, jsAdapter)
	}
	if npmSeen {
		adapters = append(adapters, npmAdapter)
	}
	return edges, issues, adapters
}

func isJavaScriptSource(file string) bool {
	extension := strings.ToLower(path.Ext(file))
	index := sort.SearchStrings(jsExtensions, extension)
	return index < len(jsExtensions) && jsExtensions[index] == extension
}

func npmPhase(section string) string {
	if section == "devDependencies" {
		return "BUILD"
	}
	return "RUNTIME"
}

func npmManifestEdge(file, section, name string, local []string) DependencyEdge {
	edge := DependencyEdge{EvidenceID: "npm-manifest:" + file + ":" + section + ":" + name,
		Source: file, Target: name, EdgeType: "IMPORTS", Phase: npmPhase(section),
		Provenance: "DECLARED", Adapter: npmAdapter}
	var note string
	switch {
	case len(local) > 1:
		edge.Resolution, edge.TargetScope = "UNRESOLVED", "UNRESOLVED_TARGET"
		note = "Local npm package identity is ambiguous for declaration in " + section
	case len(local) == 1 && local[0] != file:
		edge.Resolution, edge.TargetScope, edge.ResolvedPath = "RESOLVED", "PROJECT_COMPONENT", &local[0]
		note = "Local npm package dependency declared in " + section
	default:
		edge.Resolution, edge.TargetScope = "EXTERNAL", "EXTERNAL_DEPENDENCY"
		note = "External npm package dependency declared in " + section
	}
	edge.Note = &note
	return edge
}

func nearestNPMManifest(file string, manifests map[string]npmManifest, files map[string]bool) npmManifest {
	for directory := path.Dir(file); ; directory = path.Dir(directory) {
		candidate := path.Join(directory, "package.json")
		if files[candidate] || directory == "." {
			return manifests[candidate]
		}
	}
}

func jsPhase(file string) string {
	file = "/" + strings.ToLower(file)
	for _, token := range []string{"/test/", "/tests/", "/__tests__/", ".test.", ".spec."} {
		if strings.Contains(file, token) {
			return "TEST"
		}
	}
	return "UNKNOWN"
}

func resolveJSDependency(edge *DependencyEdge, c dependencyContext, names map[string][]string, nearest npmManifest) {
	edge.Resolution, edge.TargetScope = "UNRESOLVED", "UNRESOLVED_TARGET"
	target := edge.Target
	var note string
	if strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") {
		resolved, ambiguous := resolveJSRelative(edge.Source, target, c)
		switch {
		case resolved != "":
			edge.Resolution, edge.TargetScope, edge.ResolvedPath = "RESOLVED", "PROJECT_COMPONENT", &resolved
			return
		case ambiguous:
			note = "Relative JavaScript or TypeScript import has multiple inventoried targets"
		default:
			note = "Relative JavaScript or TypeScript import target is outside or absent from the source inventory"
		}
	} else {
		name, rest, _ := strings.Cut(target, "/")
		if strings.HasPrefix(name, "@") && rest != "" {
			local, _, _ := strings.Cut(rest, "/")
			name += "/" + local
		}
		switch local := names[name]; len(local) {
		case 1:
			edge.Resolution, edge.TargetScope, edge.ResolvedPath = "RESOLVED", "PROJECT_COMPONENT", &local[0]
			note = "Target resolves to a local npm package manifest"
		case 0:
			note = "Bare package import is not declared in the nearest package.json"
			for _, section := range npmSections {
				dependencies := nearest.sections[section]
				index := sort.SearchStrings(dependencies, name)
				if index < len(dependencies) && dependencies[index] == name {
					edge.Resolution, edge.TargetScope = "EXTERNAL", "EXTERNAL_DEPENDENCY"
					if edge.Phase == "UNKNOWN" {
						edge.Phase = npmPhase(section)
					}
					note = "Target matches nearest npm " + section + " declaration"
					break
				}
			}
		default:
			note = "Bare package import matches multiple local npm package manifests"
		}
	}
	edge.Note = &note
}

func resolveJSRelative(source, specifier string, c dependencyContext) (string, bool) {
	target := path.Join(path.Dir(source), specifier)
	if target == ".." || strings.HasPrefix(target, "../") || strings.ContainsAny(target, "\\\x00") {
		return "", false
	}
	if c.exists(target) {
		return target, false
	}
	var resolved string
	for _, extension := range jsExtensions {
		for _, candidate := range []string{target + extension, path.Join(target, "index"+extension)} {
			if c.exists(candidate) {
				if resolved != "" && resolved != candidate {
					return "", true
				}
				resolved = candidate
			}
		}
	}
	return resolved, false
}

type jsToken struct {
	text string
	line int
	kind byte
}

type jsLexer struct {
	text    string
	index   int
	line    int
	tokens  []jsToken
	problem string
	jsx     bool
}

func (l *jsLexer) fail(message string) {
	if l.problem == "" {
		l.problem = fmt.Sprintf("line %d: %s", l.line, message)
	}
}

func (l *jsLexer) emit(text string, line int, kind byte) {
	if len(l.tokens) >= 256<<10 {
		l.fail("source exceeds the dependency token limit")
		return
	}
	l.tokens = append(l.tokens, jsToken{text: text, line: line, kind: kind})
}

func jsIdentifier(char byte) bool {
	return char >= utf8.RuneSelf || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' ||
		char >= '0' && char <= '9' || char == '_' || char == '$'
}

func (l *jsLexer) code(expression bool, depth int) {
	if depth > 64 {
		l.fail("source exceeds the dependency expression nesting limit")
		return
	}
	braces := 0
	var controlParens []bool
	for l.index < len(l.text) && l.problem == "" {
		start := l.index
		char := l.text[start]
		l.index++
		switch char {
		case '\n':
			l.line++
		case ' ', '\t', '\r', '\f', '\v':
		case '#':
			if start == 0 && strings.HasPrefix(l.text[l.index:], "!") {
				l.lineComment()
			} else {
				l.emit("#", l.line, 'p')
			}
		case '/':
			switch {
			case strings.HasPrefix(l.text[l.index:], "/"):
				l.lineComment()
			case strings.HasPrefix(l.text[l.index:], "*"):
				end := strings.Index(l.text[l.index+1:], "*/")
				if end < 0 {
					l.fail("unterminated block comment")
					break
				}
				end += l.index + 3
				l.line += strings.Count(l.text[l.index:end], "\n")
				l.index = end
			case l.expressionStart() && l.regexLiteral():
				l.emit("", l.line, 'x')
			default:
				l.emit("/", l.line, 'p')
			}
		case '\'', '"':
			l.quote(char)
		case '`':
			l.emit("", l.line, 'x')
			l.template(depth + 1)
			l.emit("", l.line, 'x')
		case '<':
			if l.jsx && l.expressionStart() && l.jsxTagStart() {
				l.emit("", l.line, 'x')
				l.jsxElement(depth + 1)
				l.emit("", l.line, 'x')
			} else {
				l.emit("<", l.line, 'p')
			}
		case '(':
			controlParens = append(controlParens, l.precedesControlParen())
			l.emit("(", l.line, 'p')
		case ')':
			kind := byte('p')
			if last := len(controlParens) - 1; last >= 0 {
				if controlParens[last] {
					kind = 'c'
				}
				controlParens = controlParens[:last]
			}
			l.emit(")", l.line, kind)
		case '=', '+', '-':
			if l.index < len(l.text) && (char == '=' && l.text[l.index] == '>' || char != '=' && l.text[l.index] == char) {
				l.index++
			}
			l.emit(l.text[start:l.index], l.line, 'p')
		case '{':
			braces++
			l.emit("{", l.line, 'p')
		case '}':
			l.emit("}", l.line, 'p')
			if expression && braces == 0 {
				return
			}
			if braces > 0 {
				braces--
			}
		default:
			if jsIdentifier(char) {
				for l.index < len(l.text) && jsIdentifier(l.text[l.index]) {
					l.index++
				}
				l.emit(l.text[start:l.index], l.line, 'i')
			} else {
				l.emit(l.text[start:l.index], l.line, 'p')
			}
		}
	}
	if expression {
		l.fail("unterminated embedded JavaScript expression")
	}
}

func (l *jsLexer) lineComment() {
	if end := strings.IndexByte(l.text[l.index:], '\n'); end >= 0 {
		l.index += end
	} else {
		l.index = len(l.text)
	}
}

func (l *jsLexer) quote(quote byte) {
	start, line := l.index, l.line
	kind := byte('s')
	for l.index < len(l.text) {
		char := l.text[l.index]
		l.index++
		switch char {
		case quote:
			l.emit(l.text[start:l.index-1], line, kind)
			return
		case '\r', '\n':
			l.fail("unterminated string literal")
			return
		case '\\':
			kind = 'x'
			if l.index < len(l.text) {
				if strings.HasPrefix(l.text[l.index:], "\r\n") {
					l.index++
				}
				if l.text[l.index] == '\n' {
					l.line++
				}
				l.index++
			}
		}
	}
	l.fail("unterminated string literal")
}

func (l *jsLexer) expressionStart() bool {
	if len(l.tokens) == 0 {
		return true
	}
	previous := l.tokens[len(l.tokens)-1]
	if previous.kind == 'c' {
		return true
	}
	if previous.kind == 'p' {
		return previous.text == "=>" || strings.Contains("([{=,:;!~?&|+-*%^", previous.text)
	}
	if previous.kind == 'i' {
		switch previous.text {
		case "return", "throw", "case", "yield", "await", "typeof", "delete", "void":
			return true
		}
	}
	return false
}

func (l *jsLexer) precedesControlParen() bool {
	if len(l.tokens) == 0 {
		return false
	}
	previous := l.tokens[len(l.tokens)-1]
	if previous.kind != 'i' {
		return false
	}
	switch previous.text {
	case "if", "while", "for", "with", "switch", "catch":
		return true
	default:
		return false
	}
}

func (l *jsLexer) regexLiteral() bool {
	inClass := false
	for index := l.index; index < len(l.text); index++ {
		switch l.text[index] {
		case '\r', '\n':
			return false
		case '\\':
			index++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				l.index = index + 1
				for l.index < len(l.text) && jsIdentifier(l.text[l.index]) {
					l.index++
				}
				return true
			}
		}
	}
	return false
}

func (l *jsLexer) template(depth int) {
	for l.index < len(l.text) && l.problem == "" {
		char := l.text[l.index]
		l.index++
		switch char {
		case '`':
			return
		case '\n':
			l.line++
		case '\\':
			if l.index < len(l.text) {
				if l.text[l.index] == '\n' {
					l.line++
				}
				l.index++
			}
		case '$':
			if strings.HasPrefix(l.text[l.index:], "{") {
				l.index++
				l.emit("{", l.line, 'p')
				l.code(true, depth)
			}
		}
	}
	l.fail("unterminated template literal")
}

func (l *jsLexer) jsxTagStart() bool {
	if l.index >= len(l.text) {
		return false
	}
	char := l.text[l.index]
	if char == '>' {
		return true
	}
	if !jsIdentifier(char) || char >= '0' && char <= '9' {
		return false
	}
	// TSX prefers a comma or extends-clause for generic arrow functions.
	end := l.index + 1
	for end < len(l.text) && jsIdentifier(l.text[end]) {
		end++
	}
	tail := strings.TrimLeft(l.text[end:], " \t\r\n")
	return !strings.HasPrefix(tail, ",") && !strings.HasPrefix(tail, "extends ")
}

func (l *jsLexer) jsxElement(depth int) {
	if depth > 64 {
		l.fail("source exceeds the JSX nesting limit")
		return
	}
	closedTag := false
	for l.index < len(l.text) && l.problem == "" && !closedTag {
		char := l.text[l.index]
		l.index++
		switch char {
		case '>':
			closedTag = true
		case '/':
			if strings.HasPrefix(l.text[l.index:], ">") {
				l.index++
				return
			}
		case '\n':
			l.line++
		case '\'', '"':
			end := strings.IndexByte(l.text[l.index:], char)
			if end < 0 {
				l.fail("unterminated JSX attribute")
				break
			}
			l.line += strings.Count(l.text[l.index:l.index+end], "\n")
			l.index += end + 1
		case '{':
			l.emit("{", l.line, 'p')
			l.code(true, depth)
		}
	}
	for l.index < len(l.text) && l.problem == "" {
		char := l.text[l.index]
		l.index++
		switch char {
		case '\n':
			l.line++
		case '{':
			l.emit("{", l.line, 'p')
			l.code(true, depth)
		case '<':
			if strings.HasPrefix(l.text[l.index:], "/") {
				end := strings.IndexByte(l.text[l.index:], '>')
				if end < 0 {
					l.fail("unterminated JSX closing tag")
					return
				}
				l.line += strings.Count(l.text[l.index:l.index+end], "\n")
				l.index += end + 1
				return
			}
			l.jsxElement(depth + 1)
		}
	}
	l.fail("unterminated JSX element")
}

func jsDependency(tokens []jsToken, index int) (target, kind string, dynamic bool, problem string) {
	keyword := tokens[index]
	if keyword.kind != 'i' || index+1 >= len(tokens) {
		return "", "", false, ""
	}
	if index > 0 {
		previous := tokens[index-1]
		if previous.kind == 'p' && (previous.text == "." || previous.text == "#") || previous.kind == 'i' && previous.text == "function" {
			return "", "", false, ""
		}
	}
	next := tokens[index+1]
	switch keyword.text {
	case "import", "require":
		if next.kind == 'p' && next.text == "(" {
			kind := "LOADS"
			if keyword.text == "require" {
				kind = "IMPORTS"
			}
			if index+3 < len(tokens) && tokens[index+2].kind == 's' && tokens[index+2].text != "" &&
				tokens[index+3].kind == 'p' && tokens[index+3].text == ")" {
				return tokens[index+2].text, kind, false, ""
			}
			return "<dynamic-import>", "LOADS", true, ""
		}
		if keyword.text == "require" {
			return "", "", false, ""
		}
	case "export":
		if next.kind == 'i' && next.text == "type" && index+2 < len(tokens) {
			next = tokens[index+2]
		}
		if next.kind != 'p' || (next.text != "{" && next.text != "*") {
			return "", "", false, ""
		}
	default:
		return "", "", false, ""
	}
	if next.kind == 's' && next.text != "" {
		return next.text, "IMPORTS", false, ""
	}
	if next.kind == 'x' {
		return "", "", false, "static dependency specifier cannot be read as a plain string literal"
	}
	// A top-level from can follow a multiline binding list of identifiers.
	braces := 0
	for at := index + 1; at+1 < len(tokens); at++ {
		if at-index > 1024 {
			return "", "", false, "dependency statement exceeds the supported token length"
		}
		token := tokens[at]
		if token.kind == 'p' {
			switch token.text {
			case "{":
				braces++
			case "}":
				if braces == 0 {
					return "", "", false, ""
				}
				braces--
			case ".", "=", "(", ";":
				return "", "", false, ""
			}
		}
		if token.kind == 'i' && braces == 0 {
			switch token.text {
			case "import", "export", "const", "let", "var", "return":
				return "", "", false, ""
			case "from":
				specifier := tokens[at+1]
				if specifier.kind == 's' && specifier.text != "" {
					return specifier.text, "IMPORTS", false, ""
				}
				if specifier.kind == 'x' {
					return "", "", false, "static dependency specifier cannot be read as a plain string literal"
				}
				return "", "", false, ""
			}
		}
	}
	return "", "", false, ""
}
