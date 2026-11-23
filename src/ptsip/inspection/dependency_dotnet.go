package inspection

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const (
	dotnetManifestAdapter = "dotnet-csproj"
	dotnetSourceAdapter   = "dotnet-source"
)

type dotnetProject struct {
	file, directory, namespace string
	packages                   []string
	references                 map[string]bool
}

func scanDotNet(c dependencyContext) ([]DependencyEdge, []DependencyIssue, []string) {
	var projects []dotnetProject
	var sources, adapters []string
	var edges []DependencyEdge
	var issues []DependencyIssue
	foundManifest := false
	for _, file := range c.paths {
		switch strings.ToLower(path.Ext(file)) {
		case ".csproj":
			foundManifest = true
			project, foundEdges, foundIssues := readDotNetProject(c, file)
			if project.file != "" {
				projects = append(projects, project)
			}
			edges = append(edges, foundEdges...)
			issues = append(issues, foundIssues...)
		case ".cs":
			sources = append(sources, file)
		}
	}
	if foundManifest {
		adapters = append(adapters, dotnetManifestAdapter)
	}
	if foundManifest || len(sources) != 0 {
		adapters = append(adapters, dotnetSourceAdapter)
	}
	for _, file := range sources {
		foundEdges, foundIssues := scanCSharpSource(c, file, projects)
		edges, issues = append(edges, foundEdges...), append(issues, foundIssues...)
	}
	return edges, issues, adapters
}

func readDotNetProject(c dependencyContext, file string) (dotnetProject, []DependencyEdge, []DependencyIssue) {
	invalid := func(err error) (dotnetProject, []DependencyEdge, []DependencyIssue) {
		return dotnetProject{}, nil, []DependencyIssue{{Adapter: dotnetManifestAdapter, Path: file, Message: err.Error()}}
	}
	body, err := c.read(file)
	if err != nil {
		return invalid(err)
	}
	decoder, err := dotnetXML(body)
	if err != nil {
		return invalid(err)
	}
	project := dotnetProject{file: file, directory: path.Dir(file), references: make(map[string]bool)}
	var edges []DependencyEdge
	var issues []DependencyIssue
	var ancestors []struct {
		name        string
		conditional bool
	}
	var rootNamespaces, assemblyNames []string
	seenRoot := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return invalid(err)
		}
		switch element := token.(type) {
		case xml.Directive:
			return invalid(fmt.Errorf("XML document directives are not evaluated"))
		case xml.CharData:
			if len(ancestors) == 0 && len(bytes.TrimSpace(element)) != 0 {
				return invalid(fmt.Errorf("unexpected text outside the MSBuild Project root"))
			}
		case xml.EndElement:
			ancestors = ancestors[:len(ancestors)-1]
		case xml.StartElement:
			conditional := dotnetAttr(element, "Condition") != ""
			if len(ancestors) == 0 {
				if seenRoot || element.Name.Local != "Project" {
					return invalid(fmt.Errorf("expected one MSBuild Project root"))
				}
				seenRoot = true
			}
			var parent string
			if len(ancestors) > 0 {
				ancestor := ancestors[len(ancestors)-1]
				parent, conditional = ancestor.name, conditional || ancestor.conditional
			}
			var diagnostic string
			switch element.Name.Local {
			case "Choose", "When", "Otherwise", "Target":
				conditional = true
			case "Import", "ImportGroup":
				diagnostic = "MSBuild imports are not evaluated"
			case "RootNamespace", "AssemblyName":
				if parent == "PropertyGroup" {
					var value string
					if err := decoder.DecodeElement(&value, &element); err != nil {
						return invalid(err)
					}
					value = strings.TrimSpace(value)
					if value == "" && !conditional {
						continue
					}
					if conditional || dotnetDynamic(value) {
						// An unknown override can change root namespace identity.
						value = ""
						issues = append(issues, DependencyIssue{Adapter: dotnetManifestAdapter, Path: file, Message: element.Name.Local + " depends on MSBuild evaluation"})
					}
					if element.Name.Local == "RootNamespace" {
						rootNamespaces = append(rootNamespaces, value)
					} else {
						assemblyNames = append(assemblyNames, value)
					}
					continue
				}
			case "PackageReference":
				name := dotnetAttr(element, "Include")
				if name == "" {
					name = dotnetAttr(element, "Update")
				}
				if name != "" {
					if conditional || dotnetDynamic(name) {
						diagnostic = fmt.Sprintf("PackageReference %q depends on MSBuild evaluation", name)
					} else {
						project.packages = append(project.packages, name)
					}
				}
			case "ProjectReference":
				include := dotnetAttr(element, "Include")
				if include != "" {
					var edge DependencyEdge
					edge, diagnostic = dotnetProjectReference(c, project, include, conditional)
					edges = append(edges, edge)
				}
			}
			if diagnostic != "" {
				issues = append(issues, DependencyIssue{Adapter: dotnetManifestAdapter, Path: file, Message: diagnostic})
			}
			ancestors = append(ancestors, struct {
				name        string
				conditional bool
			}{element.Name.Local, conditional})
		}
	}
	if !seenRoot {
		return invalid(fmt.Errorf("missing MSBuild Project root"))
	}
	switch {
	case len(rootNamespaces) > 0:
		project.namespace = uniqueDotnetProperty(rootNamespaces)
	case len(assemblyNames) > 0:
		project.namespace = uniqueDotnetProperty(assemblyNames)
	default:
		project.namespace = strings.TrimSuffix(path.Base(file), path.Ext(file))
	}
	if project.namespace == "" {
		issues = append(issues, DependencyIssue{Adapter: dotnetManifestAdapter, Path: file, Message: "root namespace is ambiguous"})
	}
	return project, edges, issues
}

