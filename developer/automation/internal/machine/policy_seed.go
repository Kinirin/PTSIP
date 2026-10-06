package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v3"
	"go/format"
	"os"
	"sort"
	"strings"
)

const policySeedAcceptancePath = "developer/tests/rootfamily/seeded_pp_102_acceptance_test.go"
const policySeedProfileNote = "# Project Profile %s\n\nState: Current Project Profile contract\nTransition: pp.1.01 -> %s / SEMANTIC_MIGRATION\nSpecification binding: 0.3.7-draft @ %s\n\n## Purpose\n\nThis contract separates developer-owned Project Profile generation from\nuser-owned profile lineage and removes path-shaped examples that could be\nmisread as repository architecture authority.\n\n## Contract changes\n\n- ptsip.revision is now required and uses canonical Rev.#### form.\n- Developer-distributed baselines begin at Rev.0001.\n- ptsip.specification.family is no longer serialized in current Project\n  Profiles. Specification identity remains exact through source plus the\n  immutable Git revision.\n- ptsip.profile_role distinguishes repository authority (PROJECT) from\n  distributed examples (DISTRIBUTED_EXAMPLE).\n- Distributed examples require project-path materialization and must not be\n  treated as canonical repository layout.\n- New repository-local profile storage uses .ptsip/profiles/index.yaml with\n  an explicit default_profile and one or more *.ptsip.yaml resources.\n- New adoption defaults to .ptsip/profiles/main.ptsip.yaml.\n- Repository-root ptsip.yaml remains a compatibility/migration input rather\n  than the default target for new profile creation.\n\n## Example-path safety\n\nPublic examples use unresolved project-owned selector placeholders instead of\nprescribing paths such as product/app/** or a developer's private tooling\nlayout. A project or agent must resolve those selectors against repository\nevidence and explicit project authority before materializing a PROJECT profile.\n\n## Authority boundaries\n\nThis Project Profile transition does not change the frozen Specification\nfamily or its immutable revision. It also does not rewrite historical Tool\nrelease notes. Tool SemVer, Project Profile contract identity,\nptsip.revision, and Specification revision remain separate axes.\n"
const policySeedAcceptance = "package rootfamily\n\nimport (\n \"encoding/json\"\n \"os\"\n \"path/filepath\"\n \"testing\"\n \"go.yaml.in/yaml/v3\"\n)\n\nfunc seededPP102Read(t *testing.T, path string) map[string]any {\n t.Helper(); root, err := filepath.Abs(\"../../..\"); if err != nil {t.Fatal(err)}\n content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path))); if err != nil {t.Fatal(err)}\n var value map[string]any; if filepath.Ext(path)==\".json\" {err=json.Unmarshal(content,&value)} else {err=yaml.Unmarshal(content,&value)}\n if err!=nil {t.Fatal(err)};return value\n}\nfunc TestSeededPP102SemanticAcceptance(t *testing.T){\n registry:=seededPP102Read(t,\"registry/project-profile-contracts.yaml\");if registry[\"current\"]!=\"pp.1.02\"{t.Fatal(\"pp.1.02 transition output is not current\")}\n rows:=registry[\"contracts\"].([]any); var schemaPath string;for _,raw:=range rows{row:=raw.(map[string]any);if row[\"version\"]==\"pp.1.02\"{schemaPath=row[\"schema\"].(string)}}\n schema:=seededPP102Read(t,schemaPath);header:=schema[\"properties\"].(map[string]any)[\"ptsip\"].(map[string]any);properties:=header[\"properties\"].(map[string]any)\n revision:=properties[\"revision\"].(map[string]any);if revision[\"pattern\"]!=\"^Rev\\\\.[0-9]{4}$\"{t.Fatal(\"user revision contract missing\")}\n specification:=properties[\"specification\"].(map[string]any)[\"properties\"].(map[string]any);if _,present:=specification[\"family\"];present{t.Fatal(\"serialized specification family retained\")}\n catalog:=seededPP102Read(t,\"profiles/index.yaml\");for _,raw:=range catalog[\"profiles\"].([]any){row:=raw.(map[string]any);if row[\"contract\"]!=\"pp.1.02\"||row[\"profile_role\"]!=\"DISTRIBUTED_EXAMPLE\"||row[\"materialization\"]!=\"PROJECT_PATH_RESOLUTION_REQUIRED\"{t.Fatal(\"public example authority boundary invalid\")};profile:=seededPP102Read(t,\"profiles/\"+row[\"resource\"].(string));ptsip:=profile[\"ptsip\"].(map[string]any);if ptsip[\"version\"]!=\"pp.1.02\"||ptsip[\"revision\"]!=\"Rev.0001\"||ptsip[\"profile_role\"]!=\"DISTRIBUTED_EXAMPLE\"{t.Fatal(\"distributed profile header invalid\")}}\n local:=seededPP102Read(t,\".ptsip/profiles/index.yaml\");if local[\"schema_version\"]!=\"ptsip-local-profile-catalog/v1\"||local[\"default_profile\"]!=\"main\"{t.Fatal(\"local catalog contract invalid\")};profile:=seededPP102Read(t,\".ptsip/profiles/main.ptsip.yaml\");header=profile[\"ptsip\"].(map[string]any);if header[\"profile_role\"]!=\"PROJECT\"||header[\"version\"]!=\"pp.1.02\"||header[\"revision\"]!=\"Rev.0001\"{t.Fatal(\"project identity boundary invalid\")}\n}\n"

