package machine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func pythonTree(source []byte) (*gts.Tree, *gts.Language, error) {
	lang := grammars.PythonLanguage()
	parser := gts.NewParser(lang)
	tree, err := parser.ParseStrict(source)
	if err != nil {
		return nil, nil, fmt.Errorf("Python grammar parse did not complete: %w", err)
	}
	if tree == nil || tree.RootNode() == nil || tree.RootNode().HasErrorOrMissing() {
		if tree != nil {
			tree.Release()
		}
		return nil, nil, fmt.Errorf("implementation source is not valid Python")
	}
	return tree, lang, nil
}
func pythonDeclaration(node *gts.Node, lang *gts.Language) *gts.Node {
	if node.Type(lang) == "decorated_definition" {
		return node.ChildByFieldName("definition", lang)
	}
	return node
}
func pythonNamedDeclarations(container *gts.Node, lang *gts.Language, source []byte, kind, name string) []*gts.Node {
	matches := []*gts.Node{}
	for i := 0; i < container.NamedChildCount(); i++ {
		node := pythonDeclaration(container.NamedChild(i), lang)
		if node == nil || node.Type(lang) != kind {
			continue
		}
		field := node.ChildByFieldName("name", lang)
		if field != nil && field.Text(source) == name {
			matches = append(matches, node)
		}
	}
	return matches
}

func pythonStringLiteral(node *gts.Node, lang *gts.Language, source []byte) (string, bool) {
	if node == nil || node.Type(lang) != "string" {
		return "", false
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		if node.NamedChild(i).Type(lang) == "interpolation" {
			return "", false
		}
	}
	raw := node.Text(source)
	start := strings.IndexAny(raw, "\"'")
	if start < 0 {
		return "", false
	}
	prefix := strings.ToLower(raw[:start])
	if strings.ContainsAny(prefix, "fb") {
		return "", false
	}
	quoted := raw[start:]
	if len(quoted) < 2 {
		return "", false
	}
	quote := string(quoted[0])
	width := 1
	if strings.HasPrefix(quoted, strings.Repeat(quote, 3)) {
		width = 3
	}
	if len(quoted) < width*2 || !strings.HasSuffix(quoted, strings.Repeat(quote, width)) {
		return "", false
	}
	body := quoted[width : len(quoted)-width]
	if strings.Contains(prefix, "r") {
		return body, true
	}
	out := strings.Builder{}
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' {
			out.WriteByte(body[i])
			continue
		}
		i++
		if i >= len(body) {
			return "", false
		}
		switch body[i] {
		case '\n':
			continue
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case '\\', '\'', '"':
			out.WriteByte(body[i])
		case 'x', 'u', 'U':
			width := 2
			if body[i] == 'u' {
				width = 4
			} else if body[i] == 'U' {
				width = 8
			}
			if i+width >= len(body) {
				return "", false
			}
			code, err := strconv.ParseInt(body[i+1:i+1+width], 16, 32)
			if err != nil {
				return "", false
			}
			out.WriteRune(rune(code))
			i += width
		default:
			if body[i] >= '0' && body[i] <= '7' {
				digits := string(body[i])
				for len(digits) < 3 && i+1 < len(body) && body[i+1] >= '0' && body[i+1] <= '7' {
					i++
					digits += string(body[i])
				}
				code, _ := strconv.ParseInt(digits, 8, 32)
				out.WriteRune(rune(code))
			} else {
				out.WriteByte('\\')
				out.WriteByte(body[i])
			}
		}
	}
	return out.String(), true
}

func pythonCLIComparison(node *gts.Node, lang *gts.Language, source []byte, command string) bool {
	if node == nil || node.Type(lang) != "comparison_operator" || node.NamedChildCount() != 2 {
		return false
	}
	left, right := node.NamedChild(0), node.NamedChild(1)
	if left.Type(lang) != "attribute" {
		return false
	}
	object := left.ChildByFieldName("object", lang)
	attribute := left.ChildByFieldName("attribute", lang)
	if object == nil || attribute == nil || object.Type(lang) != "identifier" || object.Text(source) != "args" || attribute.Text(source) != "command" {
		return false
	}
	between := strings.TrimSpace(string(source[left.EndByte():right.StartByte()]))
	if between != "==" {
		return false
	}
	value, ok := pythonStringLiteral(right, lang, source)
	return ok && value == command
}
func pythonLastSemanticRow(node *gts.Node, lang *gts.Language) uint32 {
	if node == nil {
		return 0
	}
	if node.Type(lang) == "comment" {
		return 0
	}
	if node.NamedChildCount() == 0 {
		return node.EndPoint().Row + 1
	}
	row := uint32(0)
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Type(lang) == "comment" {
			continue
		}
		candidate := pythonLastSemanticRow(child, lang)
		if candidate > row {
			row = candidate
		}
	}
	if row == 0 {
		return node.EndPoint().Row + 1
	}
	return row
}