func dotnetXML(body []byte) (*xml.Decoder, error) {
	utf16 := bytes.HasPrefix(body, []byte{0xff, 0xfe}) || bytes.HasPrefix(body, []byte{0xfe, 0xff})
	if utf16 {
		var err error
		body, _, err = transform.Bytes(unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder(), body)
		if err != nil {
			return nil, err
		}
	}
	body = bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf})
	decoder := xml.NewDecoder(bytes.NewReader(body))
	if utf16 {
		decoder.CharsetReader = func(charset string, stream io.Reader) (io.Reader, error) {
			switch strings.ToLower(charset) {
			case "utf-16", "utf-16le", "utf-16be":
				return stream, nil
			default:
				return nil, fmt.Errorf("unsupported MSBuild XML encoding %q", charset)
			}
		}
	}
	return decoder, nil
}

func dotnetAttr(element xml.StartElement, name string) string {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == name && attribute.Name.Space == "" {
			return strings.TrimSpace(attribute.Value)
		}
	}
	return ""
}

func dotnetDynamic(value string) bool {
	return strings.ContainsAny(value, "*?;\x00") || strings.Contains(value, "$(") || strings.Contains(value, "@(") || strings.Contains(value, "%(")
}

func uniqueDotnetProperty(values []string) string {
	for _, value := range values[1:] {
		if values[0] != value {
			return ""
		}
	}
	return values[0]
}

func dotnetProjectReference(c dependencyContext, project dotnetProject, include string, conditional bool) (DependencyEdge, string) {
	target := strings.ReplaceAll(include, "\\", "/")
	edge := DependencyEdge{EvidenceID: "dotnet-project-reference:" + project.file + ":" + include,
		Source: project.file, Target: target, EdgeType: "LINKS", Phase: "UNKNOWN", Adapter: dotnetManifestAdapter,
		Resolution: "UNRESOLVED", TargetScope: "UNRESOLVED_TARGET", Provenance: "DECLARED"}
	file := path.Join(project.directory, target)
	var note string
	switch {
	case conditional || dotnetDynamic(target):
		note = fmt.Sprintf("ProjectReference %q depends on MSBuild evaluation", include)
	case strings.HasPrefix(target, "/") || strings.Contains(target, ":") || !fs.ValidPath(file) || file == ".":
		note = fmt.Sprintf("ProjectReference %q is outside the observed repository", include)
	case !c.files[file]:
		note = fmt.Sprintf("ProjectReference %q is absent from the observed repository", include)
	case !c.exists(file):
		note = fmt.Sprintf("ProjectReference %q is not an available regular observed file", include)
	default:
		edge.Resolution, edge.TargetScope, edge.ResolvedPath = "RESOLVED", "PROJECT_COMPONENT", &file
		project.references[file] = true
	}
	if note != "" {
		edge.Note = &note
	}
	return edge, note
}

