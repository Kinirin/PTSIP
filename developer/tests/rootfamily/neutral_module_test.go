package rootfamily

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Execute the same neutral module with an independent Go interpreter. Its
// business predicates come from the admitted program, rather than Go constants.
func executeNeutral(t *testing.T, root string, module, graph, source map[string]any) any {
	t.Helper()
	env := map[string]any{"parameters": module["parameters"], "registry": graph, "source": source}
	var expression func(any) any
	expression = func(raw any) any {
		n := mapping(t, raw)
		operator := n["op"].(string)
		if operator == "literal" {
			return cloneJSON(t, n["value"])
		}
		if operator == "ref" {
			var value any = env
			for _, token := range strings.Split(n["path"].(string)[1:], "/") {
				if m, ok := value.(map[string]any); ok {
					value = m[token]
				} else {
					return nil
				}
			}
			return value
		}
		args := []any{}
		for _, a := range sequence(t, n["args"]) {
			args = append(args, expression(a))
		}
		switch operator {
		case "eq":
			return string(canonicalJSON(t, args[0])) == string(canonicalJSON(t, args[1]))
		case "all":
			for _, a := range args {
				if a != true {
					return false
				}
			}
			return true
		case "not":
			return args[0] != true
		case "contains":
			for _, a := range sequence(t, args[0]) {
				if reflect.DeepEqual(a, args[1]) {
					return true
				}
			}
			return false
		case "get":
			if m, ok := args[0].(map[string]any); ok {
				return m[args[1].(string)]
			}
			return nil
		case "concat":
			v := ""
			for _, a := range args {
				v += fmt.Sprint(a)
			}
			return v
		case "starts_with":
			return strings.HasPrefix(args[0].(string), args[1].(string))
		case "has_key":
			m, ok := args[0].(map[string]any)
			if !ok {
				return false
			}
			_, exists := m[args[1].(string)]
			return exists
		case "pluck":
			v := []any{}
			if args[0] == nil {
				return v
			}
			for _, a := range sequence(t, args[0]) {
				v = append(v, mapping(t, a)[args[1].(string)])
			}
			return v
		case "pointer_first":
			v := strings.Split(args[0].(string)[1:], "/")[0]
			return strings.ReplaceAll(strings.ReplaceAll(v, "~1", "/"), "~0", "~")
		case "pointer_nonoverlap":
			p := args[0].(string)
			if !strings.HasPrefix(p, "/") {
				return false
			}
			for _, a := range sequence(t, args[1]) {
				v := a.(string)
				if p == v || strings.HasPrefix(p, v+"/") || strings.HasPrefix(v, p+"/") {
					return false
				}
			}
			return true
		case "read_record":
			return read(t, filepath.Join(root, args[0].(string)))
		case "record_digest":
			return fmt.Sprintf("%x", sha256.Sum256(canonicalJSON(t, args[0])))
		default:
			t.Fatalf("unknown neutral expression %s", operator)
		}
		return nil
	}
	var execute func([]any) any
	execute = func(steps []any) any {
		for _, raw := range steps {
			s := mapping(t, raw)
			switch s["op"] {
			case "let":
				env[s["name"].(string)] = cloneJSON(t, expression(s["value"]))
			case "append":
				name := s["name"].(string)
				env[name] = append(sequence(t, env[name]), expression(s["value"]))
			case "assert":
				if expression(s["predicate"]) != true {
					t.Fatal(s["error"])
				}
			case "foreach":
				for _, item := range sequence(t, expression(s["items"])) {
					env[s["name"].(string)] = item
					execute(sequence(t, s["body"]))
				}
			case "set_pointer":
				pointer := expression(s["pointer"]).(string)
				target := mapping(t, env[s["name"].(string)])
				tokens := strings.Split(pointer[1:], "/")
				for i := range tokens {
					tokens[i] = strings.ReplaceAll(strings.ReplaceAll(tokens[i], "~1", "/"), "~0", "~")
				}
				for _, token := range tokens[:len(tokens)-1] {
					if target[token] == nil {
						target[token] = map[string]any{}
					}
					target = mapping(t, target[token])
				}
				target[tokens[len(tokens)-1]] = cloneJSON(t, expression(s["value"]))
			case "return":
				return expression(s["value"])
			default:
				t.Fatalf("unknown neutral instruction %v", s["op"])
			}
		}
		return nil
	}
	return execute(sequence(t, module["program"]))
}