// ValidateImplementationRef uses complete grammar trees and closed typed selectors.
// Python sources are inspected as data; they are never imported or executed.
func ValidateImplementationRef(r *Repository, reference Object) (Object, error) {
	ref := Text(reference["path"])
	selector := Map(reference["selector"])
	if ref == "" || selector == nil {
		return nil, fmt.Errorf("implementation reference requires path and selector")
	}
	relative, err := r.Scope(ref)
	if err != nil {
		return nil, err
	}
	source, err := agentText(r, ref)
	if err != nil {
		return nil, err
	}
	kind := Text(selector["kind"])
	var start, end int
	if strings.HasPrefix(kind, "GO_") {
		if filepath.Ext(ref) != ".go" {
			return nil, fmt.Errorf("Go selector must target Go source")
		}
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, relative, source, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("invalid Go source: %w", err)
		}
		if kind == "GO_MODULE" {
			start = 1
			end = len(strings.Split(strings.TrimSuffix(source, "\n"), "\n"))
		} else {
			matches := []ast.Node{}
			for _, raw := range file.Decls {
				switch node := raw.(type) {
				case *ast.FuncDecl:
					if kind == "GO_FUNCTION" && node.Recv == nil && node.Name.Name == Text(selector["name"]) {
						matches = append(matches, node)
					}
					if kind == "GO_METHOD" && node.Recv != nil && node.Name.Name == Text(selector["method"]) {
						receiver := node.Recv.List[0].Type
						if star, ok := receiver.(*ast.StarExpr); ok {
							receiver = star.X
						}
						if identifier, ok := receiver.(*ast.Ident); ok && identifier.Name == Text(selector["type"]) {
							matches = append(matches, node)
						}
					}
				case *ast.GenDecl:
					if kind == "GO_TYPE" {
						for _, spec := range node.Specs {
							if typ, ok := spec.(*ast.TypeSpec); ok && typ.Name.Name == Text(selector["name"]) {
								matches = append(matches, typ)
							}
						}
					}
				}
			}
			if len(matches) != 1 {
				return nil, fmt.Errorf("typed Go selector must resolve exactly once")
			}
			start = set.Position(matches[0].Pos()).Line
			end = set.Position(matches[0].End()).Line
		}
	} else {
		if filepath.Ext(ref) != ".py" {
			return nil, fmt.Errorf("Python selector must target Python source")
		}
		if kind != "PYTHON_FUNCTION" && kind != "PYTHON_METHOD" && kind != "CLI_COMMAND_BRANCH" && kind != "PYTHON_MODULE" {
			return nil, fmt.Errorf("unsupported implementation selector kind %q", kind)
		}
		data := []byte(source)
		tree, lang, err := pythonTree(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", relative, err)
		}
		defer tree.Release()
		root := tree.RootNode()
		matches := []*gts.Node{}
		switch kind {
		case "PYTHON_MODULE":
			start = 1
			end = len(strings.Split(strings.TrimSuffix(source, "\n"), "\n"))
			if end < 1 {
				end = 1
			}
		case "PYTHON_FUNCTION":
			name := Text(selector["name"])
			if name == "" {
				return nil, fmt.Errorf("PYTHON_FUNCTION requires name")
			}
			matches = pythonNamedDeclarations(root, lang, data, "function_definition", name)
		case "PYTHON_METHOD":
			className, method := Text(selector["class"]), Text(selector["method"])
			if className == "" || method == "" {
				return nil, fmt.Errorf("PYTHON_METHOD requires class and method")
			}
			classes := pythonNamedDeclarations(root, lang, data, "class_definition", className)
			if len(classes) != 1 {
				return nil, fmt.Errorf("class %q must resolve exactly once", className)
			}
			body := classes[0].ChildByFieldName("body", lang)
			if body == nil {
				return nil, fmt.Errorf("class body not resolved")
			}
			matches = pythonNamedDeclarations(body, lang, data, "function_definition", method)
		case "CLI_COMMAND_BRANCH":
			command := Text(selector["command"])
			if command == "" {
				return nil, fmt.Errorf("CLI_COMMAND_BRANCH requires command")
			}
			var visit func(*gts.Node)
			visit = func(node *gts.Node) {
				typ := node.Type(lang)
				if (typ == "if_statement" || typ == "elif_clause") && pythonCLIComparison(node.ChildByFieldName("condition", lang), lang, data, command) {
					matches = append(matches, node)
				}
				for i := 0; i < node.NamedChildCount(); i++ {
					visit(node.NamedChild(i))
				}
			}
			visit(root)
		}
		if kind != "PYTHON_MODULE" {
			if len(matches) != 1 {
				return nil, fmt.Errorf("%s: %s must resolve exactly once", relative, kind)
			}
			selected := matches[0]
			start = int(selected.StartPoint().Row) + 1
			end = int(pythonLastSemanticRow(selected, lang))
			// ast.If representing an elif owns every following elif/else branch.
			if selected.Type(lang) == "elif_clause" {
				parent := selected.Parent()
				if parent != nil && parent.Type(lang) == "if_statement" {
					end = int(pythonLastSemanticRow(parent, lang))
				}
			}
		}
	}
	return Object{"path": relative, "selector": selector, "resolved_location": Object{"line_start": start, "line_end": end}}, nil
}

