package inspection

import (
	"bytes"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	gts "github.com/odvcencio/gotreesitter"
	grammar "github.com/odvcencio/gotreesitter/grammars/python"
	"github.com/pelletier/go-toml/v2"
)

const pythonAdapter = "python"

type pythonResolver struct {
	files, directories, declared map[string]bool
	roots                        map[string]bool
}

func scanPython(c dependencyContext) ([]DependencyEdge, []DependencyIssue, []string) {
	var sources []string
	resolver := pythonResolver{files: map[string]bool{}, directories: map[string]bool{}, declared: map[string]bool{}, roots: map[string]bool{".": true}}
	for _, file := range c.paths {
		if !strings.EqualFold(path.Ext(file), ".py") {
			continue
		}
		sources = append(sources, file)
		if !c.exists(file) {
			continue
		}
		resolver.files[file] = true
		for directory := path.Dir(file); directory != "." && !resolver.directories[directory]; directory = path.Dir(directory) {
			resolver.directories[directory] = true
		}
		if strings.HasPrefix(file, "src/") {
			resolver.roots["src"] = true
		}
	}
	if len(sources) == 0 {
		return nil, nil, nil
	}
	issues := resolver.declarations(c)
	lang := grammar.Language()
	parser := gts.NewParser(lang)
	parser.SetTimeoutMicros(2_000_000)
	var edges []DependencyEdge
	for _, file := range sources {
		observed, err := scanPythonFile(c, file, resolver, lang, parser)
		if err != nil {
			issues = append(issues, DependencyIssue{Adapter: pythonAdapter, Path: file, Message: err.Error()})
			continue
		}
		edges = append(edges, observed...)
	}
	return edges, issues, []string{pythonAdapter}
}