func cloneJSON(t *testing.T, value any) any {
	t.Helper()
	var result any
	if err := json.Unmarshal(canonicalJSON(t, value), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestNeutralModuleExecutesInGoForBothPlanes(t *testing.T) {
	acceptance := contract(t).Neutral
	if _, err := os.Stat(filepath.Join(repository(t), "src/ptsip/policy_projection.py")); !os.IsNotExist(err) {
		t.Fatal("new handwritten Python module is still present")
	}
	for _, p := range contract(t).Planes {
		t.Run(p.PolicyClass, func(t *testing.T) {
			root := filepath.Join(repository(t), p.Path)
			index := read(t, filepath.Join(root, "index.yaml"))
			if index["projection_module_ref"] != acceptance.ModuleRef {
				t.Fatal("neutral module is not admitted")
			}
			module := read(t, filepath.Join(root, index["projection_module_ref"].(string)))
			if module["policy_class"] != p.PolicyClass || module["language_neutral"] != true || module["program_language"] != acceptance.ProgramLanguage {
				t.Fatal("neutral module identity mismatch")
			}
			graph := read(t, filepath.Join(root, "registries/root-family-migration.json"))
			for _, raw := range sequence(t, graph["sources"]) {
				source := mapping(t, raw)
				result := executeNeutral(t, root, module, graph, source)
				if string(canonicalJSON(t, result)) != string(canonicalJSON(t, read(t, filepath.Join(root, source["archive_path"].(string))))) {
					t.Fatal("neutral Go result changed source semantics", source["source_policy_id"])
				}
			}
		})
	}
	binary := buildAutomation(t)
	for _, scope := range []string{"src/new_module.json", "new_module.json"} {
		result, err, output := automation(t, binary, repository(t), "policy-resolver", "resolve", "--scope", scope, "--operation", "MODIFY")
		if err != nil {
			t.Fatalf("native module creation routing failed: %v\n%s", err, output)
		}
		found := false
		for _, raw := range sequence(t, result["policies"]) {
			if mapping(t, raw)["policy_id"] == acceptance.CreationPolicyRef {
				found = true
			}
		}
		if !found {
			t.Fatal("future module creation policy not resolved", scope)
		}
	}
}

func TestNeutralModuleAdmissionFailsClosed(t *testing.T) {
	for _, example := range contract(t).Neutral.Cases {
		failure := example.ID
		t.Run(failure, func(t *testing.T) {
			root := copyPolicy(t)
			path := filepath.Join(root, contract(t).Neutral.ModuleRef)
			switch failure {
			case "missing_program":
				os.Remove(path)
			case "foreign_program":
				v := read(t, path)
				v["policy_class"] = "PTSIP_DEVELOPER_POLICY"
				write(t, path, v)
			case "program_drift":
				v := read(t, path)
				v["program"] = []any{map[string]any{"op": "return", "value": map[string]any{"op": "literal", "value": map[string]any{}}}}
				write(t, path, v)
			case "owner_missing":
				os.Remove(filepath.Join(root, "CNTR/SFP-CNTR-0004.yaml"))
			case "checkout_crlf":
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				lf := strings.ReplaceAll(string(data), "\r\n", "\n")
				if err := os.WriteFile(path, []byte(strings.ReplaceAll(lf, "\n", "\r\n")), 0600); err != nil {
					t.Fatal(err)
				}
			}
			result := python(t, "-c", `import json,sys
from pathlib import Path
from ptsip.governance.authority import validate_migration
try:
 validate_migration(Path(sys.argv[1]),'PTSIP_SUPPORT_FEATURE')
 print(json.dumps({'accepted':True}))
except (ValueError,OSError) as exc:
 print(json.dumps({'accepted':False,'error':str(exc)}))`, root)
			if result["accepted"] != example.Accepted {
				t.Fatalf("%s expected admission %t, got %v", failure, example.Accepted, result)
			}
		})
	}
}