var policySeedMarkers = [][3]string{
	{"releasenote/project-profile/README.md", "Current intended contract:\n\n```text\npp.1.01\n```\n\nSee [`pp.1.01.md`](pp.1.01.md) for the identity-only compatibility notice covering historical `0.3.6-draft` profiles.\n", "Current intended contract:\n\n```text\npp.1.02\n```\n\nSee [`pp.1.02.md`](pp.1.02.md) for the current semantic-migration contract.\nThe historical [`pp.1.01.md`](pp.1.01.md) identity-only bridge remains preserved.\n"},
	{"releasenote/README.md", "releasenote/project-profile/pp.1.01.md\n", "releasenote/project-profile/pp.1.01.md\nreleasenote/project-profile/pp.1.02.md\n"},
	{"releasenote/README.md", "| `pp.1.01` | **Current contract; reused unchanged by Tool 0.3.8a1** | [`project-profile/pp.1.01.md`](project-profile/pp.1.01.md) |", "| `pp.1.02` | **Current contract; semantic migration from pp.1.01** | [`project-profile/pp.1.02.md`](project-profile/pp.1.02.md) |\n| `pp.1.01` | Historical identity-only baseline; superseded by current PP contract | [`project-profile/pp.1.01.md`](project-profile/pp.1.01.md) |"},
	{"releasenote/README.md", "| Project Profile | `pp.1.01` | Current contract / repository adopted | [`project-profile/pp.1.01.md`](project-profile/pp.1.01.md) |", "| Project Profile | `pp.1.02` | Current contract / repository adopted | [`project-profile/pp.1.02.md`](project-profile/pp.1.02.md) |"},
	{"README.md", "**Project Profile contract:** `pp.1.01`<br>", "**Project Profile contract:** `pp.1.02`<br>"},
	{"README.md", "The default project-owned profile is repository-root `ptsip.yaml`; projects may consistently use another explicit path through `--profile`.", "New project-owned profiles are selected through `.ptsip/profiles/index.yaml`; the catalog's `default_profile` resolves the active `*.ptsip.yaml` resource. Repository-root `ptsip.yaml` remains a compatibility/migration input, and an explicit `--profile` path still takes precedence."},
	{"README.md", "A Decision Authority does not replace `ptsip.yaml` and does not prove conformance.", "A Decision Authority does not replace the Project Profile selected by `.ptsip/profiles/index.yaml` and does not prove conformance. Legacy root `ptsip.yaml` remains compatibility input only."},
	{"README.md", "Tool `0.3.8a1` is bound to independent PP and Specification identities:\n\n```text\nProject Profile pp.1.01\nSpecification 0.3.7-draft\nSPEC_REVISION 3c47816770d194ae42f98faedc911d980db0e62a\n```\n", "The current source tree exposes independent PP and Specification identities:\n\n```text\nProject Profile pp.1.02\nSpecification 0.3.7-draft\nSPEC_REVISION %s\n```\n\nThe already-published Tool `0.3.8a1` release retains its historical PP binding\nin its Tool release note; advancing the PP contract does not rewrite that Tool history.\n"},
	{"STATUS.md", "- Project Profile contract: `pp.1.01`", "- Project Profile contract: `pp.1.02`"},
}