// The anchored expression handles ordinary, global, static, and aliased using
// declarations. Resource-using statements are outside this observation.
var csharpUsing = regexp.MustCompile(`\b(?:global[\t ]+)?using[\t \r\n]+(?:static[\t \r\n]+)?(?:@?[A-Za-z_][A-Za-z_0-9]*[\t \r\n]*=[\t \r\n]*)?(?:global::)?(@?[A-Za-z_][A-Za-z_0-9]*(?:[\t \r\n]*\.[\t \r\n]*@?[A-Za-z_][A-Za-z_0-9]*)*)[\t \r\n]*;`)
var csharpUsingStart = regexp.MustCompile(`(?m)^[\t ]*(?:global[\t ]+)?using(?:[\t ]+|$)`)
var csharpResourceUsing = regexp.MustCompile(`^(?:\(|var[\t ]|[A-Za-z_][\w.<>?]*[\t ]+[A-Za-z_][\w]*[\t ]*=)`)

func scanCSharpSource(c dependencyContext, file string, projects []dotnetProject) ([]DependencyEdge, []DependencyIssue) {
	var edges []DependencyEdge
	var issues []DependencyIssue
	addIssue := func(message string) {
		issues = append(issues, DependencyIssue{Adapter: dotnetSourceAdapter, Path: file, Message: message})
	}
	body, err := c.read(file)
	if err != nil {
		addIssue(err.Error())
		return edges, issues
	}
	if !utf8.Valid(body) {
		addIssue("C# source is not valid UTF-8")
		return edges, issues
	}
	body = bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf})
	code, err := maskCSharpLiterals(body)
	if err != nil {
		addIssue(err.Error())
	}
	project, ambiguous := nearestDotNetProject(file, projects)
	if ambiguous {
		addIssue("multiple .NET project files claim the nearest source directory")
	}
	conditions, err := csharpConditions(code)
	if err != nil {
		addIssue(err.Error())
	}
	matched := make(map[int]bool)
	last, line := 0, 1
	for _, found := range csharpUsing.FindAllSubmatchIndex(code, -1) {
		lineStart := bytes.LastIndexByte(code[:found[0]], '\n') + 1
		if found[0] > 0 && code[found[0]-1] == '@' || bytes.HasPrefix(bytes.TrimSpace(code[lineStart:found[0]]), []byte{'#'}) {
			continue
		}
		line += bytes.Count(code[last:found[0]], []byte{'\n'})
		last = found[0]
		matched[line] = true
		namespace := strings.ReplaceAll(strings.Join(strings.Fields(string(code[found[2]:found[3]])), ""), "@", "")
		phase := "RUNTIME"
		lower := "/" + strings.ToLower(file)
		for _, token := range []string{"/test/", "/tests/", ".tests/", ".test/"} {
			if strings.Contains(lower, token) {
				phase = "TEST"
				break
			}
		}
		if conditions[line] {
			phase = "UNKNOWN"
			addIssue(fmt.Sprintf("line %d: using declaration is inside an unevaluated preprocessor branch", line))
		}
		edge, diagnostic := dotnetUsingEdge(file, line, namespace, phase, project, projects)
		edges = append(edges, edge)
		if diagnostic != "" {
			addIssue(diagnostic)
		}
	}
	last, line = 0, 1
	for _, start := range csharpUsingStart.FindAllIndex(code, -1) {
		line += bytes.Count(code[last:start[0]], []byte{'\n'})
		last = start[0]
		if matched[line] {
			continue
		}
		first, _, _ := bytes.Cut(code[start[1]:], []byte{'\n'})
		if !csharpResourceUsing.Match(bytes.TrimSpace(first)) {
			addIssue(fmt.Sprintf("line %d: unsupported or incomplete C# using declaration", line))
		}
	}
	return edges, issues
}