var pythonRequirement = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]*)(?:[\t \[!<>=~;@]|$)`)

func normalizePythonDistribution(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_"), ".", "_")
}

func (r pythonResolver) declarations(c dependencyContext) []DependencyIssue {
	var issues []DependencyIssue
	for _, file := range c.paths {
		problem := func(message string) {
			issues = append(issues, DependencyIssue{Adapter: "python-declarations", Path: file, Message: message})
		}
		switch {
		case path.Base(file) == "pyproject.toml":
			data, err := c.read(file)
			if err != nil {
				problem(err.Error())
				continue
			}
			var declaration struct {
				Project struct {
					Dependencies []string            `toml:"dependencies"`
					Optional     map[string][]string `toml:"optional-dependencies"`
					Dynamic      []string            `toml:"dynamic"`
				} `toml:"project"`
				Tool struct {
					Setuptools struct {
						PackageDir map[string]string `toml:"package-dir"`
						Packages   any               `toml:"packages"`
					} `toml:"setuptools"`
					Poetry map[string]any `toml:"poetry"`
				} `toml:"tool"`
			}
			if err := toml.Unmarshal(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), &declaration); err != nil {
				problem(err.Error())
				continue
			}
			if slices.Contains(declaration.Project.Dynamic, "dependencies") || slices.Contains(declaration.Project.Dynamic, "optional-dependencies") || declaration.Tool.Poetry["dependencies"] != nil {
				problem("dynamic or non-PEP-621 dependencies are not evaluated")
			}
			if file == "pyproject.toml" {
				dependencies := slices.Clone(declaration.Project.Dependencies)
				for _, name := range slices.Sorted(maps.Keys(declaration.Project.Optional)) {
					dependencies = append(dependencies, declaration.Project.Optional[name]...)
				}
				for _, dependency := range dependencies {
					if match := pythonRequirement.FindStringSubmatch(strings.TrimSpace(dependency)); match != nil {
						r.declared[normalizePythonDistribution(match[1])] = true
					} else {
						problem("unsupported Python dependency declaration: " + dependency)
					}
				}
			}
			addRoot := func(relative string) {
				portable := strings.ReplaceAll(relative, "\\", "/")
				root := path.Join(path.Dir(file), portable)
				if strings.HasPrefix(portable, "/") || strings.ContainsAny(portable, ":\x00") || !fs.ValidPath(root) {
					problem("setuptools source root is outside the observed repository: " + relative)
					return
				}
				r.roots[root] = true
			}
			for _, name := range slices.Sorted(maps.Keys(declaration.Tool.Setuptools.PackageDir)) {
				if name != "" {
					problem("named setuptools package-dir mapping is not resolved: " + name)
					continue
				}
				addRoot(declaration.Tool.Setuptools.PackageDir[name])
			}
			packages, _ := declaration.Tool.Setuptools.Packages.(map[string]any)
			find, _ := packages["find"].(map[string]any)
			where, _ := find["where"].([]any)
			for _, location := range where {
				if relative, ok := location.(string); ok {
					addRoot(relative)
				} else {
					problem("setuptools find.where contains a non-path value")
				}
			}
		case path.Dir(file) == "." && strings.HasPrefix(path.Base(file), "requirements") && (path.Ext(file) == ".txt" || path.Ext(file) == ".in"):
			data, err := c.read(file)
			if err != nil {
				problem(err.Error())
				continue
			}
			if !utf8.Valid(data) {
				problem("Python requirements are not valid UTF-8")
				continue
			}
			for i, line := range strings.Split(strings.TrimPrefix(string(data), "\ufeff"), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if match := pythonRequirement.FindStringSubmatch(line); match != nil {
					r.declared[normalizePythonDistribution(match[1])] = true
				} else {
					problem(fmt.Sprintf("line %d: unsupported Python requirement or directive", i+1))
				}
			}
		case file == "setup.py" || file == "setup.cfg":
			problem("legacy Python packaging dependencies are not evaluated")
		}
	}
	return issues
}

type pythonTarget struct {
	name, file string
	uncertain  bool
	relative   bool
}

func pythonIdentifier(text string) bool {
	if text == "" {
		return false
	}
	for i, char := range text {
		if char != '_' && !unicode.IsLetter(char) && (i == 0 || !unicode.IsDigit(char) && !unicode.IsMark(char)) {
			return false
		}
	}
	return true
}

func (r pythonResolver) packageRoot(file string) (string, bool) {
	parent := path.Dir(file)
	regular := false
	for r.files[path.Join(parent, "__init__.py")] && parent != "." {
		regular = true
		parent = path.Dir(parent)
	}
	if regular {
		return parent, true
	}
	best := ""
	for root := range r.roots {
		if (root == "." || strings.HasPrefix(file, root+"/")) && len(root) > len(best) {
			best = root
		}
	}
	return best, best != "" && path.Dir(file) != best
}

func (r pythonResolver) filesFor(base string) []string {
	result := []string{}
	for _, file := range []string{base + ".py", path.Join(base, "__init__.py")} {
		if r.files[file] {
			result = append(result, file)
		}
	}
	return result
}

func (r pythonResolver) absolute(file, module string) pythonTarget {
	result := pythonTarget{name: module, relative: strings.HasPrefix(module, ".")}
	if result.relative {
		return result
	}
	parts := strings.Split(module, ".")
	for _, part := range parts {
		if !pythonIdentifier(part) {
			return result
		}
	}
	roots := maps.Clone(r.roots)
	if local, ok := r.packageRoot(file); ok {
		roots[local] = true
	}
	candidates := map[string]bool{}
	for root := range roots {
		base := root
		for _, part := range parts {
			base = path.Join(base, part)
			if r.directories[base] || len(r.filesFor(base)) != 0 {
				result.uncertain = true
			}
		}
		for _, candidate := range r.filesFor(base) {
			candidates[candidate] = true
		}
	}
	if len(candidates) == 1 {
		for candidate := range candidates {
			result.file = candidate
		}
		result.uncertain = false
	}
	return result
}

func (r pythonResolver) relative(file, module string, imported []string) []pythonTarget {
	root, ok := r.packageRoot(file)
	parent := path.Dir(file)
	level := len(module) - len(strings.TrimLeft(module, "."))
	tail := module[level:]
	packagePath := parent
	if root != "." {
		packagePath = strings.TrimPrefix(parent, root+"/")
	}
	if !ok || level > len(strings.Split(packagePath, "/")) {
		return []pythonTarget{{name: module, relative: true}}
	}
	for i := 1; i < level; i++ {
		parent = path.Dir(parent)
	}
	nameOf := func(file string) string {
		if root != "." {
			file = strings.TrimPrefix(file, root+"/")
		}
		return strings.ReplaceAll(file, "/", ".")
	}
	if tail != "" {
		base := path.Join(parent, strings.ReplaceAll(tail, ".", "/"))
		return []pythonTarget{selectPythonRelative(nameOf(base), r.filesFor(base))}
	}
	var targets []pythonTarget
	for _, item := range imported {
		base := path.Join(parent, strings.ReplaceAll(item, ".", "/"))
		var candidates []string
		if item != "*" {
			candidates = r.filesFor(base)
		}
		if len(candidates) == 0 && r.files[path.Join(parent, "__init__.py")] {
			targets = append(targets, pythonTarget{name: nameOf(parent), file: path.Join(parent, "__init__.py"), relative: true})
		} else {
			targets = append(targets, selectPythonRelative(nameOf(base), candidates))
		}
	}
	return targets
}

func selectPythonRelative(name string, candidates []string) pythonTarget {
	result := pythonTarget{name: name, relative: true, uncertain: len(candidates) > 1}
	if len(candidates) == 1 {
		result.file = candidates[0]
	}
	return result
}

// Stable public module names shared by CPython 3.11 and 3.14. Module names
// outside this portable catalog need local or declared evidence.
var pythonPlatform = strings.Fields(`__future__ abc argparse array ast asyncio atexit base64 bdb binascii bisect builtins bz2 calendar cmath
cmd code codecs codeop collections colorsys compileall concurrent configparser contextlib contextvars copy copyreg csv ctypes curses
dataclasses datetime dbm decimal difflib dis doctest email encodings ensurepip enum errno faulthandler fcntl filecmp fileinput fnmatch
fractions ftplib functools gc genericpath getopt getpass gettext glob graphlib grp gzip hashlib heapq hmac html http idlelib imaplib
importlib inspect io ipaddress itertools json keyword linecache locale logging lzma mailbox marshal math mimetypes mmap modulefinder
msvcrt multiprocessing netrc nt ntpath numbers opcode operator optparse os pathlib pdb pickle pickletools pkgutil platform plistlib
poplib posix posixpath pprint profile pstats pty pwd py_compile pyclbr pydoc pydoc_data pyexpat queue quopri random re readline
reprlib resource rlcompleter runpy sched secrets select selectors shelve shlex shutil signal site smtplib socket socketserver sqlite3
ssl stat statistics string stringprep struct subprocess symtable sys sysconfig syslog tabnanny tarfile tempfile termios textwrap
threading time timeit tkinter token tokenize tomllib trace traceback tracemalloc tty turtle turtledemo types typing unicodedata
unittest urllib uuid venv warnings wave weakref webbrowser winreg winsound wsgiref xml xmlrpc zipapp zipfile zipimport zlib zoneinfo`)

func (r pythonResolver) edge(file string, line int, target pythonTarget, dynamicKind string) DependencyEdge {
	edge := DependencyEdge{EvidenceID: fmt.Sprintf("python:%s:%d:%s", file, line, target.name), Source: file, Target: target.name,
		EdgeType: "IMPORTS", Phase: "UNKNOWN", Resolution: "UNRESOLVED", TargetScope: "UNRESOLVED_TARGET", Provenance: "OBSERVED", Line: &line, Adapter: pythonAdapter}
	root, _, _ := strings.Cut(target.name, ".")
	var note string
	switch {
	case target.file != "":
		edge.Resolution, edge.TargetScope, edge.ResolvedPath = "RESOLVED", "PROJECT_COMPONENT", &target.file
	case target.relative:
		note = "Relative or ambiguous local target lacks exact tracked evidence"
	case target.uncertain:
		note = "Incomplete or ambiguous tracked local target"
	case slices.Contains(pythonPlatform, root):
		edge.Resolution, edge.TargetScope, note = "EXTERNAL", "PLATFORM", "Python standard-library target"
	case r.declared[normalizePythonDistribution(root)]:
		edge.Resolution, edge.TargetScope, note = "EXTERNAL", "EXTERNAL_DEPENDENCY", "Target matches a declared Python dependency"
	}
	if dynamicKind != "" {
		edge.EvidenceID, edge.EdgeType = fmt.Sprintf("python-dynamic:%s:%d", file, line), "LOADS"
		if dynamicKind == "BOUNDED_DYNAMIC_IMPORT" {
			edge.EvidenceID += ":" + target.name
		}
		if dynamicKind == "UNBOUNDED_DYNAMIC_IMPORT" {
			edge.Resolution = "DYNAMIC"
		}
		if note != "" {
			note = dynamicKind + ": " + note
		} else {
			note = dynamicKind
		}
	}
	if note != "" {
		edge.Note = &note
	}
	return edge
}

var pythonCoding = regexp.MustCompile(`^[\t\f ]*#[^\r\n]*?coding[:=][\t ]*([-_.A-Za-z0-9]+)`)

type pythonImportName struct{ name, alias string }

type pythonTree struct {
	lang    *gts.Language
	body    []byte
	writers map[string][]uint32
}

func scanPythonFile(c dependencyContext, file string, r pythonResolver, lang *gts.Language, parser *gts.Parser) ([]DependencyEdge, error) {
	data, err := c.read(file)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("Python source is not valid UTF-8")
	}
	body := bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	lines := bytes.SplitN(body, []byte{'\n'}, 3)
	for _, line := range lines[:min(2, len(lines))] {
		if match := pythonCoding.FindSubmatch(line); match != nil {
			encoding := strings.ReplaceAll(strings.ToLower(string(match[1])), "_", "-")
			if encoding != "utf-8" && encoding != "utf8" && encoding != "utf-8-sig" && encoding != "ascii" {
				return nil, fmt.Errorf("unsupported Python source encoding %q", match[1])
			}
			if encoding == "ascii" && bytes.IndexFunc(body, func(char rune) bool { return char >= utf8.RuneSelf }) >= 0 {
				return nil, fmt.Errorf("Python source contains non-ASCII text under an ASCII encoding declaration")
			}
		}
	}
	tree, err := parser.Parse(body)
	if tree != nil {
		defer tree.Release()
	}
	if err != nil {
		return nil, fmt.Errorf("parse Python source: %w", err)
	}
	if tree == nil || tree.RootNode() == nil || tree.ParseStopReason() != gts.ParseStopAccepted || tree.RootNode().HasErrorOrMissing() {
		return nil, fmt.Errorf("Python syntax is invalid or the static parse was incomplete")
	}
	state := pythonTree{lang: lang, body: body, writers: map[string][]uint32{}}
	var imports, calls []*gts.Node
	stack := []*gts.Node{tree.RootNode()}
	for len(stack) != 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		switch state.kind(node) {
		case "import_statement", "import_from_statement", "future_import_statement":
			imports = append(imports, node)
		case "call":
			calls = append(calls, node)
		case "assignment", "augmented_assignment", "for_statement", "for_in_clause":
			state.writeNames(state.field(node, "left"))
		case "function_definition", "class_definition", "named_expression":
			state.writeNames(state.field(node, "name"))
		case "as_pattern":
			state.writeNames(state.field(node, "alias"))
		case "parameters", "lambda_parameters", "delete_statement":
			for i := 0; i < node.NamedChildCount(); i++ {
				state.writeNames(node.NamedChild(i))
			}
		}
		for i := node.NamedChildCount() - 1; i >= 0; i-- {
			stack = append(stack, node.NamedChild(i))
		}
	}
	var edges []DependencyEdge
	type aliasDeclaration struct {
		position uint32
		usable   bool
	}
	moduleAliases, importAliases := map[string]aliasDeclaration{}, map[string]aliasDeclaration{}
	wildcardImports := false
	for _, node := range imports {
		line := int(node.StartPoint().Row) + 1
		names := state.importNames(node)
		module := state.importPath(state.field(node, "module_name"))
		if state.kind(node) == "future_import_statement" {
			module = "__future__"
		}
		if state.kind(node) == "import_statement" {
			for _, item := range names {
				target := r.absolute(file, item.name)
				edges = append(edges, r.edge(file, line, target, ""))
				if item.name == "importlib" {
					usable := state.kind(node.Parent()) == "module" && target.file == "" && !target.uncertain
					moduleAliases[item.binding()] = aliasDeclaration{node.StartByte(), usable}
				} else {
					state.writers[item.binding()] = append(state.writers[item.binding()], node.StartByte())
				}
			}
			continue
		}
		if strings.HasPrefix(module, ".") {
			var imported []string
			for _, item := range names {
				imported = append(imported, item.name)
			}
			for _, target := range r.relative(file, module, imported) {
				edges = append(edges, r.edge(file, line, target, ""))
			}
		} else if module != "" {
			edges = append(edges, r.edge(file, line, r.absolute(file, module), ""))
		} else {
			return nil, fmt.Errorf("line %d: unsupported Python import declaration", line)
		}
		for _, item := range names {
			switch {
			case item.name == "*":
				wildcardImports = true
			case module == "importlib" && item.name == "import_module":
				target := r.absolute(file, module)
				usable := state.kind(node.Parent()) == "module" && target.file == "" && !target.uncertain
				importAliases[item.binding()] = aliasDeclaration{node.StartByte(), usable}
			default:
				state.writers[item.binding()] = append(state.writers[item.binding()], node.StartByte())
			}
		}
	}
	usedIDs := map[string]bool{}
	for _, node := range calls {
		function := state.field(node, "function")
		callee := state.text(function)
		builtin := state.kind(function) == "identifier" && callee == "__import__"
		known, usable := builtin, builtin
		switch state.kind(function) {
		case "attribute":
			object := state.field(function, "object")
			if state.kind(object) != "identifier" || state.text(state.field(function, "attribute")) != "import_module" {
				continue
			}
			callee = state.text(object)
			alias, exists := moduleAliases[callee]
			known, usable = exists, alias.usable && alias.position < node.StartByte()
		case "identifier":
			if alias, exists := importAliases[callee]; exists && !builtin {
				known, usable = true, alias.usable && alias.position < node.StartByte()
			}
		}
		if !known {
			continue
		}
		kind := "UNBOUNDED_DYNAMIC_IMPORT"
		var targets []string
		if usable && !wildcardImports && len(state.writers[callee]) == 0 {
			kind, targets = state.dynamicTargets(node, builtin)
		}
		if len(targets) == 0 {
			targets = []string{"<dynamic-import>"}
		}
		line := int(node.StartPoint().Row) + 1
		for _, target := range targets {
			edge := r.edge(file, line, r.absolute(file, target), kind)
			if usedIDs[edge.EvidenceID] {
				edge.EvidenceID += fmt.Sprintf(":column-%d", node.StartPoint().Column+1)
			}
			usedIDs[edge.EvidenceID] = true
			edges = append(edges, edge)
		}
	}
	return edges, nil
}

