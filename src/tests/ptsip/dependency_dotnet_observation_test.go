package contracts_test

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/Kinirin/PTSIP/src/ptsip/inspection"
)

func observeDotNetFixture(t *testing.T, fixtures map[string]string, ignored string) inspection.DependencyScan {
	t.Helper()
	root := repositoryFixture(t)
	var files []string
	for path, content := range fixtures {
		writeRepositoryFixture(t, root, path, content)
		if path != ignored {
			files = append(files, path)
		}
	}
	data, err := json.Marshal(object{"root": root, "files": files, "observation_errors": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	scan, err := inspection.ObserveDependencies(root, data)
	if err != nil {
		t.Fatal(err)
	}
	return scan
}

func TestDependencyDotNetUsesDeclaredProjectIdentity(t *testing.T) {
	utf16Project := []byte{0xff, 0xfe}
	for _, unit := range utf16.Encode([]rune(`<?xml version="1.0" encoding="utf-16"?><Project><PropertyGroup><AssemblyName>Library</AssemblyName></PropertyGroup></Project>`)) {
		utf16Project = binary.LittleEndian.AppendUint16(utf16Project, unit)
	}
	scan := observeDotNetFixture(t, map[string]string{
		"src/app/App.csproj": `<Project xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <PropertyGroup><RootNamespace>App</RootNamespace></PropertyGroup>
  <ItemGroup><ProjectReference Include="..\lib\Library.csproj"/><PackageReference Include="NUnit.Framework"/></ItemGroup>
</Project>`,
		"src/lib/Library.csproj": string(utf16Project),
		"src/other/Loose.csproj": `<Project><PropertyGroup><RootNamespace>LocalOnly</RootNamespace></PropertyGroup></Project>`,
		"src/app/tests/Example.cs": `global using Alias = global::Library.Tools;
using static Library.Extensions; using System.Text;
using NUnit.Framework.Internal;
using LocalOnly.Other;
/*
using Comment.Hidden;
*/
var raw = """
using Raw.Hidden;
""";
var quoted = @"
using Verbatim.Hidden;
";
var interpolation = $"{F("using Nested.Hidden;")}";
#warning using Directive.Hidden;
using var file = Open();
`,
	}, "")
	if !reflect.DeepEqual(scan.Adapters, []string{"dotnet-csproj", "dotnet-source"}) || !scan.CoverageComplete {
		t.Fatalf("unexpected .NET observation coverage: %+v", scan)
	}
	if len(scan.Edges) != 6 {
		t.Fatalf("expected one declaration and five observed using directives: %+v", scan.Edges)
	}
	for _, edge := range scan.Edges {
		switch edge.Target {
		case "../lib/Library.csproj", "Library.Tools", "Library.Extensions":
			if edge.Resolution != "RESOLVED" || edge.TargetScope != "PROJECT_COMPONENT" || edge.ResolvedPath == nil || *edge.ResolvedPath != "src/lib/Library.csproj" {
				t.Fatalf("local declaration or namespace lost its exact project path: %+v", edge)
			}
		case "System.Text":
			if edge.Resolution != "EXTERNAL" || edge.TargetScope != "PLATFORM" {
				t.Fatalf(".NET platform namespace classified incorrectly: %+v", edge)
			}
		case "NUnit.Framework.Internal":
			if edge.Resolution != "EXTERNAL" || edge.TargetScope != "EXTERNAL_DEPENDENCY" {
				t.Fatalf("literal package declaration lost: %+v", edge)
			}
		case "LocalOnly.Other":
			if edge.Resolution != "UNRESOLVED" || edge.ResolvedPath != nil {
				t.Fatalf("unreferenced local project was inferred: %+v", edge)
			}
		default:
			t.Fatalf("literal or comment text escaped the C# observation filter: %+v", edge)
		}
		if edge.Adapter == "dotnet-source" && (edge.Line == nil || edge.Phase != "TEST" || edge.Provenance != "OBSERVED") {
			t.Fatalf("source provenance or line was lost: %+v", edge)
		}
	}
}

func TestDependencyDotNetRetainsConditionalAndMissingEvidence(t *testing.T) {
	scan := observeDotNetFixture(t, map[string]string{
		"App.csproj": `<Project>
  <ItemGroup Condition="'$(Configuration)' == 'Debug'"><ProjectReference Include="hidden\Missing.csproj"/></ItemGroup>
  <ItemGroup><ProjectReference Include="hidden\Unobserved.csproj"/><ProjectReference Include="$(OtherProject)"/></ItemGroup>
</Project>`,
		"hidden/Unobserved.csproj": `<Project/>`,
		"Example.cs": `#if DEBUG
using Conditional;
#endif
using Alias = List<int>;
`,
	}, "hidden/Unobserved.csproj")
	if scan.CoverageComplete || len(scan.Edges) != 4 {
		t.Fatalf("unevaluated declarations were dropped or inferred: %+v", scan)
	}
	for _, edge := range scan.Edges {
		if edge.Resolution != "UNRESOLVED" || edge.ResolvedPath != nil || edge.Phase != "UNKNOWN" {
			t.Fatalf("conditional, dynamic or unobserved target was inferred: %+v", edge)
		}
	}
	for _, text := range []string{"depends on MSBuild evaluation", "absent from the observed repository", "unevaluated preprocessor branch", "unsupported or incomplete C# using declaration"} {
		if !slices.ContainsFunc(scan.Issues, func(issue inspection.DependencyIssue) bool { return strings.Contains(issue.Message, text) }) {
			t.Fatalf("missing %q diagnostic: %+v", text, scan.Issues)
		}
	}
}