func nearestDotNetProject(file string, projects []dotnetProject) (*dotnetProject, bool) {
	var match *dotnetProject
	ambiguous := false
	for i := range projects {
		project := &projects[i]
		if project.directory != "." && !strings.HasPrefix(file, project.directory+"/") {
			continue
		}
		if match == nil || len(project.directory) > len(match.directory) {
			match, ambiguous = project, false
		} else if match.directory == project.directory {
			ambiguous = true
		}
	}
	if ambiguous {
		return nil, true
	}
	return match, false
}

func dotnetUsingEdge(file string, line int, namespace, phase string, origin *dotnetProject, projects []dotnetProject) (DependencyEdge, string) {
	edge := DependencyEdge{EvidenceID: fmt.Sprintf("dotnet-source:%s:%d:%s", file, line, namespace), Source: file,
		Target: namespace, EdgeType: "IMPORTS", Phase: phase, Adapter: dotnetSourceAdapter, Provenance: "OBSERVED",
		Resolution: "UNRESOLVED", TargetScope: "UNRESOLVED_TARGET", Line: &line}
	var local *dotnetProject
	namespaceOnly, ambiguous := false, false
	for i := range projects {
		project := &projects[i]
		if project == origin || project.namespace == "" || !dotnetNamespaceMatches(namespace, project.namespace) {
			continue
		}
		namespaceOnly = true
		if origin != nil && origin.references[project.file] {
			if local == nil || len(project.namespace) > len(local.namespace) {
				local, ambiguous = project, false
			} else if len(project.namespace) == len(local.namespace) {
				ambiguous = true
			}
		}
	}
	var packageMatches []string
	if origin != nil {
		normalized := strings.ReplaceAll(strings.ToLower(namespace), "_", ".")
		for _, pkg := range origin.packages {
			if !dotnetNamespaceMatches(normalized, strings.ReplaceAll(strings.ToLower(pkg), "-", ".")) {
				continue
			}
			if len(packageMatches) == 0 || len(pkg) > len(packageMatches[0]) {
				packageMatches = []string{pkg}
			} else if len(pkg) == len(packageMatches[0]) && !slices.Contains(packageMatches, pkg) {
				packageMatches = append(packageMatches, pkg)
			}
		}
	}
	var note, diagnostic string
	switch {
	case ambiguous:
		note = "namespace matches multiple directly referenced .NET projects"
		diagnostic = fmt.Sprintf("line %d: %s for %q", line, note, namespace)
	case local != nil:
		edge.Resolution, edge.TargetScope, edge.ResolvedPath = "RESOLVED", "PROJECT_COMPONENT", &local.file
		note = fmt.Sprintf("namespace matches referenced .NET project root namespace %q", local.namespace)
	case dotnetNamespaceMatches(namespace, "System"):
		edge.Resolution, edge.TargetScope = "EXTERNAL", "PLATFORM"
		note = ".NET platform namespace"
	case len(packageMatches) > 1 || len(packageMatches) != 0 && namespaceOnly:
		note = "namespace has ambiguous local project or PackageReference identity"
		diagnostic = fmt.Sprintf("line %d: %s for %q", line, note, namespace)
	case len(packageMatches) == 1:
		edge.Resolution, edge.TargetScope = "EXTERNAL", "EXTERNAL_DEPENDENCY"
		note = fmt.Sprintf("namespace deterministically matches PackageReference %q", packageMatches[0])
	case namespaceOnly:
		note = "namespace matches a local project without an observed ProjectReference"
	default:
		note = "namespace is not attributable to a local project, platform namespace or PackageReference"
	}
	edge.Note = &note
	return edge, diagnostic
}