func (tree pythonTree) kind(node *gts.Node) string {
	if node == nil {
		return ""
	}
	return node.Type(tree.lang)
}

func (tree pythonTree) field(node *gts.Node, name string) *gts.Node {
	if node == nil {
		return nil
	}
	return node.ChildByFieldName(name, tree.lang)
}

func (tree pythonTree) text(node *gts.Node) string {
	if node == nil {
		return ""
	}
	return node.Text(tree.body)
}

func (tree pythonTree) importPath(node *gts.Node) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.ReplaceAll(tree.text(node), "\\\r\n", ""), "\\\n", "")), "")
}

func (name pythonImportName) binding() string {
	if name.alias != "" {
		return name.alias
	}
	first, _, _ := strings.Cut(name.name, ".")
	return first
}

func (tree pythonTree) importNames(statement *gts.Node) []pythonImportName {
	var names []pythonImportName
	for i := 0; i < statement.ChildCount(); i++ {
		item := statement.Child(i)
		if statement.FieldNameForChild(i, tree.lang) != "name" && tree.kind(item) != "wildcard_import" {
			continue
		}
		if tree.kind(item) == "aliased_import" {
			names = append(names, pythonImportName{name: tree.importPath(tree.field(item, "name")), alias: tree.text(tree.field(item, "alias"))})
		} else {
			names = append(names, pythonImportName{name: tree.importPath(item)})
		}
	}
	return names
}

