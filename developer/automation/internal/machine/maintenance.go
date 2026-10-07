package machine

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func (r *Repository) GitOutput(args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", r.Root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %v: %w: %s", args, err, output)
	}
	return strings.TrimSpace(string(output)), nil
}
func (r *Repository) InstallHooks() (Object, error) {
	hook, err := r.Path(".githooks/pre-commit")
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(hook); err != nil || info.IsDir() {
		return nil, Fail("CANONICAL_HOOK_MISSING", ".githooks/pre-commit")
	}
	inside, err := r.GitOutput("rev-parse", "--is-inside-work-tree")
	if err != nil || inside != "true" {
		return nil, Fail("NOT_GIT_WORKTREE", r.Root)
	}
	if _, err := r.GitOutput("config", "--local", "core.hooksPath", ".githooks"); err != nil {
		return nil, err
	}
	actual, err := r.GitOutput("config", "--local", "--get", "core.hooksPath")
	if err != nil || actual != ".githooks" {
		return nil, Fail("HOOKS_PATH_NOT_ACTIVATED", "core.hooksPath did not resolve to .githooks")
	}
	return Object{"status": "PASS", "hooks_path": ".githooks", "pre_commit": hook}, nil
}

type DependencyHit struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Reference string `json:"reference"`
}