func dotnetNamespaceMatches(namespace, prefix string) bool {
	return namespace == prefix || strings.HasPrefix(namespace, prefix+".")
}

// Blank comments and literal text without changing byte offsets or line breaks.
func maskCSharpLiterals(source []byte) ([]byte, error) {
	code := bytes.Clone(source)
	for i := 0; i < len(source); {
		end, complete := csharpNonCode(source, i, 0)
		if end == i {
			i++
			continue
		}
		for j := i; j < end; j++ {
			if code[j] != '\n' && code[j] != '\r' {
				code[j] = ' '
			}
		}
		if !complete {
			return code, fmt.Errorf("line %d: unterminated or unrecognized C# comment or literal", 1+bytes.Count(source[:i], []byte{'\n'}))
		}
		i = end
	}
	return code, nil
}

func csharpNonCode(source []byte, offset, depth int) (int, bool) {
	rest := source[offset:]
	if bytes.HasPrefix(rest, []byte("//")) {
		if end := bytes.IndexByte(rest, '\n'); end >= 0 {
			return offset + end, true
		}
		return len(source), true
	}
	if bytes.HasPrefix(rest, []byte("/*")) {
		if end := bytes.Index(rest[2:], []byte("*/")); end >= 0 {
			return offset + end + 4, true
		}
		return len(source), false
	}
	quote, interpolated, verbatim := offset, false, false
	for quote < len(source) && (source[quote] == '$' || source[quote] == '@') {
		switch source[quote] {
		case '$':
			interpolated = true
		case '@':
			if verbatim {
				return offset, true
			}
			verbatim = true
		}
		quote++
	}
	if quote >= len(source) || (source[quote] != '"' && (source[quote] != '\'' || quote != offset)) {
		return offset, true
	}
	if depth >= 32 {
		return len(source), false
	}
	delimiter := source[quote]
	size := 1
	for quote+size < len(source) && source[quote+size] == delimiter {
		size++
	}
	if delimiter == '"' && !verbatim && size >= 3 {
		raw := source[quote+size:]
		if end := bytes.Index(raw, source[quote:quote+size]); end >= 0 {
			return quote + 2*size + end, true
		}
		return len(source), false
	}
	braces := 0
	for i := quote + 1; i < len(source); i++ {
		ch := source[i]
		if braces > 0 {
			if end, complete := csharpNonCode(source, i, depth+1); end != i {
				if !complete {
					return len(source), false
				}
				i = end - 1
				continue
			}
			switch ch {
			case '{':
				braces++
			case '}':
				braces--
			}
			continue
		}
		switch {
		case ch == delimiter:
			if verbatim && i+1 < len(source) && source[i+1] == delimiter {
				i++
				continue
			}
			return i + 1, true
		case ch == '\\' && !verbatim:
			i++
		case interpolated && ch == '{':
			if i+1 < len(source) && source[i+1] == '{' {
				i++
			} else {
				braces = 1
			}
		case ch == '\n' && !verbatim:
			return len(source), false
		}
	}
	return len(source), false
}

func csharpConditions(source []byte) (map[int]bool, error) {
	conditional := make(map[int]bool)
	depth := 0
	for index, line := range bytes.Split(source, []byte{'\n'}) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] == '#' {
			parts := bytes.Fields(trimmed[1:])
			if len(parts) != 0 {
				switch string(parts[0]) {
				case "if":
					depth++
				case "endif":
					if depth == 0 {
						return conditional, fmt.Errorf("line %d: unmatched C# #endif", index+1)
					}
					depth--
				case "elif", "else":
					if depth == 0 {
						return conditional, fmt.Errorf("line %d: unmatched C# #%s", index+1, parts[0])
					}
				}
			}
		}
		if depth > 0 {
			conditional[index+1] = true
		}
	}
	if depth != 0 {
		return conditional, fmt.Errorf("unterminated C# preprocessor condition")
	}
	return conditional, nil
}