func (tree pythonTree) writeNames(node *gts.Node) {
	switch tree.kind(node) {
	case "identifier":
		name := tree.text(node)
		tree.writers[name] = append(tree.writers[name], node.StartByte())
	case "attribute", "subscript":
		owner := tree.field(node, "object")
		if owner == nil {
			owner = tree.field(node, "value")
		}
		tree.writeNames(owner)
	case "default_parameter", "typed_default_parameter":
		tree.writeNames(tree.field(node, "name"))
	case "typed_parameter":
		tree.writeNames(node.NamedChild(0))
	case "pattern_list", "tuple_pattern", "list_pattern", "list_splat_pattern", "dictionary_splat_pattern", "as_pattern_target",
		"tuple", "list", "expression_list", "parenthesized_expression":
		for i := 0; i < node.NamedChildCount(); i++ {
			tree.writeNames(node.NamedChild(i))
		}
	}
}

func (tree pythonTree) dynamicTargets(call *gts.Node, builtin bool) (kind string, targets []string) {
	const unknown = "UNBOUNDED_DYNAMIC_IMPORT"
	args := tree.field(call, "arguments")
	if args == nil {
		return unknown, nil
	}
	var positional []*gts.Node
	var levels []*gts.Node
	for i := 0; i < args.NamedChildCount(); i++ {
		arg := args.NamedChild(i)
		switch tree.kind(arg) {
		case "comment":
		case "list_splat", "dictionary_splat":
			return unknown, nil
		case "keyword_argument":
			if builtin && tree.text(tree.field(arg, "name")) == "level" {
				levels = append(levels, tree.field(arg, "value"))
			}
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) == 0 || builtin && len(positional) > 5 {
		return unknown, nil
	}
	if builtin && len(positional) == 5 {
		levels = append(levels, positional[4])
	}
	if len(levels) > 1 {
		return unknown, nil
	}
	if len(levels) == 1 {
		level := levels[0]
		if tree.kind(level) != "integer" {
			return unknown, nil
		}
		value, err := strconv.ParseUint(strings.ReplaceAll(tree.text(level), "_", ""), 0, 64)
		if err != nil || value != 0 {
			return unknown, nil
		}
	}
	value := positional[0]
	if text, ok := tree.stringLiteral(value); ok {
		return "STATIC_LITERAL_DYNAMIC_IMPORT", []string{text}
	}
	for loop := call.Parent(); loop != nil; loop = loop.Parent() {
		switch tree.kind(loop) {
		case "function_definition", "lambda", "class_definition":
			return unknown, nil
		case "for_statement":
			variable, values, body := tree.field(loop, "left"), tree.field(loop, "right"), tree.field(loop, "body")
			if tree.kind(variable) != "identifier" || values == nil || body == nil || call.StartByte() < body.StartByte() || call.EndByte() > body.EndByte() ||
				strings.HasPrefix(tree.text(loop), "async ") {
				continue
			}
			if !slices.Contains([]string{"list", "tuple", "set"}, tree.kind(values)) || values.NamedChildCount() > 32 {
				continue
			}
			name := tree.text(variable)
			if slices.ContainsFunc(tree.writers[name], func(position uint32) bool { return position >= body.StartByte() && position < body.EndByte() }) {
				continue
			}
			alternatives := []string{}
			for i := 0; i < values.NamedChildCount(); i++ {
				item := values.NamedChild(i)
				if tree.kind(item) == "comment" {
					continue
				}
				text, ok := tree.stringLiteral(item)
				if !ok {
					alternatives = nil
					break
				}
				alternatives = append(alternatives, text)
			}
			if len(alternatives) != 0 {
				if result := tree.boundExpression(value, name, alternatives, 0); len(result) != 0 {
					slices.Sort(result)
					return "BOUNDED_DYNAMIC_IMPORT", slices.Compact(result)
				}
			}
		}
	}
	return unknown, nil
}

func (tree pythonTree) boundExpression(node *gts.Node, variable string, values []string, depth int) []string {
	if depth >= 16 {
		return nil
	}
	if tree.kind(node) == "identifier" && tree.text(node) == variable {
		return slices.Clone(values)
	}
	if text, ok := tree.stringLiteral(node); ok {
		return []string{text}
	}
	if tree.kind(node) == "parenthesized_expression" && node.NamedChildCount() == 1 {
		return tree.boundExpression(node.NamedChild(0), variable, values, depth+1)
	}
	if tree.kind(node) != "binary_operator" || tree.text(tree.field(node, "operator")) != "+" {
		return nil
	}
	left := tree.boundExpression(tree.field(node, "left"), variable, values, depth+1)
	right := tree.boundExpression(tree.field(node, "right"), variable, values, depth+1)
	count := len(left) * len(right)
	if count == 0 || count > 32 {
		return nil
	}
	result := make([]string, 0, count)
	for _, first := range left {
		for _, second := range right {
			result = append(result, first+second)
		}
	}
	return result
}

func (tree pythonTree) stringLiteral(node *gts.Node) (string, bool) {
	var value strings.Builder
	switch tree.kind(node) {
	case "concatenated_string":
		for i := 0; i < node.NamedChildCount(); i++ {
			part := node.NamedChild(i)
			if tree.kind(part) == "comment" {
				continue
			}
			if tree.kind(part) != "string" {
				return "", false
			}
			text, ok := tree.stringLiteral(part)
			if !ok {
				return "", false
			}
			value.WriteString(text)
		}
		return value.String(), true
	case "parenthesized_expression":
		if node.NamedChildCount() == 1 && tree.kind(node.NamedChild(0)) == "string" {
			return tree.stringLiteral(node.NamedChild(0))
		}
		return "", false
	case "string":
	default:
		return "", false
	}
	raw := tree.text(node)
	quote := strings.IndexAny(raw, "\"'")
	if quote < 0 {
		return "", false
	}
	prefix := strings.ToLower(raw[:quote])
	if prefix != "" && prefix != "u" && prefix != "r" {
		return "", false
	}
	width := 1
	if len(raw) >= quote+6 && raw[quote] == raw[quote+1] && raw[quote] == raw[quote+2] {
		width = 3
	}
	if len(raw) < quote+2*width || !strings.HasSuffix(raw, raw[quote:quote+width]) {
		return "", false
	}
	body := raw[quote+width : len(raw)-width]
	if prefix == "r" || !strings.Contains(body, "\\") {
		return body, true
	}
	for len(body) != 0 {
		before, rest, escaped := strings.Cut(body, "\\")
		value.WriteString(before)
		if !escaped {
			break
		}
		if len(rest) == 0 {
			return "", false
		}
		switch rest[0] {
		case '\n':
			body = rest[1:]
		case '\r':
			body = strings.TrimPrefix(rest[1:], "\n")
		case '\\', '\'', '"':
			value.WriteByte(rest[0])
			body = rest[1:]
		case 'x', 'u', 'U':
			width := 2
			if rest[0] == 'u' {
				width = 4
			} else if rest[0] == 'U' {
				width = 8
			}
			if len(rest) < width+1 {
				return "", false
			}
			char, err := strconv.ParseUint(rest[1:width+1], 16, 32)
			if err != nil || !utf8.ValidRune(rune(char)) {
				return "", false
			}
			value.WriteRune(rune(char))
			body = rest[width+1:]
		default:
			return "", false
		}
	}
	return value.String(), true
}