func (r *Repository) LegacyDependencyHits() ([]DependencyHit, error) {
	// Current surface selectors are explicit. Audit evidence is deliberately outside them.
	exact := []string{"AGENTS.md", ".ptsip/profiles/main.ptsip.yaml", "developer/planning/0.4.0/WU-02/WU-02.yaml", "developer/planning/0.4.0/WU-02/WU-02-P01.yaml", "releasenote/project-profile/pp.1.01.md", "developer/policy/index.yaml", "src/ptsip/specdata/support-policy-index.yaml", "schemas/ptsip-authorization-transition.schema.json"}
	roots := []string{"developer/automation", "developer/policy", "developer/planning/schemas", "src/ptsip", "schemas"}
	files := map[string]bool{}
	for _, relative := range exact {
		candidate, err := r.Path(relative)
		if err != nil {
			return nil, err
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			files[relative] = true
		}
	}
	for _, relative := range roots {
		directory, err := r.Path(relative)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			continue
		}
		if err := filepath.WalkDir(directory, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "legacy" || entry.Name() == "analysis" || entry.Name() == "approvals" || entry.Name() == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			relative, _ := filepath.Rel(r.Root, name)
			relative = filepath.ToSlash(relative)
			if relative == "src/ptsip/app/github_authority.py" {
				return nil
			}
			if legacyDependencyCurrentSurface(relative) {
				files[relative] = true
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	names := []string{}
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	pattern := regexp.MustCompile(`decisions/[A-Za-z0-9_.*{}<>/\-]+(?:\.[A-Za-z0-9_.*{}<>/\-]+)?|\bADR-[0-9]{4}\b`)
	result := []DependencyHit{}
	for _, name := range names {
		data, err := r.ReadSource(name)
		if err != nil {
			return nil, err
		}
		for line, text := range strings.Split(string(data), "\n") {
			for _, reference := range pattern.FindAllString(text, -1) {
				result = append(result, DependencyHit{name, line + 1, reference})
			}
		}
	}
	return result, nil
}

// Match the original current-surface contract while including native Go
// implementation packages. Frozen PP schemas and test/audit inputs do not become
// operational dependencies merely because the implementation is now recursive.
func legacyDependencyCurrentSurface(ref string) bool {
	name := filepath.Base(ref)
	switch {
	case strings.HasPrefix(ref, "developer/automation/"):
		return !strings.Contains(ref, "/testdata/") && !strings.HasSuffix(ref, "_test.go") && (strings.HasSuffix(ref, ".py") || strings.HasSuffix(ref, ".go"))
	case strings.HasPrefix(ref, "developer/policy/"):
		return strings.HasPrefix(name, "MPD-") && strings.HasSuffix(name, ".yaml") || strings.HasPrefix(ref, "developer/policy/registries/") && strings.HasSuffix(name, ".yaml") || strings.HasPrefix(ref, "developer/policy/schemas/") && strings.HasSuffix(name, ".json")
	case strings.HasPrefix(ref, "developer/planning/schemas/"):
		return strings.HasSuffix(name, ".json")
	case strings.HasPrefix(ref, "src/ptsip/"):
		return strings.HasSuffix(name, ".py") || strings.HasPrefix(ref, "src/ptsip/specdata/") && (strings.HasPrefix(name, "SFP-") && strings.HasSuffix(name, ".yaml") || strings.HasPrefix(name, "ptsip-support-") && (strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".json")))
	case strings.HasPrefix(ref, "schemas/"):
		return strings.HasPrefix(name, "ptsip-support-") && strings.HasSuffix(name, ".json")
	}
	return false
}
func (r *Repository) EvaluateLegacyRemoval() (Object, error) {
	plan, err := r.Read("developer/planning/0.4.0/WU-02/WU-02-P01.yaml")
	if err != nil {
		return nil, err
	}
	migration := Map(Map(plan["migration_stages"])["P01_E_LEGACY_REMOVAL"])
	blockers := []string{}
	hits, err := r.LegacyDependencyHits()
	if err != nil {
		return nil, err
	}
	if len(hits) > 0 {
		blockers = append(blockers, "CURRENT_LEGACY_DEPENDENCY_NONZERO")
	}
	found := []Object{}
	for _, raw := range List(Map(plan["p01_e_execution_plan"])["execution_order"]) {
		if value := Map(raw); value["id"] == "P01_E4_MIGRATION_ONLY_RETIREMENT_AND_GATE_SIMPLIFICATION" {
			found = append(found, value)
		}
	}
	if len(found) != 1 || found[0]["status"] != "COMPLETE" {
		blockers = append(blockers, "P01_E4_VALIDATION_NOT_COMPLETE")
	}
	action := Text(migration["preauthorized_action"])
	if action == "" {
		blockers = append(blockers, "LEGACY_DECISIONS_REMOVAL_NOT_PREAUTHORIZED")
	}
	if migration["confirmation_required"] != false {
		blockers = append(blockers, "LEGACY_DECISIONS_REMOVAL_CONFIRMATION_POLICY_INVALID")
	}
	sort.Strings(blockers)
	if len(blockers) > 0 {
		return Object{"state": "HOLD_NOT_AUTHORIZED", "action": nil, "blockers": blockers, "confirmation_required": false}, nil
	}
	return Object{"state": "AUTHORIZED", "action": action, "blockers": []string{}, "confirmation_required": false}, nil
}

func (r *Repository) WU02Lanes(lane, execution string) ([]string, error) {
	parentPath := "developer/planning/0.4.0/WU-02/WU-02.yaml"
	parent, err := r.Read(parentPath)
	if err != nil {
		return nil, err
	}
	declared := map[string]Object{}
	for _, raw := range List(Map(parent["parallel_work_lanes"])["canonical_lane_plans"]) {
		entry := Map(raw)
		declared[Text(entry["id"])] = entry
	}
	required := []string{"record", "branch_baseline", "responsibility", "owned_implementation_surface", "session_protocol", "execution_state", "decisions_made", "unresolved_questions", "verification_state", "integration_handoff"}
	expectedReads := []string{parentPath}
	for _, name := range []string{"S1", "S2", "S3"} {
		expectedReads = append(expectedReads, "developer/planning/0.4.0/WU-02/WU-02-"+name+".yaml")
	}
	errors := []string{}
	ownership := map[string]string{}
	for _, name := range []string{"S1", "S2", "S3"} {
		id := "WU-02-" + name
		path := expectedReads[1+int(name[1]-'1')]
		entry := declared[id]
		if entry == nil || entry["path"] != path || entry["branch"] != "dev/0.3.8" {
			errors = append(errors, parentPath+": invalid canonical lane "+id)
		}
		payload, err := r.Read(path)
		if err != nil {
			return nil, err
		}
		missing := false
		for _, section := range required {
			if _, exists := payload[section]; !exists {
				errors = append(errors, path+": missing required section "+section)
				missing = true
			}
		}
		if missing {
			continue
		}
		record := Map(payload["record"])
		if record["id"] != id || record["branch"] != "dev/0.3.8" || record["base_branch"] != "dev/0.3.8" {
			errors = append(errors, path+": record identity/branch mismatch")
		}
		baseline := Map(payload["branch_baseline"])
		if baseline["control_plane_branch"] != "dev/0.3.8" || baseline["canonical_plan_source"] != "CENTRAL_CONTROL_PLANE" {
			errors = append(errors, path+": branch baseline mismatch")
		}
		surface := Map(payload["owned_implementation_surface"])
		allowed := Strings(surface["allowed_paths"])
		forbidden := Strings(surface["integration_only_files_forbidden"])
		if !Has(allowed, path) || Has(forbidden, path) {
			errors = append(errors, path+": invalid own lane path scope")
		}
		for _, own := range allowed {
			if prior := ownership[own]; prior != "" && prior != name {
				errors = append(errors, own+": duplicate lane ownership")
			}
			ownership[own] = name
		}
		reads := Strings(Map(payload["session_protocol"])["required_read_before_work"])
		for _, requiredRead := range expectedReads {
			if !Has(reads, requiredRead) {
				errors = append(errors, path+": required read missing "+requiredRead)
			}
		}
		for section, keys := range map[string][]string{"execution_state": {"completed", "remaining", "blockers"}, "verification_state": {"required", "passed", "failed"}, "integration_handoff": {"requests_to_other_lanes", "integration_notes"}} {
			for _, key := range keys {
				if _, ok := Map(payload[section])[key].([]any); !ok {
					errors = append(errors, path+": "+section+"."+key+" must be list")
				}
			}
		}
	}
	if lane == "" {
		return errors, nil
	}
	lane = strings.ToUpper(lane)
	if !Has([]string{"S1", "S2", "S3"}, lane) {
		return []string{"Unknown lane " + lane}, nil
	}
	if execution == "" {
		execution, err = r.GitOutput("branch", "--show-current")
		if err != nil {
			return nil, err
		}
	}
	exists := func(ref string) bool { _, err := r.GitOutput("rev-parse", "--verify", ref); return err == nil }
	canonical := exists("origin/dev/0.3.8") || exists("dev/0.3.8")
	legacy := exists("origin/dev/0.4.0") || exists("dev/0.4.0")
	if execution != "dev/0.3.8" && !(execution == "dev/0.4.0" && legacy && !canonical) {
		return append(errors, "Current branch does not match lane control branch"), nil
	}
	control := ""
	for _, ref := range []string{"origin/dev/0.3.8", "dev/0.3.8", "origin/dev/0.4.0", "dev/0.4.0"} {
		if exists(ref) {
			control = ref
			break
		}
	}
	if control == "" {
		return nil, Fail("CONTROL_REF_MISSING", "no canonical or registered legacy control ref")
	}
	changed, err := r.GitOutput("diff", "--name-only", control+"...HEAD")
	if err != nil {
		return nil, err
	}
	payload, err := r.Read("developer/planning/0.4.0/WU-02/WU-02-" + lane + ".yaml")
	if err != nil {
		return nil, err
	}
	surface := Map(payload["owned_implementation_surface"])
	for _, path := range strings.Split(changed, "\n") {
		if path != "" && (Has(Strings(surface["integration_only_files_forbidden"]), path) || !Has(Strings(surface["allowed_paths"]), path)) {
			errors = append(errors, lane+": path outside lane ownership: "+path)
		}
	}
	return errors, nil
}
func init() {
	RegisterOperations("developer-setup", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		if command == "install-hooks" {
			return r.InstallHooks()
		}
		return nil, Fail("UNREGISTERED_SETUP_OPERATION", command)
	})
	RegisterOperations("dependency-gate", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		hits, err := r.LegacyDependencyHits()
		if err != nil {
			return nil, err
		}
		status := "PASS"
		if len(hits) > 0 {
			status = "FAIL"
		}
		return Object{"status": status, "hits": hits}, nil
	})
	RegisterOperations("current-dependency-gate", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		return r.DispatchOperation("dependency-gate", "verify", opts, args)
	})
	RegisterOperations("transition", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		return r.EvaluateLegacyRemoval()
	})
	RegisterOperations("wu02", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		if command == "control-context" {
			branch := strings.TrimSpace(opts["--execution-branch"])
			errors := []string{}
			if branch != "" && branch != "dev/0.3.8" {
				errors = append(errors, "Execution branch does not match control branch dev/0.3.8")
			}
			status := "PASS"
			if len(errors) > 0 {
				status = "FAIL"
			}
			return Object{"status": status, "errors": errors, "control_branch": "dev/0.3.8", "validation_performed": branch != ""}, nil
		}
		errors, err := r.WU02Lanes(opts["--lane"], opts["--execution-branch"])
		if err != nil {
			return nil, err
		}
		status := "PASS"
		if len(errors) > 0 {
			status = "FAIL"
		}
		return Object{"status": status, "errors": errors}, nil
	})
}