func policySeedHeader(payload Object, version, role string) error {
	header := Map(payload["ptsip"])
	if header == nil {
		return policyFailure("SEED_PROFILE_INVALID", "profile has no ptsip mapping")
	}
	specification := Map(header["specification"])
	source, revision := Text(specification["source"]), Text(specification["revision"])
	if source == "" || revision == "" {
		return policyFailure("SEED_SPECIFICATION_INVALID", "profile Specification source/revision required")
	}
	payload["ptsip"] = Object{"version": version, "revision": "Rev.0001", "profile_role": role, "specification": Object{"source": source, "revision": revision}}
	return nil
}
func policySeedPlaceholderizeExample(payload Object) {
	replacements := map[string][]any{"product-runtime": {"${PRODUCT_RUNTIME_ROOT}/**"}, "product-sdk": {"${PRODUCT_SDK_ROOT}/**"}, "product-tests": {"${PRODUCT_TEST_ROOT}/**"}, "development-toolkit": {"${DEVELOPMENT_TOOLING_ROOT}/**"}, "release-automation": {"${RELEASE_AUTOMATION_FILE}"}, "production-operations": {"${OPERATIONS_ROOT}/**"}, "shared-contracts": {"${CONTRACT_ROOT}/**"}}
	for _, raw := range List(payload["components"]) {
		component := Map(raw)
		id := Text(component["id"])
		if replacement, found := replacements[id]; found {
			component["include"] = replacement
		}
		if id == "development-toolkit" {
			component["analysis_inputs"] = []any{"${PRODUCT_ROOT}/**", "${CONTRACT_ROOT}/**"}
		}
	}
	for _, raw := range List(payload["associated_artifacts"]) {
		artifact := Map(raw)
		if artifact["id"] == "development-toolkit-docs" {
			artifact["include"] = []any{"${DEVELOPMENT_TOOLING_DOCS_ROOT}/**"}
		}
	}
}
func policySeedPlaceholderizeHybrid(payload Object) {
	for _, raw := range List(Map(Map(payload["responsibility_map"])["overrides"])["components"]) {
		component := Map(raw)
		if component["id"] == "package" {
			component["include"] = []any{"${PACKAGE_ROOT}/**"}
		} else if component["id"] == "package-tests" {
			component["include"] = []any{"${PRODUCT_TEST_ROOT}/**"}
		}
	}
}
func policySeedControlRoot(payload Object) {
	for _, raw := range List(payload["components"]) {
		component := Map(raw)
		if component["id"] == "repository-architecture" {
			include, ok := component["include"].([]any)
			if ok && !policyContains(policyStrings(include), ".ptsip/**") {
				component["include"] = append(include, ".ptsip/**")
			}
		}
	}
}
func policySeedSchema(payload Object) error {
	header := Map(Map(payload["properties"])["ptsip"])
	properties := Map(header["properties"])
	specification := Map(properties["specification"])
	if header == nil || properties == nil || specification == nil {
		return policyFailure("SEED_SCHEMA_INVALID", "ptsip Specification schema required")
	}
	header["required"] = []any{"version", "revision", "profile_role", "specification"}
	properties["revision"] = Object{"type": "string", "pattern": "^Rev\\.[0-9]{4}$", "description": "User-owned authoritative Project Profile generation within one PP contract. Developer-distributed baselines begin at Rev.0001."}
	properties["profile_role"] = Object{"enum": []any{"PROJECT", "DISTRIBUTED_EXAMPLE"}, "description": "PROJECT is repository authority. DISTRIBUTED_EXAMPLE is non-authoritative and requires project-path materialization before use."}
	specification["required"] = []any{"source", "revision"}
	delete(Map(specification["properties"]), "family")
	revision := Map(Map(specification["properties"])["revision"])
	if revision == nil {
		return policyFailure("SEED_SCHEMA_INVALID", "Specification revision schema missing")
	}
	revision["description"] = "Immutable Specification Git revision. This identity is independent from ptsip.revision, which is user-owned Project Profile lineage."
	return nil
}
func policySeedJSON(value any) ([]byte, error) {
	raw, err := CanonicalJSON(value)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return nil, err
	}
	return append(out.Bytes(), '\n'), nil
}
func (r *Repository) SeedPP102Transition(apply bool) (Object, error) {
	if !apply {
		return nil, policyFailure("SEED_APPLY_REQUIRED", "--apply is required; this operation seeds approved semantic inputs only")
	}
	status, err := r.GitOutput("status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(status) != "" {
		return nil, policyFailure("SEED_WORKTREE_DIRTY", "working tree/index must be clean")
	}
	registry, err := r.Read("registry/project-profile-contracts.yaml")
	if err != nil {
		return nil, err
	}
	if registry["current"] != "pp.1.01" {
		return nil, policyFailure("SEED_REPLAY_FORBIDDEN", fmt.Sprintf("seed requires current pp.1.01, found %v", registry["current"]))
	}
	catalog, err := r.Read("profiles/index.yaml")
	if err != nil {
		return nil, err
	}
	rows := List(catalog["profiles"])
	if len(rows) == 0 {
		return nil, policyFailure("SEED_CATALOG_EMPTY", "public profile catalog requires resources")
	}
	updates := map[string][]byte{}
	yamlUpdate := func(path string, value any) error {
		data, err := yaml.Marshal(value)
		if err == nil {
			updates[path] = data
		}
		return err
	}
	schemaPath := "schemas/ptsip-profile-pp-1.01.schema.json"
	schema, err := r.Read(schemaPath)
	if err != nil {
		return nil, err
	}
	if err := policySeedSchema(schema); err != nil {
		return nil, err
	}
	data, err := policySeedJSON(schema)
	if err != nil {
		return nil, err
	}
	updates[schemaPath] = data
	seenResources := map[string]bool{}
	for _, raw := range rows {
		row := Map(raw)
		if row == nil {
			return nil, policyFailure("SEED_CATALOG_INVALID", "profile catalog entry must be mapping")
		}
		resource := Text(row["resource"])
		relative, err := r.Scope("profiles/" + resource)
		if resource == "" || err != nil || !strings.HasPrefix(relative, "profiles/") || seenResources[relative] {
			return nil, policyFailure("SEED_RESOURCE_INVALID", "public profile resource must be unique and stay inside profiles/")
		}
		seenResources[relative] = true
		profile, err := r.Read(relative)
		if err != nil {
			return nil, err
		}
		if err := policySeedHeader(profile, "pp.1.01", "DISTRIBUTED_EXAMPLE"); err != nil {
			return nil, err
		}
		if row["id"] == "example" {
			policySeedPlaceholderizeExample(profile)
		} else if row["id"] == "hybrid-python-package" {
			policySeedPlaceholderizeHybrid(profile)
		}
		if err := yamlUpdate(relative, profile); err != nil {
			return nil, err
		}
		row["profile_role"] = "DISTRIBUTED_EXAMPLE"
		row["materialization"] = "PROJECT_PATH_RESOLUTION_REQUIRED"
	}
	if err := yamlUpdate("profiles/index.yaml", catalog); err != nil {
		return nil, err
	}
	rootProfile, err := r.Read("ptsip.yaml")
	if err != nil {
		return nil, err
	}
	if err := policySeedHeader(rootProfile, "pp.1.02", "PROJECT"); err != nil {
		return nil, err
	}
	policySeedControlRoot(rootProfile)
	if err := yamlUpdate(".ptsip/profiles/main.ptsip.yaml", rootProfile); err != nil {
		return nil, err
	}
	data, err = policySeedJSON(Object{"schema_version": "ptsip-local-profile-catalog/v1", "default_profile": "main", "profiles": []any{Object{"id": "main", "resource": "main.ptsip.yaml"}}})
	if err != nil {
		return nil, err
	}
	updates[".ptsip/profiles/index.yaml"] = data
	developerProfile, err := r.Read("developer/profiles/ptsip-repository.yaml")
	if err != nil {
		return nil, err
	}
	if err := policySeedHeader(developerProfile, "pp.1.02", "PROJECT"); err != nil {
		return nil, err
	}
	policySeedControlRoot(developerProfile)
	if err := yamlUpdate("developer/profiles/ptsip-repository.yaml", developerProfile); err != nil {
		return nil, err
	}
	revision := Text(Map(Map(rootProfile["ptsip"])["specification"])["revision"])
	updates["releasenote/project-profile/pp.1.02.md"] = []byte(fmt.Sprintf(policySeedProfileNote, "pp.1.02", "pp.1.02", revision))
	for _, marker := range policySeedMarkers {
		reference, old, replacement := marker[0], marker[1], marker[2]
		text, found := updates[reference]
		if !found {
			absolute, err := r.Path(reference)
			if err != nil {
				return nil, err
			}
			text, err = os.ReadFile(absolute)
			if err != nil {
				return nil, err
			}
			text = bytes.ReplaceAll(text, []byte("\r\n"), []byte("\n"))
		}
		if strings.Count(string(text), old) != 1 {
			return nil, policyFailure("SEED_DOCUMENT_MARKER_MISMATCH", "exactly one marker required in "+reference)
		}
		if strings.Contains(replacement, "SPEC_REVISION %s") {
			replacement = fmt.Sprintf(replacement, revision)
		}
		updates[reference] = []byte(strings.Replace(string(text), old, replacement, 1))
	}
	accepted, err := format.Source([]byte(policySeedAcceptance))
	if err != nil {
		return nil, err
	}
	updates[policySeedAcceptancePath] = accepted
	forbidden := []string{"registry/project-profile-contracts.yaml", "src/ptsip/specdata/project-profile-contracts.yaml", "profiles/history/", "schemas/ptsip-profile-pp-1.02.schema.json", "src/ptsip/specdata/ptsip-profile-pp-1.02.schema.json"}
	paths := []string{}
	original := map[string][]byte{}
	exists := map[string]bool{}
	digests := map[string]string{}
	for reference := range updates {
		for _, prefix := range forbidden {
			if reference == prefix || strings.HasPrefix(reference, prefix) {
				return nil, policyFailure("SEED_AUTOMATION_OUTPUT_FORBIDDEN", reference)
			}
		}
		absolute, err := r.Path(reference)
		if err != nil {
			return nil, err
		}
		content, err := os.ReadFile(absolute)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil {
			exists[reference] = true
			original[reference] = content
			digests[reference] = SHA256(content)
		}
		paths = append(paths, reference)
	}
	sort.Strings(paths)
	status, err = r.GitOutput("status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return nil, policyFailure("SEED_WORKTREE_DIRTY", "repository changed while computing seed")
	}
	written := []string{}
	for _, reference := range paths {
		digest := digests[reference]
		if err := r.AtomicWrite(reference, updates[reference], &digest); err != nil {
			for i := len(written) - 1; i >= 0; i-- {
				prior := written[i]
				if exists[prior] {
					if restoreErr := r.AtomicWrite(prior, original[prior], nil); restoreErr != nil {
						return nil, fmt.Errorf("%w; rollback: %w", err, restoreErr)
					}
				} else {
					absolute, pathErr := r.Path(prior)
					if pathErr != nil {
						return nil, pathErr
					}
					if removeErr := os.Remove(absolute); removeErr != nil && !os.IsNotExist(removeErr) {
						return nil, fmt.Errorf("%w; rollback: %w", err, removeErr)
					}
				}
			}
			return nil, err
		}
		written = append(written, reference)
	}
	return Object{"status": "PASS", "source_pp": "pp.1.01", "target_pp": "pp.1.02", "changed_paths": paths, "stage_instruction": "Stage only the listed semantic/source paths; pre-commit generates PP transition outputs.", "specification_revision_preserved": revision, "acceptance_test_ref": policySeedAcceptancePath}, nil
}