func PythonTestNodeExists(r *Repository, nodeID string) bool {
	parts := strings.Split(nodeID, "::")
	if len(parts) < 2 {
		return false
	}
	source, err := agentText(r, parts[0])
	if err != nil {
		return false
	}
	data := []byte(source)
	tree, lang, err := pythonTree(data)
	if err != nil {
		return false
	}
	defer tree.Release()
	container := tree.RootNode()
	for i, name := range parts[1:] {
		matches := pythonNamedDeclarations(container, lang, data, "function_definition", name)
		matches = append(matches, pythonNamedDeclarations(container, lang, data, "class_definition", name)...)
		if len(matches) != 1 {
			return false
		}
		selected := matches[0]
		if i < len(parts)-2 {
			if selected.Type(lang) != "class_definition" {
				return false
			}
			container = selected.ChildByFieldName("body", lang)
			if container == nil {
				return false
			}
		}
	}
	return true
}

// PythonIdentifierPresent preserves the former ast.Name/FunctionDef/
// AsyncFunctionDef inspection. It excludes strings, comments, attributes,
// declaration-only class/parameter names, and import aliases.
func PythonIdentifierPresent(r *Repository, ref, name string) (bool, error) {
	source, err := agentText(r, ref)
	if err != nil {
		return false, err
	}
	data := []byte(source)
	tree, lang, err := pythonTree(data)
	if err != nil {
		return false, err
	}
	defer tree.Release()
	found := false
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if found {
			return
		}
		kind := node.Type(lang)
		if kind == "function_definition" {
			identifier := node.ChildByFieldName("name", lang)
			if identifier != nil && identifier.Text(data) == name {
				found = true
				return
			}
		}
		if kind == "identifier" && node.Text(data) == name {
			parent := node.Parent()
			allowed := true
			if parent != nil {
				parentKind := parent.Type(lang)
				switch parentKind {
				case "attribute":
					allowed = parent.ChildByFieldName("attribute", lang) != node
				case "class_definition":
					allowed = parent.ChildByFieldName("name", lang) != node
				case "parameters", "lambda_parameters", "type_parameter":
					allowed = false
				case "default_parameter", "typed_default_parameter", "keyword_argument":
					allowed = parent.ChildByFieldName("name", lang) != node
				case "typed_parameter":
					allowed = parent.NamedChild(0) != node
				}
				for ancestor := parent; allowed && ancestor != nil; ancestor = ancestor.Parent() {
					typ := ancestor.Type(lang)
					if typ == "import_statement" || typ == "import_from_statement" {
						allowed = false
					}
				}
			}
			if allowed {
				found = true
				return
			}
		}
		for i := 0; i < node.NamedChildCount(); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(tree.RootNode())
	return found, nil
}
